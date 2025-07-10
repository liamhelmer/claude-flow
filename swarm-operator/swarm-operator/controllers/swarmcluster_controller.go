package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
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

// SwarmClusterReconciler reconciles a SwarmCluster object
type SwarmClusterReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	SwarmNamespace    string
	HiveMindNamespace string
}

// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmmemories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services;configmaps;secrets;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

func (r *SwarmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("swarmcluster", req.NamespacedName)

	// Fetch the SwarmCluster instance
	cluster := &swarmv1alpha1.SwarmCluster{}
	err := r.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add finalizer for cleanup
	if !controllerutil.ContainsFinalizer(cluster, "swarmcluster.claudeflow.io/finalizer") {
		controllerutil.AddFinalizer(cluster, "swarmcluster.claudeflow.io/finalizer")
		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle deletion
	if !cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, cluster)
	}

	// Update status phase
	if cluster.Status.Phase == "" {
		cluster.Status.Phase = "Initializing"
		if err := r.Status().Update(ctx, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile components in order
	if err := r.reconcileHiveMind(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile hive-mind")
		return ctrl.Result{}, err
	}

	if err := r.reconcileMemoryBackend(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile memory backend")
		return ctrl.Result{}, err
	}

	if err := r.reconcileAgents(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile agents")
		return ctrl.Result{}, err
	}

	if err := r.reconcileAutoscaling(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile autoscaling")
		return ctrl.Result{}, err
	}

	if err := r.reconcileMonitoring(ctx, cluster); err != nil {
		log.Error(err, "Failed to reconcile monitoring")
		return ctrl.Result{}, err
	}

	// Update cluster status
	if err := r.updateClusterStatus(ctx, cluster); err != nil {
		log.Error(err, "Failed to update cluster status")
		return ctrl.Result{}, err
	}

	// Requeue periodically for status updates
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *SwarmClusterReconciler) reconcileHiveMind(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	if !cluster.Spec.HiveMind.Enabled {
		return nil
	}

	// Determine namespace
	namespace := r.getNamespaceForComponent(cluster, "hivemind")
	
	// Create hive-mind StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-hivemind", cluster.Name),
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Labels = map[string]string{
			"swarm-cluster": cluster.Name,
			"component":     "hivemind",
		}

		replicas := int32(3) // Default to 3 replicas for HA
		sts.Spec = appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"swarm-cluster": cluster.Name,
					"component":     "hivemind",
				},
			},
			ServiceName: fmt.Sprintf("%s-hivemind", cluster.Name),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"swarm-cluster": cluster.Name,
						"component":     "hivemind",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "hivemind",
							Image: getHiveMindImage(cluster),
							Ports: []corev1.ContainerPort{
								{
									Name:          "sqlite",
									ContainerPort: 3306,
								},
								{
									Name:          "sync",
									ContainerPort: 8080,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CLUSTER_NAME",
									Value: cluster.Name,
								},
								{
									Name:  "SYNC_INTERVAL",
									Value: cluster.Spec.HiveMind.SyncInterval,
								},
								{
									Name:  "BACKUP_ENABLED",
									Value: fmt.Sprintf("%t", cluster.Spec.HiveMind.BackupEnabled),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(
									getOrDefault(cluster.Spec.HiveMind.DatabaseSize, "10Gi"),
								),
							},
						},
					},
				},
			},
		}

		return controllerutil.SetControllerReference(cluster, sts, r.Scheme)
	})

	if err != nil {
		return err
	}

	// Create hive-mind service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-hivemind", cluster.Name),
			Namespace: cluster.Namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = map[string]string{
			"swarm-cluster": cluster.Name,
			"component":     "hivemind",
		}

		svc.Spec = corev1.ServiceSpec{
			Selector: map[string]string{
				"swarm-cluster": cluster.Name,
				"component":     "hivemind",
			},
			ClusterIP: corev1.ClusterIPNone, // Headless service for StatefulSet
			Ports: []corev1.ServicePort{
				{
					Name: "sqlite",
					Port: 3306,
				},
				{
					Name: "sync",
					Port: 8080,
				},
			},
		}

		return controllerutil.SetControllerReference(cluster, svc, r.Scheme)
	})

	return err
}

func (r *SwarmClusterReconciler) reconcileMemoryBackend(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	if cluster.Spec.Memory.Type == "" {
		cluster.Spec.Memory.Type = "redis" // Default to Redis
	}

	switch cluster.Spec.Memory.Type {
	case "redis":
		return r.deployRedis(ctx, cluster)
	case "hazelcast":
		return r.deployHazelcast(ctx, cluster)
	case "etcd":
		return r.deployEtcd(ctx, cluster)
	default:
		return fmt.Errorf("unsupported memory backend: %s", cluster.Spec.Memory.Type)
	}
}

func (r *SwarmClusterReconciler) reconcileAgents(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	// Get agent configuration based on topology
	agentConfigs := getTopologyAgentConfig(cluster.Spec.Topology)

	for agentType, count := range agentConfigs {
		for i := 0; i < count; i++ {
			// Determine namespace for agent
			namespace := r.getNamespaceForComponent(cluster, "swarm")
			
			agent := &swarmv1alpha1.SwarmAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-%s-%d", cluster.Name, agentType, i),
					Namespace: namespace,
				},
			}

			_, err := controllerutil.CreateOrUpdate(ctx, r.Client, agent, func() error {
				agent.Labels = map[string]string{
					"swarm-cluster": cluster.Name,
					"agent-type":    string(agentType),
				}

				agent.Spec = swarmv1alpha1.SwarmAgentSpec{
					Type:       agentType,
					ClusterRef: cluster.Name,
					CognitivePattern: getCognitivePattern(agentType),
					Priority:   getAgentPriority(agentType),
					MaxConcurrentTasks: getMaxConcurrentTasks(agentType),
					Resources: getAgentResources(cluster, agentType),
					Image: getOrDefault(cluster.Spec.AgentTemplate.Image, "claudeflow/swarm-executor:2.0.0"),
				}

				// Set capabilities based on agent type
				agent.Spec.Capabilities = getAgentCapabilities(agentType)

				// Set neural models if enabled
				if cluster.Spec.Neural.Enabled {
					agent.Spec.NeuralModels = getNeuralModelsForAgent(agentType)
				}

				return controllerutil.SetControllerReference(cluster, agent, r.Scheme)
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *SwarmClusterReconciler) reconcileAutoscaling(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	if !cluster.Spec.Autoscaling.Enabled {
		return nil
	}

	// Create HPA for each agent type
	agentTypes := getAgentTypesForTopology(cluster.Spec.Topology)

	for _, agentType := range agentTypes {
		hpa := &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-hpa", cluster.Name, agentType),
				Namespace: cluster.Namespace,
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, hpa, func() error {
			hpa.Labels = map[string]string{
				"swarm-cluster": cluster.Name,
				"agent-type":    string(agentType),
			}

			// Calculate min/max replicas based on topology ratios
			minReplicas := int32(1)
			maxReplicas := int32(10)
			if ratio, ok := cluster.Spec.Autoscaling.TopologyRatios[string(agentType)]; ok {
				maxReplicas = ratio * cluster.Spec.Autoscaling.MaxAgents / 100
				if maxReplicas < 1 {
					maxReplicas = 1
				}
			}

			targetCPU := cluster.Spec.Autoscaling.TargetUtilization
			if targetCPU == 0 {
				targetCPU = 80
			}

			hpa.Spec = autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       fmt.Sprintf("%s-%s", cluster.Name, agentType),
				},
				MinReplicas: &minReplicas,
				MaxReplicas: maxReplicas,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &targetCPU,
							},
						},
					},
				},
			}

			// Add custom metrics if specified
			for _, metric := range cluster.Spec.Autoscaling.Metrics {
				if metric.Type == "custom" {
					hpa.Spec.Metrics = append(hpa.Spec.Metrics, autoscalingv2.MetricSpec{
						Type: autoscalingv2.PodsMetricSourceType,
						Pods: &autoscalingv2.PodsMetricSource{
							Metric: autoscalingv2.MetricIdentifier{
								Name: metric.Name,
							},
							Target: autoscalingv2.MetricTarget{
								Type:         autoscalingv2.AverageValueMetricType,
								AverageValue: resource.MustParse(metric.Target),
							},
						},
					})
				}
			}

			// Set behavior for stabilization
			if cluster.Spec.Autoscaling.StabilizationWindow != "" {
				windowSeconds := int32(300) // Default 5 minutes
				hpa.Spec.Behavior = &autoscalingv2.HorizontalPodAutoscalerBehavior{
					ScaleDown: &autoscalingv2.HPAScalingRules{
						StabilizationWindowSeconds: &windowSeconds,
					},
				}
			}

			return controllerutil.SetControllerReference(cluster, hpa, r.Scheme)
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *SwarmClusterReconciler) reconcileMonitoring(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	if !cluster.Spec.Monitoring.Enabled {
		return nil
	}

	// Create ConfigMap for Prometheus scrape config
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-metrics-config", cluster.Name),
			Namespace: cluster.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = map[string]string{
			"swarm-cluster": cluster.Name,
			"component":     "monitoring",
		}

		metricsPort := cluster.Spec.Monitoring.MetricsPort
		if metricsPort == 0 {
			metricsPort = 9090
		}

		cm.Data = map[string]string{
			"prometheus.yml": fmt.Sprintf(`
global:
  scrape_interval: 15s

scrape_configs:
- job_name: 'swarm-agents'
  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - %s
  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_swarm_cluster]
    regex: %s
    action: keep
  - source_labels: [__meta_kubernetes_pod_label_component]
    regex: agent
    action: keep
  - source_labels: [__address__]
    regex: ([^:]+)(?::\d+)?
    replacement: $1:%d
    target_label: __address__
`, cluster.Namespace, cluster.Name, metricsPort),
		}

		return controllerutil.SetControllerReference(cluster, cm, r.Scheme)
	})

	return err
}

func (r *SwarmClusterReconciler) updateClusterStatus(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	// Count agents
	agentList := &swarmv1alpha1.SwarmAgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(cluster.Namespace),
		client.MatchingLabels{"swarm-cluster": cluster.Name}); err != nil {
		return err
	}

	readyAgents := int32(0)
	agentTypes := make(map[string]int32)
	for _, agent := range agentList.Items {
		if agent.Status.Status == swarmv1alpha1.AgentStatusReady ||
			agent.Status.Status == swarmv1alpha1.AgentStatusIdle {
			readyAgents++
		}
		agentTypes[string(agent.Spec.Type)]++
	}

	// Count tasks
	taskList := &swarmv1alpha1.SwarmTaskList{}
	if err := r.List(ctx, taskList, client.InNamespace(cluster.Namespace),
		client.MatchingLabels{"swarm-cluster": cluster.Name}); err != nil {
		return err
	}

	activeTasks := int32(0)
	completedTasks := int32(0)
	for _, task := range taskList.Items {
		switch task.Status.Phase {
		case "Running", "Pending":
			activeTasks++
		case "Completed":
			completedTasks++
		}
	}

	// Update hive-mind status
	if cluster.Spec.HiveMind.Enabled {
		sts := &appsv1.StatefulSet{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      fmt.Sprintf("%s-hivemind", cluster.Name),
			Namespace: cluster.Namespace,
		}, sts)
		if err == nil {
			cluster.Status.HiveMindStatus.Connected = sts.Status.ReadyReplicas
			cluster.Status.HiveMindStatus.SyncStatus = "Active"
			cluster.Status.HiveMindStatus.LastSyncTime = &metav1.Time{Time: time.Now()}
		}
	}

	// Update status
	cluster.Status.ReadyAgents = readyAgents
	cluster.Status.TotalAgents = int32(len(agentList.Items))
	cluster.Status.AgentTypes = agentTypes
	cluster.Status.ActiveTasks = activeTasks
	cluster.Status.CompletedTasks = completedTasks
	cluster.Status.ObservedGeneration = cluster.Generation

	// Set phase
	if readyAgents == 0 {
		cluster.Status.Phase = "Pending"
	} else if readyAgents < cluster.Status.TotalAgents {
		cluster.Status.Phase = "Scaling"
	} else {
		cluster.Status.Phase = "Ready"
	}

	return r.Status().Update(ctx, cluster)
}

func (r *SwarmClusterReconciler) handleDeletion(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	// Cleanup logic here
	// Remove finalizer
	controllerutil.RemoveFinalizer(cluster, "swarmcluster.claudeflow.io/finalizer")
	if err := r.Update(ctx, cluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// Helper functions

func getHiveMindImage(cluster *swarmv1alpha1.SwarmCluster) string {
	return "claudeflow/hivemind:2.0.0"
}

func getOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getTopologyAgentConfig(topology swarmv1alpha1.SwarmTopology) map[swarmv1alpha1.AgentType]int {
	switch topology {
	case swarmv1alpha1.TopologyHierarchical:
		return map[swarmv1alpha1.AgentType]int{
			swarmv1alpha1.AgentTypeCoordinator: 1,
			swarmv1alpha1.AgentTypeResearcher:  2,
			swarmv1alpha1.AgentTypeCoder:       2,
			swarmv1alpha1.AgentTypeAnalyst:     1,
			swarmv1alpha1.AgentTypeTester:      1,
		}
	case swarmv1alpha1.TopologyMesh:
		return map[swarmv1alpha1.AgentType]int{
			swarmv1alpha1.AgentTypeCoordinator: 2,
			swarmv1alpha1.AgentTypeResearcher:  2,
			swarmv1alpha1.AgentTypeCoder:       2,
			swarmv1alpha1.AgentTypeSpecialist:  2,
		}
	case swarmv1alpha1.TopologyRing:
		return map[swarmv1alpha1.AgentType]int{
			swarmv1alpha1.AgentTypeCoordinator: 1,
			swarmv1alpha1.AgentTypeCoder:       3,
			swarmv1alpha1.AgentTypeReviewer:    2,
		}
	case swarmv1alpha1.TopologyStar:
		return map[swarmv1alpha1.AgentType]int{
			swarmv1alpha1.AgentTypeCoordinator: 1,
			swarmv1alpha1.AgentTypeSpecialist:  4,
		}
	default:
		return map[swarmv1alpha1.AgentType]int{
			swarmv1alpha1.AgentTypeCoordinator: 1,
			swarmv1alpha1.AgentTypeCoder:       2,
		}
	}
}

func getCognitivePattern(agentType swarmv1alpha1.AgentType) swarmv1alpha1.CognitivePattern {
	switch agentType {
	case swarmv1alpha1.AgentTypeResearcher:
		return swarmv1alpha1.PatternDivergent
	case swarmv1alpha1.AgentTypeAnalyst:
		return swarmv1alpha1.PatternConvergent
	case swarmv1alpha1.AgentTypeArchitect:
		return swarmv1alpha1.PatternSystems
	case swarmv1alpha1.AgentTypeOptimizer:
		return swarmv1alpha1.PatternCritical
	default:
		return swarmv1alpha1.PatternAdaptive
	}
}

func getAgentPriority(agentType swarmv1alpha1.AgentType) int32 {
	switch agentType {
	case swarmv1alpha1.AgentTypeCoordinator:
		return 100
	case swarmv1alpha1.AgentTypeArchitect:
		return 90
	case swarmv1alpha1.AgentTypeAnalyst:
		return 80
	default:
		return 50
	}
}

func getMaxConcurrentTasks(agentType swarmv1alpha1.AgentType) int32 {
	switch agentType {
	case swarmv1alpha1.AgentTypeCoordinator:
		return 10
	case swarmv1alpha1.AgentTypeCoder:
		return 3
	case swarmv1alpha1.AgentTypeAnalyst:
		return 5
	default:
		return 2
	}
}

func getAgentResources(cluster *swarmv1alpha1.SwarmCluster, agentType swarmv1alpha1.AgentType) swarmv1alpha1.ResourceRequirements {
	// Default resources
	resources := swarmv1alpha1.ResourceRequirements{
		CPU:    "200m",
		Memory: "512Mi",
	}

	// Override with cluster defaults
	if cluster.Spec.AgentTemplate.Resources.CPU != "" {
		resources.CPU = cluster.Spec.AgentTemplate.Resources.CPU
	}
	if cluster.Spec.AgentTemplate.Resources.Memory != "" {
		resources.Memory = cluster.Spec.AgentTemplate.Resources.Memory
	}

	// Adjust based on agent type
	switch agentType {
	case swarmv1alpha1.AgentTypeCoordinator:
		resources.CPU = "500m"
		resources.Memory = "1Gi"
	case swarmv1alpha1.AgentTypeAnalyst:
		resources.CPU = "1000m"
		resources.Memory = "2Gi"
	case swarmv1alpha1.AgentTypeOptimizer:
		resources.CPU = "2000m"
		resources.Memory = "4Gi"
	}

	return resources
}

func getAgentCapabilities(agentType swarmv1alpha1.AgentType) []string {
	switch agentType {
	case swarmv1alpha1.AgentTypeResearcher:
		return []string{"search", "analyze", "summarize", "cite"}
	case swarmv1alpha1.AgentTypeCoder:
		return []string{"code", "test", "debug", "refactor"}
	case swarmv1alpha1.AgentTypeAnalyst:
		return []string{"analyze", "optimize", "benchmark", "profile"}
	case swarmv1alpha1.AgentTypeArchitect:
		return []string{"design", "plan", "structure", "integrate"}
	default:
		return []string{"general"}
	}
}

func getNeuralModelsForAgent(agentType swarmv1alpha1.AgentType) []string {
	switch agentType {
	case swarmv1alpha1.AgentTypeOptimizer:
		return []string{"optimization-v1", "performance-v1"}
	case swarmv1alpha1.AgentTypeAnalyst:
		return []string{"pattern-recognition-v1", "prediction-v1"}
	default:
		return []string{"general-v1"}
	}
}

func getAgentTypesForTopology(topology swarmv1alpha1.SwarmTopology) []swarmv1alpha1.AgentType {
	config := getTopologyAgentConfig(topology)
	types := make([]swarmv1alpha1.AgentType, 0, len(config))
	for agentType := range config {
		types = append(types, agentType)
	}
	return types
}

func (r *SwarmClusterReconciler) deployRedis(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	// Deploy Redis for memory backend
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-redis", cluster.Name),
			Namespace: cluster.Namespace,
		},
	}

	replicas := cluster.Spec.Memory.Replication
	if replicas == 0 {
		replicas = 1
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		deploy.Labels = map[string]string{
			"swarm-cluster": cluster.Name,
			"component":     "memory",
			"backend":       "redis",
		}

		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"swarm-cluster": cluster.Name,
					"component":     "memory",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"swarm-cluster": cluster.Name,
						"component":     "memory",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis:7-alpine",
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: 6379,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse(getOrDefault(cluster.Spec.Memory.Size, "2Gi")),
								},
							},
						},
					},
				},
			},
		}

		return controllerutil.SetControllerReference(cluster, deploy, r.Scheme)
	})

	return err
}

func (r *SwarmClusterReconciler) deployHazelcast(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	// Implementation for Hazelcast deployment
	return fmt.Errorf("hazelcast backend not yet implemented")
}

func (r *SwarmClusterReconciler) deployEtcd(ctx context.Context, cluster *swarmv1alpha1.SwarmCluster) error {
	// Implementation for etcd deployment
	return fmt.Errorf("etcd backend not yet implemented")
}

// getNamespaceForComponent returns the appropriate namespace for a component
func (r *SwarmClusterReconciler) getNamespaceForComponent(cluster *swarmv1alpha1.SwarmCluster, component string) string {
	// Check if cluster has custom namespace configuration
	if cluster.Spec.NamespaceConfig.HiveMindNamespace != "" && component == "hivemind" {
		return cluster.Spec.NamespaceConfig.HiveMindNamespace
	}
	if cluster.Spec.NamespaceConfig.SwarmNamespace != "" && component == "swarm" {
		return cluster.Spec.NamespaceConfig.SwarmNamespace
	}
	
	// Use defaults
	if component == "hivemind" && r.HiveMindNamespace != "" {
		return r.HiveMindNamespace
	}
	if component == "swarm" && r.SwarmNamespace != "" {
		return r.SwarmNamespace
	}
	
	// Fallback to cluster namespace
	return cluster.Namespace
}

func (r *SwarmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swarmv1alpha1.SwarmCluster{}).
		Owns(&swarmv1alpha1.SwarmAgent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Complete(r)
}