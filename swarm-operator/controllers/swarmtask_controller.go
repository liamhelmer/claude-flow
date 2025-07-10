/*
Copyright 2025 The Claude Flow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
	"github.com/claude-flow/swarm-operator/pkg/github"
)

const (
	swarmTaskFinalizer = "swarmtask.swarm.claudeflow.io/finalizer"
)

// SwarmTaskReconciler reconciles a SwarmTask object
type SwarmTaskReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Recorder          record.EventRecorder
	SwarmNamespace    string
	HiveMindNamespace string
	TokenGenerator    *github.TokenGenerator
}

// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmtasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmtasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmtasks/finalizers,verbs=update
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmagents,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create

func (r *SwarmTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the SwarmTask
	task := &swarmv1alpha1.SwarmTask{}
	err := r.Get(ctx, req.NamespacedName, task)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if task.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(task, swarmTaskFinalizer) {
			// Cleanup any resources
			if err := r.finalizeSwarmTask(ctx, task); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(task, swarmTaskFinalizer)
			if err := r.Update(ctx, task); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(task, swarmTaskFinalizer) {
		controllerutil.AddFinalizer(task, swarmTaskFinalizer)
		if err := r.Update(ctx, task); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Determine target namespace
	targetNamespace := r.determineNamespace(task)

	// Ensure namespace exists
	if err := r.ensureNamespace(ctx, targetNamespace); err != nil {
		log.Error(err, "Failed to ensure namespace", "namespace", targetNamespace)
		return ctrl.Result{}, err
	}

	// Get the SwarmCluster
	cluster := &swarmv1alpha1.SwarmCluster{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      task.Spec.SwarmCluster,
		Namespace: task.Namespace,
	}, cluster)
	if err != nil {
		log.Error(err, "Failed to get SwarmCluster", "cluster", task.Spec.SwarmCluster)
		return ctrl.Result{}, err
	}

	// Generate GitHub token if needed
	var githubTokenSecret string
	if cluster.Spec.GitHubApp != nil && len(task.Spec.Repositories) > 0 {
		tokenSecret, err := r.ensureGitHubToken(ctx, task, cluster.Spec.GitHubApp, targetNamespace)
		if err != nil {
			log.Error(err, "Failed to ensure GitHub token")
			return ctrl.Result{}, err
		}
		githubTokenSecret = tokenSecret
	}

	// Create or update the Job
	job, err := r.createOrUpdateJob(ctx, task, targetNamespace, githubTokenSecret)
	if err != nil {
		log.Error(err, "Failed to create/update job")
		return ctrl.Result{}, err
	}

	// Update task status based on job status
	if err := r.updateTaskStatus(ctx, task, job); err != nil {
		log.Error(err, "Failed to update task status")
		return ctrl.Result{}, err
	}

	// Requeue to check job status
	if task.Status.Phase != "Completed" && task.Status.Phase != "Failed" {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// determineNamespace returns the appropriate namespace for the task
func (r *SwarmTaskReconciler) determineNamespace(task *swarmv1alpha1.SwarmTask) string {
	// If namespace is explicitly set in the task, use it
	if task.Spec.Namespace != "" {
		return task.Spec.Namespace
	}

	// Determine based on task type
	if task.Spec.Type == "hivemind" || task.Spec.Type == "consensus" {
		return r.HiveMindNamespace
	}

	// Default to swarm namespace
	return r.SwarmNamespace
}

// ensureNamespace ensures the target namespace exists
func (r *SwarmTaskReconciler) ensureNamespace(ctx context.Context, namespace string) error {
	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create namespace
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						"swarm.claudeflow.io/managed": "true",
					},
				},
			}
			return r.Create(ctx, ns)
		}
		return err
	}
	return nil
}

// ensureGitHubToken ensures a GitHub token exists for the task
func (r *SwarmTaskReconciler) ensureGitHubToken(ctx context.Context, task *swarmv1alpha1.SwarmTask, appConfig *swarmv1alpha1.GitHubAppConfig, namespace string) (string, error) {
	if r.TokenGenerator == nil {
		r.TokenGenerator = github.NewTokenGenerator(r.Client)
	}

	secretName := fmt.Sprintf("%s-github-token", task.Name)

	// Check if token already exists and is valid
	expired, err := r.TokenGenerator.IsTokenExpired(ctx, secretName, namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			return "", err
		}
		// Token doesn't exist, create it
		expired = true
	}

	if expired {
		// Generate new token
		token, err := r.TokenGenerator.GenerateToken(ctx, appConfig, task.Spec.Repositories, namespace)
		if err != nil {
			return "", err
		}

		// Parse TTL
		ttl, _ := time.ParseDuration(appConfig.TokenTTL)
		if ttl == 0 {
			ttl = time.Hour
		}
		expiresAt := time.Now().Add(ttl)

		// Create or update secret
		if errors.IsNotFound(err) {
			err = r.TokenGenerator.CreateTokenSecret(ctx, secretName, namespace, token, task.Spec.Repositories, expiresAt)
		} else {
			err = r.TokenGenerator.UpdateTokenSecret(ctx, secretName, namespace, token, task.Spec.Repositories, expiresAt)
		}
		if err != nil {
			return "", err
		}

		r.Recorder.Eventf(task, corev1.EventTypeNormal, "GitHubTokenCreated", 
			"Created GitHub token for repositories: %v", task.Spec.Repositories)
	}

	return secretName, nil
}

// createOrUpdateJob creates or updates the Kubernetes Job for the task
func (r *SwarmTaskReconciler) createOrUpdateJob(ctx context.Context, task *swarmv1alpha1.SwarmTask, namespace string, githubTokenSecret string) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-job", task.Name)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"swarm.claudeflow.io/task":    task.Name,
				"swarm.claudeflow.io/cluster": task.Spec.SwarmCluster,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"swarm.claudeflow.io/task":    task.Name,
						"swarm.claudeflow.io/cluster": task.Spec.SwarmCluster,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "task",
							Image: "busybox:latest", // This should be configurable
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{fmt.Sprintf("echo 'Executing task: %s'", task.Spec.Description)},
							Env:     r.buildEnvironment(task, githubTokenSecret),
						},
					},
				},
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(task, job, r.Scheme); err != nil {
		return nil, err
	}

	// Check if job exists
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: namespace}, existingJob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new job
			if err := r.Create(ctx, job); err != nil {
				return nil, err
			}
			return job, nil
		}
		return nil, err
	}

	return existingJob, nil
}

// buildEnvironment builds environment variables for the task
func (r *SwarmTaskReconciler) buildEnvironment(task *swarmv1alpha1.SwarmTask, githubTokenSecret string) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "SWARM_TASK_NAME",
			Value: task.Name,
		},
		{
			Name:  "SWARM_CLUSTER",
			Value: task.Spec.SwarmCluster,
		},
		{
			Name:  "SWARM_TASK_TYPE",
			Value: task.Spec.Type,
		},
	}

	// Add GitHub token if present
	if githubTokenSecret != "" {
		env = append(env, corev1.EnvVar{
			Name: "GITHUB_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: githubTokenSecret,
					},
					Key: "token",
				},
			},
		})
		
		// Add repository list
		if len(task.Spec.Repositories) > 0 {
			env = append(env, corev1.EnvVar{
				Name:  "GITHUB_REPOSITORIES",
				Value: strings.Join(task.Spec.Repositories, ","),
			})
		}
	}

	// Add custom parameters
	for k, v := range task.Spec.Parameters {
		env = append(env, corev1.EnvVar{
			Name:  fmt.Sprintf("PARAM_%s", strings.ToUpper(k)),
			Value: v,
		})
	}

	return env
}

// updateTaskStatus updates the SwarmTask status based on the Job status
func (r *SwarmTaskReconciler) updateTaskStatus(ctx context.Context, task *swarmv1alpha1.SwarmTask, job *batchv1.Job) error {
	updated := false

	// Update phase based on job status
	if job.Status.Succeeded > 0 {
		if task.Status.Phase != "Completed" {
			task.Status.Phase = "Completed"
			task.Status.CompletionTime = &metav1.Time{Time: time.Now()}
			updated = true
		}
	} else if job.Status.Failed > 0 {
		if task.Status.Phase != "Failed" {
			task.Status.Phase = "Failed"
			task.Status.CompletionTime = &metav1.Time{Time: time.Now()}
			task.Status.Message = "Job failed"
			updated = true
		}
	} else if job.Status.Active > 0 {
		if task.Status.Phase != "Running" {
			task.Status.Phase = "Running"
			if task.Status.StartTime == nil {
				task.Status.StartTime = &metav1.Time{Time: time.Now()}
			}
			updated = true
		}
	} else {
		if task.Status.Phase != "Pending" {
			task.Status.Phase = "Pending"
			updated = true
		}
	}

	if updated {
		return r.Status().Update(ctx, task)
	}

	return nil
}

// finalizeSwarmTask cleans up resources when task is deleted
func (r *SwarmTaskReconciler) finalizeSwarmTask(ctx context.Context, task *swarmv1alpha1.SwarmTask) error {
	log := log.FromContext(ctx)

	// Clean up GitHub token secret if it exists
	if task.Spec.GitHubApp != nil {
		secretName := fmt.Sprintf("%s-github-token", task.Name)
		secret := &corev1.Secret{}
		err := r.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: r.determineNamespace(task),
		}, secret)
		if err == nil {
			if err := r.Delete(ctx, secret); err != nil {
				log.Error(err, "Failed to delete GitHub token secret")
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwarmTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swarmv1alpha1.SwarmTask{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}