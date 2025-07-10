package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	swarmv1alpha1 "github.com/claudeflow/swarm-operator/api/v1alpha1"
)

// SwarmAgentReconciler reconciles a SwarmAgent object
type SwarmAgentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmagents/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *SwarmAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("swarmagent", req.NamespacedName)

	// Fetch the SwarmAgent instance
	agent := &swarmv1alpha1.SwarmAgent{}
	err := r.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get parent cluster
	cluster := &swarmv1alpha1.SwarmCluster{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      agent.Spec.ClusterRef,
		Namespace: agent.Namespace,
	}, cluster)
	if err != nil {
		log.Error(err, "Failed to get parent cluster")
		return ctrl.Result{}, err
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(agent, "swarmagent.claudeflow.io/finalizer") {
		controllerutil.AddFinalizer(agent, "swarmagent.claudeflow.io/finalizer")
		if err := r.Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle deletion
	if !agent.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, agent)
	}

	// Reconcile agent deployment
	if err := r.reconcileDeployment(ctx, agent, cluster); err != nil {
		log.Error(err, "Failed to reconcile deployment")
		return ctrl.Result{}, err
	}

	// Update agent status
	if err := r.updateAgentStatus(ctx, agent); err != nil {
		log.Error(err, "Failed to update agent status")
		return ctrl.Result{}, err
	}

	// Check if agent needs to connect to hive-mind
	if cluster.Spec.HiveMind.Enabled && !agent.Status.HiveMindConnected {
		if err := r.connectToHiveMind(ctx, agent, cluster); err != nil {
			log.Error(err, "Failed to connect to hive-mind")
			// Don't return error, just log and retry later
		}
	}

	// Requeue periodically for status updates
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *SwarmAgentReconciler) reconcileDeployment(ctx context.Context, agent *swarmv1alpha1.SwarmAgent, cluster *swarmv1alpha1.SwarmCluster) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agent.Name,
			Namespace: agent.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		deploy.Labels = map[string]string{
			"swarm-cluster": agent.Spec.ClusterRef,
			"swarm-agent":   agent.Name,
			"agent-type":    string(agent.Spec.Type),
			"component":     "agent",
		}

		replicas := int32(1)
		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"swarm-agent": agent.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"swarm-cluster": agent.Spec.ClusterRef,
						"swarm-agent":   agent.Name,
						"agent-type":    string(agent.Spec.Type),
						"component":     "agent",
					},
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "9090",
						"prometheus.io/path":   "/metrics",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: fmt.Sprintf("%s-agent", cluster.Name),
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: getOrDefault(agent.Spec.Image, cluster.Spec.AgentTemplate.Image),
							Env: r.buildAgentEnv(agent, cluster),
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 9090,
								},
								{
									Name:          "grpc",
									ContainerPort: 50051,
								},
							},
							Resources: r.buildResources(agent.Spec.Resources),
							VolumeMounts: r.buildVolumeMounts(agent, cluster),
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/ready",
										Port: intstr.FromInt(8080),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       5,
							},
						},
					},
					Volumes: r.buildVolumes(agent, cluster),
					NodeSelector: cluster.Spec.AgentTemplate.NodeSelector,
					Tolerations: r.buildTolerations(cluster.Spec.AgentTemplate.Tolerations),
					Affinity: r.buildAffinity(agent, cluster),
				},
			},
		}

		// Set security context if specified
		if cluster.Spec.AgentTemplate.SecurityContext != nil {
			deploy.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: &cluster.Spec.AgentTemplate.SecurityContext.RunAsNonRoot,
				RunAsUser:    &cluster.Spec.AgentTemplate.SecurityContext.RunAsUser,
				FSGroup:      &cluster.Spec.AgentTemplate.SecurityContext.FSGroup,
			}
		}

		return controllerutil.SetControllerReference(agent, deploy, r.Scheme)
	})

	return err
}

func (r *SwarmAgentReconciler) buildAgentEnv(agent *swarmv1alpha1.SwarmAgent, cluster *swarmv1alpha1.SwarmCluster) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "AGENT_NAME",
			Value: agent.Name,
		},
		{
			Name:  "AGENT_TYPE",
			Value: string(agent.Spec.Type),
		},
		{
			Name:  "CLUSTER_NAME",
			Value: cluster.Name,
		},
		{
			Name:  "COGNITIVE_PATTERN",
			Value: string(agent.Spec.CognitivePattern),
		},
		{
			Name:  "MAX_CONCURRENT_TASKS",
			Value: fmt.Sprintf("%d", agent.Spec.MaxConcurrentTasks),
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
	}

	// Add capabilities as environment variable
	if len(agent.Spec.Capabilities) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "AGENT_CAPABILITIES",
			Value: strings.Join(agent.Spec.Capabilities, ","),
		})
	}

	// Add neural models if assigned
	if len(agent.Spec.NeuralModels) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "NEURAL_MODELS",
			Value: strings.Join(agent.Spec.NeuralModels, ","),
		})
	}

	// Add hive-mind configuration
	if cluster.Spec.HiveMind.Enabled {
		env = append(env, corev1.EnvVar{
			Name:  "HIVEMIND_ENABLED",
			Value: "true",
		}, corev1.EnvVar{
			Name:  "HIVEMIND_ENDPOINT",
			Value: fmt.Sprintf("%s-hivemind:8080", cluster.Name),
		}, corev1.EnvVar{
			Name:  "HIVEMIND_ROLE",
			Value: getOrDefault(agent.Spec.HiveMindRole, "worker"),
		})
	}

	// Add memory backend configuration
	if cluster.Spec.Memory.Type != "" {
		env = append(env, corev1.EnvVar{
			Name:  "MEMORY_BACKEND",
			Value: cluster.Spec.Memory.Type,
		}, corev1.EnvVar{
			Name:  "MEMORY_ENDPOINT",
			Value: fmt.Sprintf("%s-%s:6379", cluster.Name, cluster.Spec.Memory.Type),
		})
	}

	// Add custom environment variables
	for _, e := range agent.Spec.Environment {
		if e.ValueFrom != nil {
			env = append(env, corev1.EnvVar{
				Name:      e.Name,
				ValueFrom: r.buildEnvVarSource(e.ValueFrom),
			})
		} else {
			env = append(env, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}
	}

	return env
}

func (r *SwarmAgentReconciler) buildEnvVarSource(source *swarmv1alpha1.EnvVarSource) *corev1.EnvVarSource {
	result := &corev1.EnvVarSource{}
	
	if source.SecretKeyRef != nil {
		result.SecretKeyRef = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: source.SecretKeyRef.Name,
			},
			Key: source.SecretKeyRef.Key,
		}
	}
	
	if source.ConfigMapKeyRef != nil {
		result.ConfigMapKeyRef = &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: source.ConfigMapKeyRef.Name,
			},
			Key: source.ConfigMapKeyRef.Key,
		}
	}
	
	return result
}

func (r *SwarmAgentReconciler) buildResources(resources swarmv1alpha1.ResourceRequirements) corev1.ResourceRequirements {
	req := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if resources.CPU != "" {
		req.Requests[corev1.ResourceCPU] = resource.MustParse(resources.CPU)
		req.Limits[corev1.ResourceCPU] = resource.MustParse(resources.CPU).DeepCopy()
		req.Limits[corev1.ResourceCPU].Add(resource.MustParse("500m"))
	}

	if resources.Memory != "" {
		req.Requests[corev1.ResourceMemory] = resource.MustParse(resources.Memory)
		req.Limits[corev1.ResourceMemory] = resource.MustParse(resources.Memory).DeepCopy()
		req.Limits[corev1.ResourceMemory].Add(resource.MustParse("512Mi"))
	}

	if resources.GPU != "" {
		req.Requests["nvidia.com/gpu"] = resource.MustParse(resources.GPU)
		req.Limits["nvidia.com/gpu"] = resource.MustParse(resources.GPU)
	}

	return req
}

func (r *SwarmAgentReconciler) buildVolumeMounts(agent *swarmv1alpha1.SwarmAgent, cluster *swarmv1alpha1.SwarmCluster) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      "agent-config",
			MountPath: "/config",
		},
		{
			Name:      "workspace",
			MountPath: "/workspace",
		},
	}

	// Add neural model mount if needed
	if len(agent.Spec.NeuralModels) > 0 && cluster.Spec.Neural.Enabled {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "neural-models",
			MountPath: "/models",
			ReadOnly:  true,
		})
	}

	return mounts
}

func (r *SwarmAgentReconciler) buildVolumes(agent *swarmv1alpha1.SwarmAgent, cluster *swarmv1alpha1.SwarmCluster) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: "agent-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-agent-config", cluster.Name),
					},
				},
			},
		},
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	// Add neural model volume if needed
	if len(agent.Spec.NeuralModels) > 0 && cluster.Spec.Neural.Enabled {
		volumes = append(volumes, corev1.Volume{
			Name: "neural-models",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-neural-models", cluster.Name),
				},
			},
		})
	}

	return volumes
}

func (r *SwarmAgentReconciler) buildTolerations(tolerations []swarmv1alpha1.Toleration) []corev1.Toleration {
	result := make([]corev1.Toleration, len(tolerations))
	for i, t := range tolerations {
		result[i] = corev1.Toleration{
			Key:      t.Key,
			Operator: corev1.TolerationOperator(t.Operator),
			Value:    t.Value,
			Effect:   corev1.TaintEffect(t.Effect),
		}
	}
	return result
}

func (r *SwarmAgentReconciler) buildAffinity(agent *swarmv1alpha1.SwarmAgent, cluster *swarmv1alpha1.SwarmCluster) *corev1.Affinity {
	affinity := &corev1.Affinity{}

	// Add pod anti-affinity to spread agents across nodes
	affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"agent-type": string(agent.Spec.Type),
						},
					},
					TopologyKey: "kubernetes.io/hostname",
				},
			},
		},
	}

	// Add custom affinity if specified
	if cluster.Spec.AgentTemplate.Affinity != nil {
		// Merge custom affinity rules
		if cluster.Spec.AgentTemplate.Affinity.NodeAffinity != nil {
			affinity.NodeAffinity = r.buildNodeAffinity(cluster.Spec.AgentTemplate.Affinity.NodeAffinity)
		}
	}

	return affinity
}

func (r *SwarmAgentReconciler) buildNodeAffinity(nodeAffinity *swarmv1alpha1.NodeAffinity) *corev1.NodeAffinity {
	result := &corev1.NodeAffinity{}
	
	if len(nodeAffinity.RequiredDuringScheduling) > 0 {
		result.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: make([]corev1.NodeSelectorTerm, len(nodeAffinity.RequiredDuringScheduling)),
		}
		
		for i, term := range nodeAffinity.RequiredDuringScheduling {
			result.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i] = corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{},
			}
			
			for k, v := range term.MatchLabels {
				result.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i].MatchExpressions = append(
					result.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[i].MatchExpressions,
					corev1.NodeSelectorRequirement{
						Key:      k,
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{v},
					},
				)
			}
		}
	}
	
	return result
}

func (r *SwarmAgentReconciler) updateAgentStatus(ctx context.Context, agent *swarmv1alpha1.SwarmAgent) error {
	// Get deployment
	deploy := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      agent.Name,
		Namespace: agent.Namespace,
	}, deploy)
	if err != nil {
		return err
	}

	// Get pod
	podList := &corev1.PodList{}
	err = r.List(ctx, podList,
		client.InNamespace(agent.Namespace),
		client.MatchingLabels{"swarm-agent": agent.Name})
	if err != nil {
		return err
	}

	// Update basic status
	if len(podList.Items) > 0 {
		pod := podList.Items[0]
		agent.Status.PodName = pod.Name
		agent.Status.NodeName = pod.Spec.NodeName
		
		// Determine agent status based on pod phase
		switch pod.Status.Phase {
		case corev1.PodRunning:
			if isPodReady(pod) {
				if len(agent.Status.AssignedTasks) > 0 {
					agent.Status.Status = swarmv1alpha1.AgentStatusBusy
				} else {
					agent.Status.Status = swarmv1alpha1.AgentStatusIdle
				}
			} else {
				agent.Status.Status = swarmv1alpha1.AgentStatusInitializing
			}
		case corev1.PodPending:
			agent.Status.Status = swarmv1alpha1.AgentStatusPending
		case corev1.PodFailed:
			agent.Status.Status = swarmv1alpha1.AgentStatusError
		default:
			agent.Status.Status = swarmv1alpha1.AgentStatusPending
		}
		
		// Update resource utilization (would need metrics API in real implementation)
		agent.Status.Utilization = calculateUtilization(agent.Status.AssignedTasks, agent.Spec.MaxConcurrentTasks)
		
		// Set start time
		if agent.Status.StartTime == nil {
			agent.Status.StartTime = pod.Status.StartTime
		}
	} else {
		agent.Status.Status = swarmv1alpha1.AgentStatusPending
	}

	// Update performance metrics (simplified)
	if agent.Status.CompletedTasks > 0 {
		agent.Status.Performance.SuccessRate = float64(agent.Status.CompletedTasks) / 
			float64(agent.Status.CompletedTasks + agent.Status.FailedTasks) * 100
	}

	// Update conditions
	agent.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "AgentReady",
			Message:            "Agent is ready to process tasks",
		},
	}

	return r.Status().Update(ctx, agent)
}

func (r *SwarmAgentReconciler) connectToHiveMind(ctx context.Context, agent *swarmv1alpha1.SwarmAgent, cluster *swarmv1alpha1.SwarmCluster) error {
	// In a real implementation, this would establish connection to hive-mind
	// For now, just mark as connected
	agent.Status.HiveMindConnected = true
	return nil
}

func (r *SwarmAgentReconciler) handleDeletion(ctx context.Context, agent *swarmv1alpha1.SwarmAgent) (ctrl.Result, error) {
	// Cleanup logic
	controllerutil.RemoveFinalizer(agent, "swarmagent.claudeflow.io/finalizer")
	if err := r.Update(ctx, agent); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// Helper functions

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func calculateUtilization(assignedTasks []string, maxTasks int32) int32 {
	if maxTasks == 0 {
		return 0
	}
	return int32(float32(len(assignedTasks)) / float32(maxTasks) * 100)
}

func (r *SwarmAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swarmv1alpha1.SwarmAgent{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}