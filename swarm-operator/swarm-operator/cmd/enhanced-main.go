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

type EnhancedOperator struct {
	clientset *kubernetes.Clientset
	dynClient dynamic.Interface
	namespace string
}

func main() {
	log.Println("Starting Enhanced Swarm Operator v2.0.0...")

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

	namespace := os.Getenv("OPERATOR_NAMESPACE")
	if namespace == "" {
		namespace = "swarm-system"
	}

	operator := &EnhancedOperator{
		clientset: clientset,
		dynClient: dynClient,
		namespace: namespace,
	}

	// Start health and metrics servers
	go operator.startHealthServer()
	go operator.startMetricsServer()

	// Start the main reconciliation loop
	operator.run()
}

func (o *EnhancedOperator) run() {
	log.Println("Starting enhanced reconciliation loop...")
	
	// Initial reconciliation
	o.reconcileTasks()
	
	// Watch for SwarmTasks
	wait.Forever(func() {
		o.reconcileTasks()
	}, 10*time.Second)
}

func (o *EnhancedOperator) reconcileTasks() {
	// List all SwarmTasks
	tasks, err := o.dynClient.Resource(taskGVR).Namespace("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing tasks: %v", err)
		return
	}

	for _, task := range tasks.Items {
		taskName := task.GetName()
		taskSpec, found, err := unstructured.NestedMap(task.Object, "spec")
		if !found || err != nil {
			continue
		}

		// Check if we already created a job for this task
		status, _, _ := unstructured.NestedMap(task.Object, "status")
		phase, _ := status["phase"].(string)
		
		// Handle resume logic
		resume, _ := taskSpec["resume"].(bool)
		if resume && phase == "Failed" {
			log.Printf("Resuming failed task: %s", taskName)
			o.updateTaskStatus(task, "Resuming", "Preparing to resume from checkpoint")
			phase = "Resuming"
		}
		
		if phase != "" && phase != "Pending" && phase != "Resuming" {
			continue
		}

		log.Printf("Processing enhanced task: %s", taskName)
		o.createEnhancedJob(taskName, task, taskSpec)
	}
}

func (o *EnhancedOperator) createEnhancedJob(taskName string, task unstructured.Unstructured, taskSpec map[string]interface{}) {
	jobName := fmt.Sprintf("swarm-job-%s", taskName)
	
	// Check if job already exists (unless resuming)
	phase, _ := taskSpec["phase"].(string)
	if phase != "Resuming" {
		_, err := o.clientset.BatchV1().Jobs("default").Get(context.TODO(), jobName, metav1.GetOptions{})
		if err == nil {
			return // Job already exists
		}
	}

	// Get task configuration
	taskDesc, _ := taskSpec["task"].(string)
	priority, _ := taskSpec["priority"].(string)
	executorImage, _ := taskSpec["executorImage"].(string)
	if executorImage == "" {
		executorImage = "claudeflow/swarm-executor:2.0.0"
	}
	
	resume, _ := taskSpec["resume"].(bool)
	
	// Create PVCs if needed
	persistentVolumes, _ := taskSpec["persistentVolumes"].([]interface{})
	volumeMounts, volumes := o.createPersistentVolumes(taskName, persistentVolumes)
	
	// Build container spec
	container := o.buildContainer(taskName, taskDesc, executorImage, taskSpec, volumeMounts, resume)
	
	// Add additional volumes
	volumes = append(volumes, o.buildAdditionalVolumes(taskSpec)...)

	// Create Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			Labels: map[string]string{
				"swarm.claudeflow.io/task":     taskName,
				"swarm.claudeflow.io/priority": priority,
				"swarm.claudeflow.io/type":     "enhanced",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr(int32(3)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyOnFailure,
					Containers:     []corev1.Container{container},
					Volumes:        volumes,
					NodeSelector:   o.getNodeSelector(taskSpec),
					Tolerations:    o.getTolerations(taskSpec),
					ServiceAccountName: "swarm-executor",
				},
			},
		},
	}

	_, err := o.clientset.BatchV1().Jobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job: %v", err)
		o.updateTaskStatus(task, "Failed", fmt.Sprintf("Failed to create job: %v", err))
		return
	}

	log.Printf("Created enhanced job %s for task %s", jobName, taskName)
	o.updateTaskStatus(task, "Running", "Enhanced job created")
	
	// Monitor job completion
	go o.monitorEnhancedJob(jobName, task)
}

func (o *EnhancedOperator) buildContainer(taskName, taskDesc, image string, taskSpec map[string]interface{}, volumeMounts []corev1.VolumeMount, resume bool) corev1.Container {
	// Base container
	container := corev1.Container{
		Name:    "task-executor",
		Image:   image,
		Command: []string{"/bin/bash", "-c"},
		Args:    []string{taskDesc},
		Env: []corev1.EnvVar{
			{Name: "TASK_NAME", Value: taskName},
			{Name: "SWARM_ID", Value: getStringValue(taskSpec, "swarmRef")},
			{Name: "RESUME_TASK", Value: fmt.Sprintf("%v", resume)},
		},
		VolumeMounts: volumeMounts,
	}

	// Add cloud credentials if available
	container.Env = append(container.Env, o.getCloudCredentialEnvs()...)
	container.VolumeMounts = append(container.VolumeMounts, o.getCloudCredentialMounts()...)

	// Add custom environment variables
	if envMap, ok := taskSpec["environment"].(map[string]interface{}); ok {
		for k, v := range envMap {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  k,
				Value: fmt.Sprintf("%v", v),
			})
		}
	}

	// Set resources
	if resources, ok := taskSpec["resources"].(map[string]interface{}); ok {
		container.Resources = o.buildResourceRequirements(resources)
	}

	return container
}

func (o *EnhancedOperator) createPersistentVolumes(taskName string, pvSpecs []interface{}) ([]corev1.VolumeMount, []corev1.Volume) {
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	for i, pvSpec := range pvSpecs {
		pv, ok := pvSpec.(map[string]interface{})
		if !ok {
			continue
		}

		pvName := getStringValue(pv, "name")
		mountPath := getStringValue(pv, "mountPath")
		size := getStringValue(pv, "size")
		storageClass := getStringValue(pv, "storageClass")
		accessMode := getStringValue(pv, "accessMode")

		if pvName == "" || mountPath == "" {
			continue
		}

		// Create PVC
		pvcName := fmt.Sprintf("%s-%s-%d", taskName, pvName, i)
		
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: "default",
				Labels: map[string]string{
					"swarm.claudeflow.io/task": taskName,
					"swarm.claudeflow.io/type": "state",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.PersistentVolumeAccessMode(accessMode),
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(size),
					},
				},
			},
		}

		if storageClass != "" {
			pvc.Spec.StorageClassName = &storageClass
		}

		// Create PVC if it doesn't exist
		_, err := o.clientset.CoreV1().PersistentVolumeClaims("default").Get(
			context.TODO(), pvcName, metav1.GetOptions{})
		if err != nil {
			_, err = o.clientset.CoreV1().PersistentVolumeClaims("default").Create(
				context.TODO(), pvc, metav1.CreateOptions{})
			if err != nil {
				log.Printf("Failed to create PVC %s: %v", pvcName, err)
				continue
			}
			log.Printf("Created PVC %s for task %s", pvcName, taskName)
		}

		// Add volume mount
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      pvName,
			MountPath: mountPath,
		})

		// Add volume
		volumes = append(volumes, corev1.Volume{
			Name: pvName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})
	}

	// Always add swarm-state volume for checkpoints
	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      "swarm-state",
		MountPath: "/swarm-state",
	})

	return volumeMounts, volumes
}

func (o *EnhancedOperator) buildAdditionalVolumes(taskSpec map[string]interface{}) []corev1.Volume {
	var volumes []corev1.Volume

	// Add swarm-state volume
	volumes = append(volumes, corev1.Volume{
		Name: "swarm-state",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})

	// Add script volume
	volumes = append(volumes, corev1.Volume{
		Name: "scripts",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "swarm-executor-scripts",
				},
				DefaultMode: ptr(int32(0755)),
			},
		},
	})

	// Add additional secrets
	additionalSecrets, _ := taskSpec["additionalSecrets"].([]interface{})
	for _, secretSpec := range additionalSecrets {
		secret, ok := secretSpec.(map[string]interface{})
		if !ok {
			continue
		}

		secretName := getStringValue(secret, "name")
		if secretName == "" {
			continue
		}

		volume := corev1.Volume{
			Name: fmt.Sprintf("secret-%s", secretName),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  secretName,
					DefaultMode: ptr(int32(0400)),
				},
			},
		}

		// Add specific items if defined
		if items, ok := secret["items"].([]interface{}); ok {
			var keyPaths []corev1.KeyToPath
			for _, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					keyPaths = append(keyPaths, corev1.KeyToPath{
						Key:  getStringValue(itemMap, "key"),
						Path: getStringValue(itemMap, "path"),
					})
				}
			}
			if len(keyPaths) > 0 {
				volume.VolumeSource.Secret.Items = keyPaths
			}
		}

		volumes = append(volumes, volume)
	}

	return volumes
}

func (o *EnhancedOperator) buildResourceRequirements(resources map[string]interface{}) corev1.ResourceRequirements {
	req := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	if limits, ok := resources["limits"].(map[string]interface{}); ok {
		if cpu := getStringValue(limits, "cpu"); cpu != "" {
			req.Limits[corev1.ResourceCPU] = resource.MustParse(cpu)
		}
		if memory := getStringValue(limits, "memory"); memory != "" {
			req.Limits[corev1.ResourceMemory] = resource.MustParse(memory)
		}
		if gpu := getStringValue(limits, "nvidia.com/gpu"); gpu != "" {
			req.Limits["nvidia.com/gpu"] = resource.MustParse(gpu)
		}
	}

	if requests, ok := resources["requests"].(map[string]interface{}); ok {
		if cpu := getStringValue(requests, "cpu"); cpu != "" {
			req.Requests[corev1.ResourceCPU] = resource.MustParse(cpu)
		}
		if memory := getStringValue(requests, "memory"); memory != "" {
			req.Requests[corev1.ResourceMemory] = resource.MustParse(memory)
		}
	}

	return req
}

func (o *EnhancedOperator) getCloudCredentialEnvs() []corev1.EnvVar {
	var envs []corev1.EnvVar

	// Check for GCP credentials
	if _, err := o.clientset.CoreV1().Secrets("default").Get(
		context.TODO(), "gcp-credentials", metav1.GetOptions{}); err == nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "GOOGLE_APPLICATION_CREDENTIALS",
			Value: "/credentials/gcp/key.json",
		})
	}

	// Check for AWS credentials
	if _, err := o.clientset.CoreV1().Secrets("default").Get(
		context.TODO(), "aws-credentials", metav1.GetOptions{}); err == nil {
		envs = append(envs, 
			corev1.EnvVar{Name: "AWS_SHARED_CREDENTIALS_FILE", Value: "/credentials/aws/credentials"},
			corev1.EnvVar{Name: "AWS_CONFIG_FILE", Value: "/credentials/aws/config"},
		)
	}

	// Check for Azure credentials
	if _, err := o.clientset.CoreV1().Secrets("default").Get(
		context.TODO(), "azure-credentials", metav1.GetOptions{}); err == nil {
		envs = append(envs, corev1.EnvVar{
			Name:  "AZURE_CONFIG_DIR",
			Value: "/credentials/azure",
		})
	}

	return envs
}

func (o *EnhancedOperator) getCloudCredentialMounts() []corev1.VolumeMount {
	var mounts []corev1.VolumeMount

	// Add mounts for cloud credentials if they exist
	credentialMounts := map[string]string{
		"gcp-credentials":   "/credentials/gcp",
		"aws-credentials":   "/credentials/aws",
		"azure-credentials": "/credentials/azure",
		"kubeconfig":        "/credentials",
	}

	for secretName, mountPath := range credentialMounts {
		if _, err := o.clientset.CoreV1().Secrets("default").Get(
			context.TODO(), secretName, metav1.GetOptions{}); err == nil {
			mounts = append(mounts, corev1.VolumeMount{
				Name:      secretName,
				MountPath: mountPath,
				ReadOnly:  true,
			})
		}
	}

	return mounts
}

func (o *EnhancedOperator) getNodeSelector(taskSpec map[string]interface{}) map[string]string {
	selector := make(map[string]string)
	
	if nodeSelector, ok := taskSpec["nodeSelector"].(map[string]interface{}); ok {
		for k, v := range nodeSelector {
			selector[k] = fmt.Sprintf("%v", v)
		}
	}
	
	return selector
}

func (o *EnhancedOperator) getTolerations(taskSpec map[string]interface{}) []corev1.Toleration {
	var tolerations []corev1.Toleration
	
	if tolSpecs, ok := taskSpec["tolerations"].([]interface{}); ok {
		for _, tolSpec := range tolSpecs {
			if tol, ok := tolSpec.(map[string]interface{}); ok {
				toleration := corev1.Toleration{
					Key:      getStringValue(tol, "key"),
					Operator: corev1.TolerationOperator(getStringValue(tol, "operator")),
					Value:    getStringValue(tol, "value"),
					Effect:   corev1.TaintEffect(getStringValue(tol, "effect")),
				}
				tolerations = append(tolerations, toleration)
			}
		}
	}
	
	return tolerations
}

func (o *EnhancedOperator) monitorEnhancedJob(jobName string, task unstructured.Unstructured) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(2 * time.Hour) // Extended timeout for long-running jobs
	
	for {
		select {
		case <-ticker.C:
			job, err := o.clientset.BatchV1().Jobs("default").Get(context.TODO(), jobName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Error getting job %s: %v", jobName, err)
				return
			}
			
			// Check for checkpoint updates
			o.updateCheckpointStatus(task, job)
			
			if job.Status.Succeeded > 0 {
				o.updateTaskStatus(task, "Completed", "Job completed successfully")
				log.Printf("Enhanced job %s completed successfully", jobName)
				return
			}
			
			if job.Status.Failed > 0 && job.Status.Failed >= *job.Spec.BackoffLimit {
				o.updateTaskStatus(task, "Failed", fmt.Sprintf("Job failed after %d attempts", job.Status.Failed))
				log.Printf("Enhanced job %s failed", jobName)
				return
			}
			
		case <-timeout:
			o.updateTaskStatus(task, "Failed", "Job timed out")
			log.Printf("Enhanced job %s timed out", jobName)
			return
		}
	}
}

func (o *EnhancedOperator) updateCheckpointStatus(task unstructured.Unstructured, job *batchv1.Job) {
	// Get pod logs to check for checkpoints
	pods, err := o.clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	
	if err != nil || len(pods.Items) == 0 {
		return
	}
	
	// For now, we'll just update that the job is running
	// In a real implementation, you'd parse checkpoint data from pod logs or a sidecar
}

func (o *EnhancedOperator) updateTaskStatus(task unstructured.Unstructured, phase, message string) {
	status := map[string]interface{}{
		"phase":              phase,
		"message":            message,
		"lastTransitionTime": time.Now().Format(time.RFC3339),
	}

	if phase == "Completed" {
		status["progress"] = int64(100)
		status["completionTime"] = time.Now().Format(time.RFC3339)
	} else if phase == "Running" {
		status["startTime"] = time.Now().Format(time.RFC3339)
	}

	task.Object["status"] = status
	
	_, err := o.dynClient.Resource(taskGVR).Namespace(task.GetNamespace()).UpdateStatus(
		context.TODO(), &task, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update task status: %v", err)
	}
}

func (o *EnhancedOperator) startHealthServer() {
	mux := http.NewServeMux()
	
	// Liveness probe
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})
	
	// Readiness probe
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check if we can list tasks
		_, err := o.dynClient.Resource(taskGVR).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf("not ready: %v", err)))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	
	log.Println("Starting health server on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("Failed to start health server: %v", err)
	}
}

func (o *EnhancedOperator) startMetricsServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		
		// Get task counts
		tasks, _ := o.dynClient.Resource(taskGVR).List(context.TODO(), metav1.ListOptions{})
		
		var pending, running, completed, failed int
		for _, task := range tasks.Items {
			status, _, _ := unstructured.NestedMap(task.Object, "status")
			phase, _ := status["phase"].(string)
			switch phase {
			case "Pending":
				pending++
			case "Running", "Resuming":
				running++
			case "Completed":
				completed++
			case "Failed":
				failed++
			}
		}
		
		metrics := fmt.Sprintf(`# HELP swarm_operator_info Swarm operator information
# TYPE swarm_operator_info gauge
swarm_operator_info{version="2.0.0",type="enhanced"} 1
# HELP swarm_tasks_total Total number of tasks by phase
# TYPE swarm_tasks_total gauge
swarm_tasks_total{phase="pending"} %d
swarm_tasks_total{phase="running"} %d
swarm_tasks_total{phase="completed"} %d
swarm_tasks_total{phase="failed"} %d
# HELP swarm_operator_ready Operator readiness
# TYPE swarm_operator_ready gauge
swarm_operator_ready 1
`, pending, running, completed, failed)
		
		w.Write([]byte(metrics))
	})
	
	log.Println("Starting metrics server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}

// Helper functions
func ptr[T any](v T) *T {
	return &v
}

func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}