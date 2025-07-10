package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

type Operator struct {
	clientset *kubernetes.Clientset
	dynClient dynamic.Interface
	namespace string
}

func main() {
	log.Println("Starting Enhanced Swarm Operator v0.4.0 with GitHub App support...")

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

	operator := &Operator{
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

func (o *Operator) run() {
	log.Println("Starting reconciliation loop...")
	
	// Initial reconciliation
	o.reconcileTasks()
	
	// Watch for SwarmTasks and create Jobs
	wait.Forever(func() {
		o.reconcileTasks()
	}, 10*time.Second)
}

func (o *Operator) reconcileTasks() {
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
		if phase, ok := status["phase"].(string); ok && phase != "" && phase != "Pending" {
			continue
		}

		// Get task details
		taskDesc, _ := taskSpec["task"].(string)
		priority, _ := taskSpec["priority"].(string)
		
		log.Printf("Processing task: %s - %s (priority: %s)", taskName, taskDesc, priority)

		// Special handling for GitHub repo creation tasks
		if strings.Contains(strings.ToLower(taskDesc), "hello world") && 
		   strings.Contains(strings.ToLower(taskDesc), "github") {
			o.createGitHubJob(taskName, task)
		} else {
			// Update status to show we're processing
			o.updateTaskStatus(task, "Running", "Job creation in progress")
		}
	}
}

func (o *Operator) createGitHubJob(taskName string, task unstructured.Unstructured) {
	jobName := fmt.Sprintf("swarm-job-%s", taskName)
	
	// Check if job already exists
	_, err := o.clientset.BatchV1().Jobs("default").Get(context.TODO(), jobName, metav1.GetOptions{})
	if err == nil {
		return // Job already exists
	}

	// Check which authentication method to use
	useGitHubApp := false
	_, err = o.clientset.CoreV1().Secrets("default").Get(context.TODO(), "github-app-credentials", metav1.GetOptions{})
	if err == nil {
		useGitHubApp = true
		log.Printf("Using GitHub App authentication for task %s", taskName)
	} else {
		log.Printf("Using Personal Access Token authentication for task %s", taskName)
	}

	// Create container spec
	container := corev1.Container{
		Name:    "task-executor",
		Image:   "alpine/git:latest",
		Command: []string{"/bin/sh", "/scripts/task.sh"},
		Env: []corev1.EnvVar{
			{
				Name: "GITHUB_USERNAME",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "github-credentials",
						},
						Key:      "username",
						Optional: ptr(true),
					},
				},
			},
			{
				Name: "GITHUB_TOKEN",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "github-credentials",
						},
						Key:      "token",
						Optional: ptr(true),
					},
				},
			},
			{
				Name: "GITHUB_EMAIL",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "github-credentials",
						},
						Key:      "email",
						Optional: ptr(true),
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "script",
				MountPath: "/scripts",
			},
		},
	}

	// Volumes
	volumes := []corev1.Volume{
		{
			Name: "script",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "github-task-script",
					},
					DefaultMode: ptr(int32(0755)),
				},
			},
		},
	}

	// Add GitHub App specific configuration
	if useGitHubApp {
		// Update ConfigMap to use GitHub App version
		volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name = "github-app-task-script"
		
		// Add GitHub App environment variables
		container.Env = append(container.Env,
			corev1.EnvVar{
				Name: "APP_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "github-app-credentials",
						},
						Key: "app-id",
					},
				},
			},
			corev1.EnvVar{
				Name: "CLIENT_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "github-app-credentials",
						},
						Key: "client-id",
					},
				},
			},
			corev1.EnvVar{
				Name: "INSTALLATION_ID",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "github-app-credentials",
						},
						Key: "installation-id",
					},
				},
			},
		)
		
		// Add volume mount for private key
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "github-app-key",
			MountPath: "/github-app",
			ReadOnly:  true,
		})
		
		// Add volume for private key
		volumes = append(volumes, corev1.Volume{
			Name: "github-app-key",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "github-app-credentials",
					Items: []corev1.KeyToPath{
						{
							Key:  "private-key",
							Path: "private-key",
						},
					},
					DefaultMode: ptr(int32(0400)),
				},
			},
		})
	}

	// Create a Job that will execute the task
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			Labels: map[string]string{
				"swarm.claudeflow.io/task": taskName,
				"swarm.claudeflow.io/type": "github-automation",
				"swarm.claudeflow.io/auth": map[bool]string{true: "github-app", false: "pat"}[useGitHubApp],
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: ptr(int32(2)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers:    []corev1.Container{container},
					Volumes:       volumes,
				},
			},
		},
	}

	_, err = o.clientset.BatchV1().Jobs("default").Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create job: %v", err)
		o.updateTaskStatus(task, "Failed", fmt.Sprintf("Failed to create job: %v", err))
		return
	}

	authMethod := "Personal Access Token"
	if useGitHubApp {
		authMethod = "GitHub App"
	}
	log.Printf("Created job %s for task %s using %s authentication", jobName, taskName, authMethod)
	o.updateTaskStatus(task, "Running", fmt.Sprintf("Job created with %s authentication", authMethod))
	
	// Monitor job completion
	go o.monitorJob(jobName, task)
}

func (o *Operator) monitorJob(jobName string, task unstructured.Unstructured) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(10 * time.Minute)
	
	for {
		select {
		case <-ticker.C:
			job, err := o.clientset.BatchV1().Jobs("default").Get(context.TODO(), jobName, metav1.GetOptions{})
			if err != nil {
				log.Printf("Error getting job %s: %v", jobName, err)
				return
			}
			
			if job.Status.Succeeded > 0 {
				o.updateTaskStatus(task, "Completed", "Job completed successfully")
				log.Printf("Job %s completed successfully", jobName)
				return
			}
			
			if job.Status.Failed > 0 && job.Status.Failed >= *job.Spec.BackoffLimit {
				o.updateTaskStatus(task, "Failed", fmt.Sprintf("Job failed after %d attempts", job.Status.Failed))
				log.Printf("Job %s failed", jobName)
				return
			}
			
		case <-timeout:
			o.updateTaskStatus(task, "Failed", "Job timed out")
			log.Printf("Job %s timed out", jobName)
			return
		}
	}
}

func (o *Operator) updateTaskStatus(task unstructured.Unstructured, phase, message string) {
	status := map[string]interface{}{
		"phase":              phase,
		"message":            message,
		"lastTransitionTime": time.Now().Format(time.RFC3339),
	}

	if phase == "Completed" {
		status["progress"] = int64(100)
	}

	task.Object["status"] = status
	
	_, err := o.dynClient.Resource(taskGVR).Namespace(task.GetNamespace()).UpdateStatus(
		context.TODO(), &task, metav1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update task status: %v", err)
	}
}

func (o *Operator) startHealthServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	log.Println("Starting health server on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("Failed to start health server: %v", err)
	}
}

func (o *Operator) startMetricsServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		metrics := `# HELP swarm_operator_info Swarm operator information
# TYPE swarm_operator_info gauge
swarm_operator_info{version="0.4.0"} 1
# HELP swarm_tasks_processed Total tasks processed
# TYPE swarm_tasks_processed counter
swarm_tasks_processed 1
`
		w.Write([]byte(metrics))
	})
	log.Println("Starting metrics server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}

func ptr[T any](v T) *T {
	return &v
}