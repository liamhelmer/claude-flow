package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	swarmGVR = schema.GroupVersionResource{
		Group:    "swarm.claudeflow.io",
		Version:  "v1alpha1",
		Resource: "swarmclusters",
	}
	taskGVR = schema.GroupVersionResource{
		Group:    "swarm.claudeflow.io",
		Version:  "v1alpha1",
		Resource: "swarmtasks",
	}
)

// SecretMount represents additional secret mounting configuration
type SecretMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	Optional  bool   `json:"optional,omitempty"`
}

// PVCConfig represents persistent volume claim configuration
type PVCConfig struct {
	Name         string `json:"name"`
	MountPath    string `json:"mountPath"`
	StorageClass string `json:"storageClass,omitempty"`
	Size         string `json:"size,omitempty"`
}

// TaskConfig represents enhanced task configuration
type TaskConfig struct {
	AdditionalSecrets []SecretMount `json:"additionalSecrets,omitempty"`
	PersistentVolumes []PVCConfig   `json:"persistentVolumes,omitempty"`
	Resume            bool          `json:"resume,omitempty"`
	ExecutorImage     string        `json:"executorImage,omitempty"`
	Resources         struct {
		Requests corev1.ResourceList `json:"requests,omitempty"`
		Limits   corev1.ResourceList `json:"limits,omitempty"`
	} `json:"resources,omitempty"`
}

type EnhancedOperator struct {
	clientset *kubernetes.Clientset
	dynClient dynamic.Interface
	namespace string
	config    *OperatorConfig
}

type OperatorConfig struct {
	DefaultExecutorImage string
	EnablePersistence    bool
	DefaultStorageClass  string
	MaxRetries           int
}

func main() {
	log.Println("Starting Enhanced Swarm Operator v0.5.0 with advanced features...")

	// Setup Kubernetes clients
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to get in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	// Load operator configuration
	operatorConfig := &OperatorConfig{
		DefaultExecutorImage: getEnvOrDefault("EXECUTOR_IMAGE", "claude-flow/swarm-executor:latest"),
		EnablePersistence:    getEnvOrDefault("ENABLE_PERSISTENCE", "true") == "true",
		DefaultStorageClass:  getEnvOrDefault("DEFAULT_STORAGE_CLASS", "standard"),
		MaxRetries:           3,
	}

	namespace := getEnvOrDefault("NAMESPACE", "default")
	
	operator := &EnhancedOperator{
		clientset: clientset,
		dynClient: dynClient,
		namespace: namespace,
		config:    operatorConfig,
	}

	// Start health endpoint
	go operator.startHealthEndpoint()

	// Start the control loop
	operator.run()
}

func (o *EnhancedOperator) run() {
	wait.Forever(func() {
		// Process SwarmClusters
		swarms, err := o.dynClient.Resource(swarmGVR).Namespace(o.namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Error listing swarms: %v", err)
			return
		}

		for _, swarm := range swarms.Items {
			o.processSwarm(swarm)
		}

		// Process SwarmTasks
		tasks, err := o.dynClient.Resource(taskGVR).Namespace(o.namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Error listing tasks: %v", err)
			return
		}

		for _, task := range tasks.Items {
			o.processTask(task)
		}
	}, 10*time.Second)
}

func (o *EnhancedOperator) processTask(task unstructured.Unstructured) {
	taskName := task.GetName()
	taskSpec, found, err := unstructured.NestedMap(task.Object, "spec")
	if !found || err != nil {
		return
	}

	// Check if we already created a job for this task
	status, _, _ := unstructured.NestedMap(task.Object, "status")
	if phase, ok := status["phase"].(string); ok && phase != "" && phase != "Pending" {
		// Check if this is a resume request
		if resume, ok := taskSpec["resume"].(bool); ok && resume && phase == "Failed" {
			log.Printf("Resuming failed task: %s", taskName)
			o.createEnhancedJob(taskName, task)
		}
		return
	}

	// Process new task
	o.createEnhancedJob(taskName, task)
}

func (o *EnhancedOperator) createEnhancedJob(taskName string, task unstructured.Unstructured) {
	jobName := fmt.Sprintf("swarm-job-%s", taskName)
	
	// Check if job already exists
	_, err := o.clientset.BatchV1().Jobs(o.namespace).Get(context.TODO(), jobName, metav1.GetOptions{})
	if err == nil {
		return // Job already exists
	}

	// Parse task configuration
	taskSpec, _, _ := unstructured.NestedMap(task.Object, "spec")
	taskConfig := o.parseTaskConfig(taskSpec)

	// Create enhanced container spec
	container := o.createEnhancedContainer(taskName, taskSpec, taskConfig)
	
	// Create volumes
	volumes, volumeMounts := o.createVolumes(taskName, taskConfig)
	container.VolumeMounts = append(container.VolumeMounts, volumeMounts...)

	// Create job spec
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(o.config.MaxRetries)
	ttlSecondsAfterFinished := int32(3600) // Clean up after 1 hour

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: o.namespace,
			Labels: map[string]string{
				"app":        "swarm-task",
				"task":       taskName,
				"managed-by": "swarm-operator",
			},
		},
		Spec: batchv1.JobSpec{
			Parallelism:             &parallelism,
			Completions:             &completions,
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":  "swarm-task",
						"task": taskName,
					},
					Annotations: map[string]string{
						"swarm.claudeflow.io/task-name": taskName,
						"swarm.claudeflow.io/created":   time.Now().Format(time.RFC3339),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers:    []corev1.Container{container},
					Volumes:       volumes,
				},
			},
		},
	}

	// Create the job
	_, err = o.clientset.BatchV1().Jobs(o.namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job for task %s: %v", taskName, err)
		o.updateTaskStatus(task, "Failed", fmt.Sprintf("Job creation failed: %v", err))
		return
	}

	log.Printf("Created enhanced job %s for task %s", jobName, taskName)
	o.updateTaskStatus(task, "Running", "Enhanced job created")
}

func (o *EnhancedOperator) createEnhancedContainer(taskName string, taskSpec map[string]interface{}, config *TaskConfig) corev1.Container {
	// Determine image to use
	image := config.ExecutorImage
	if image == "" {
		image = o.config.DefaultExecutorImage
	}

	// Base container configuration
	container := corev1.Container{
		Name:    "task-executor",
		Image:   image,
		Command: []string{"/scripts/entrypoint.sh"},
		Args:    []string{"/scripts/task.sh"},
		Env:     o.createEnvironmentVariables(taskSpec),
		Resources: corev1.ResourceRequirements{
			Requests: config.Resources.Requests,
			Limits:   config.Resources.Limits,
		},
	}

	// Add default resource limits if not specified
	if container.Resources.Requests == nil {
		container.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		}
	}
	if container.Resources.Limits == nil {
		container.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		}
	}

	return container
}

func (o *EnhancedOperator) createEnvironmentVariables(taskSpec map[string]interface{}) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "TASK_NAME",
			Value: taskSpec["task"].(string),
		},
		{
			Name:  "PRIORITY",
			Value: getStringOrDefault(taskSpec["priority"], "medium"),
		},
		{
			Name:  "SWARM_OPERATOR_VERSION",
			Value: "0.5.0",
		},
	}

	// Add GitHub credentials if available
	env = append(env, o.createGitHubEnvVars()...)

	// Add cloud provider credentials if available
	env = append(env, o.createCloudProviderEnvVars()...)

	return env
}

func (o *EnhancedOperator) createGitHubEnvVars() []corev1.EnvVar {
	vars := []corev1.EnvVar{}

	// Check for GitHub App credentials
	useGitHubApp := false
	_, err := o.clientset.CoreV1().Secrets(o.namespace).Get(context.TODO(), "github-app-credentials", metav1.GetOptions{})
	if err == nil {
		useGitHubApp = true
	}

	if useGitHubApp {
		vars = append(vars, []corev1.EnvVar{
			{
				Name: "APP_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "github-app-credentials"},
						Key:                  "app-id",
					},
				},
			},
			{
				Name: "CLIENT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "github-app-credentials"},
						Key:                  "client-id",
						Optional:             ptr(true),
					},
				},
			},
			{
				Name: "INSTALLATION_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "github-app-credentials"},
						Key:                  "installation-id",
					},
				},
			},
		}...)
	} else {
		// Use personal access token
		vars = append(vars, []corev1.EnvVar{
			{
				Name: "GITHUB_TOKEN",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "github-credentials"},
						Key:                  "token",
						Optional:             ptr(true),
					},
				},
			},
			{
				Name: "GITHUB_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "github-credentials"},
						Key:                  "username",
						Optional:             ptr(true),
					},
				},
			},
		}...)
	}

	return vars
}

func (o *EnhancedOperator) createCloudProviderEnvVars() []corev1.EnvVar {
	vars := []corev1.EnvVar{}

	// Google Cloud credentials
	_, err := o.clientset.CoreV1().Secrets(o.namespace).Get(context.TODO(), "gcp-credentials", metav1.GetOptions{})
	if err == nil {
		vars = append(vars, corev1.EnvVar{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: "/secrets/gcp/key.json",
		})
	}

	// AWS credentials
	_, err = o.clientset.CoreV1().Secrets(o.namespace).Get(context.TODO(), "aws-credentials", metav1.GetOptions{})
	if err == nil {
		vars = append(vars, []corev1.EnvVar{
			{
				Name: "AWS_ACCESS_KEY_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "aws-credentials"},
						Key:                  "access-key-id",
						Optional:             ptr(true),
					},
				},
			},
			{
				Name: "AWS_SECRET_ACCESS_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "aws-credentials"},
						Key:                  "secret-access-key",
						Optional:             ptr(true),
					},
				},
			},
			{
				Name: "AWS_DEFAULT_REGION",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "aws-credentials"},
						Key:                  "region",
						Optional:             ptr(true),
					},
				},
			},
		}...)
	}

	// Azure credentials
	_, err = o.clientset.CoreV1().Secrets(o.namespace).Get(context.TODO(), "azure-credentials", metav1.GetOptions{})
	if err == nil {
		vars = append(vars, []corev1.EnvVar{
			{
				Name: "AZURE_CLIENT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "azure-credentials"},
						Key:                  "client-id",
						Optional:             ptr(true),
					},
				},
			},
			{
				Name: "AZURE_CLIENT_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "azure-credentials"},
						Key:                  "client-secret",
						Optional:             ptr(true),
					},
				},
			},
			{
				Name: "AZURE_TENANT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "azure-credentials"},
						Key:                  "tenant-id",
						Optional:             ptr(true),
					},
				},
			},
		}...)
	}

	return vars
}

func (o *EnhancedOperator) createVolumes(taskName string, config *TaskConfig) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Add script ConfigMap
	volumes = append(volumes, corev1.Volume{
		Name: "scripts",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "swarm-task-scripts",
				},
				DefaultMode: ptr(int32(0755)),
			},
		},
	})
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "scripts",
		MountPath: "/scripts",
	})

	// Add additional secret mounts
	for i, secret := range config.AdditionalSecrets {
		volumeName := fmt.Sprintf("secret-%d", i)
		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
					Optional:   &secret.Optional,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: secret.MountPath,
			ReadOnly:  true,
		})
	}

	// Add persistent volume claims
	for i, pvc := range config.PersistentVolumes {
		volumeName := fmt.Sprintf("pvc-%d", i)
		
		// Check if PVC exists, create if not
		pvcName := fmt.Sprintf("%s-%s", taskName, pvc.Name)
		o.ensurePVC(pvcName, pvc)

		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: pvc.MountPath,
		})
	}

	// Add cloud provider credential mounts
	if _, err := o.clientset.CoreV1().Secrets(o.namespace).Get(context.TODO(), "gcp-credentials", metav1.GetOptions{}); err == nil {
		volumes = append(volumes, corev1.Volume{
			Name: "gcp-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "gcp-credentials",
					Optional:   ptr(true),
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "gcp-credentials",
			MountPath: "/secrets/gcp",
			ReadOnly:  true,
		})
	}

	return volumes, volumeMounts
}

func (o *EnhancedOperator) ensurePVC(name string, config PVCConfig) {
	// Check if PVC exists
	_, err := o.clientset.CoreV1().PersistentVolumeClaims(o.namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		return // PVC already exists
	}

	// Create PVC
	storageClass := config.StorageClass
	if storageClass == "" {
		storageClass = o.config.DefaultStorageClass
	}

	size := config.Size
	if size == "" {
		size = "10Gi"
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: o.namespace,
			Labels: map[string]string{
				"app":        "swarm-task",
				"managed-by": "swarm-operator",
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: &storageClass,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}

	_, err = o.clientset.CoreV1().PersistentVolumeClaims(o.namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create PVC %s: %v", name, err)
	} else {
		log.Printf("Created PVC %s for swarm task", name)
	}
}

func (o *EnhancedOperator) parseTaskConfig(taskSpec map[string]interface{}) *TaskConfig {
	config := &TaskConfig{}

	// Parse config if present
	if configData, ok := taskSpec["config"]; ok {
		if configMap, ok := configData.(map[string]interface{}); ok {
			// Parse additional secrets
			if secrets, ok := configMap["additionalSecrets"].([]interface{}); ok {
				for _, s := range secrets {
					if secretMap, ok := s.(map[string]interface{}); ok {
						mount := SecretMount{
							Name:      getStringOrDefault(secretMap["name"], ""),
							MountPath: getStringOrDefault(secretMap["mountPath"], ""),
							Optional:  getBoolOrDefault(secretMap["optional"], false),
						}
						if mount.Name != "" && mount.MountPath != "" {
							config.AdditionalSecrets = append(config.AdditionalSecrets, mount)
						}
					}
				}
			}

			// Parse persistent volumes
			if pvcs, ok := configMap["persistentVolumes"].([]interface{}); ok {
				for _, p := range pvcs {
					if pvcMap, ok := p.(map[string]interface{}); ok {
						pvcConfig := PVCConfig{
							Name:         getStringOrDefault(pvcMap["name"], ""),
							MountPath:    getStringOrDefault(pvcMap["mountPath"], ""),
							StorageClass: getStringOrDefault(pvcMap["storageClass"], ""),
							Size:         getStringOrDefault(pvcMap["size"], ""),
						}
						if pvcConfig.Name != "" && pvcConfig.MountPath != "" {
							config.PersistentVolumes = append(config.PersistentVolumes, pvcConfig)
						}
					}
				}
			}

			// Parse other config options
			config.Resume = getBoolOrDefault(configMap["resume"], false)
			config.ExecutorImage = getStringOrDefault(configMap["executorImage"], "")
		}
	}

	return config
}

func (o *EnhancedOperator) updateTaskStatus(task unstructured.Unstructured, phase, message string) {
	status := map[string]interface{}{
		"phase":   phase,
		"message": message,
		"lastUpdateTime": time.Now().Format(time.RFC3339),
	}

	// Update the task status
	task.Object["status"] = status
	_, err := o.dynClient.Resource(taskGVR).Namespace(o.namespace).UpdateStatus(
		context.TODO(),
		&task,
		metav1.UpdateOptions{},
	)
	if err != nil {
		log.Printf("Failed to update task status: %v", err)
	}
}

func (o *EnhancedOperator) processSwarm(swarm unstructured.Unstructured) {
	// Process swarm logic (unchanged from original)
	swarmName := swarm.GetName()
	log.Printf("Processing swarm: %s", swarmName)
}

func (o *EnhancedOperator) startHealthEndpoint() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check if we can reach the API server
		_, err := o.clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{Limit: 1})
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf("Not ready: %v", err)))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Helper functions
func ptr[T any](v T) *T {
	return &v
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getStringOrDefault(value interface{}, defaultValue string) string {
	if str, ok := value.(string); ok {
		return str
	}
	return defaultValue
}

func getBoolOrDefault(value interface{}, defaultValue bool) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	return defaultValue
}