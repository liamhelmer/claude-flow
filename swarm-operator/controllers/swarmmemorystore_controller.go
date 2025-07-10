/*
Copyright 2025.

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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	swarmv1alpha1 "github.com/liamhelmer/claude-flow/swarm-operator/api/v1alpha1"
)

// SwarmMemoryStoreReconciler reconciles a SwarmMemoryStore object
type SwarmMemoryStoreReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	SwarmNamespace string
}

//+kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmmemorystores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmmemorystores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmmemorystores/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *SwarmMemoryStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the SwarmMemoryStore instance
	memory := &swarmv1alpha1.SwarmMemoryStore{}
	err := r.Get(ctx, req.NamespacedName, memory)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("SwarmMemoryStore resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get SwarmMemoryStore")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if memory.GetDeletionTimestamp() != nil {
		return r.handleDelete(ctx, memory)
	}

	// Ensure finalizer
	if !containsString(memory.GetFinalizers(), swarmMemoryFinalizer) {
		memory.SetFinalizers(append(memory.GetFinalizers(), swarmMemoryFinalizer))
		if err := r.Update(ctx, memory); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Determine namespace
	namespace := r.determineNamespace(memory)

	// Reconcile PVC
	if err := r.reconcilePVC(ctx, memory, namespace); err != nil {
		logger.Error(err, "Failed to reconcile PVC")
		return ctrl.Result{}, err
	}

	// Reconcile ConfigMap with migration scripts
	if err := r.reconcileConfigMap(ctx, memory, namespace); err != nil {
		logger.Error(err, "Failed to reconcile ConfigMap")
		return ctrl.Result{}, err
	}

	// Reconcile StatefulSet for memory service
	if err := r.reconcileStatefulSet(ctx, memory, namespace); err != nil {
		logger.Error(err, "Failed to reconcile StatefulSet")
		return ctrl.Result{}, err
	}

	// Run migration if needed
	if memory.Spec.MigrateFromLegacy {
		if err := r.runMigration(ctx, memory, namespace); err != nil {
			logger.Error(err, "Failed to run migration")
			return ctrl.Result{}, err
		}
	}

	// Update status
	memory.Status.Phase = "Ready"
	memory.Status.StorageReady = true
	memory.Status.LastBackup = memory.Status.LastBackup // Keep existing value
	memory.Status.DatabaseSize = r.getDatabaseSize(ctx, memory, namespace)
	
	if err := r.Status().Update(ctx, memory); err != nil {
		logger.Error(err, "Failed to update SwarmMemoryStore status")
		return ctrl.Result{}, err
	}

	// Requeue for periodic backup check
	if memory.Spec.BackupInterval != "" {
		duration, _ := time.ParseDuration(memory.Spec.BackupInterval)
		if duration > 0 {
			return ctrl.Result{RequeueAfter: duration}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *SwarmMemoryStoreReconciler) determineNamespace(memory *swarmv1alpha1.SwarmMemoryStore) string {
	// If namespace is specified in the spec, use it
	if memory.Spec.Namespace != "" {
		return memory.Spec.Namespace
	}
	
	// If this is for a specific SwarmCluster, check its namespace config
	if memory.Spec.SwarmClusterRef != "" {
		// In a real implementation, we'd look up the SwarmCluster
		// For now, use the default swarm namespace
		return r.SwarmNamespace
	}
	
	// Default to the configured swarm namespace
	return r.SwarmNamespace
}

func (r *SwarmMemoryStoreReconciler) reconcilePVC(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore, namespace string) error {
	logger := log.FromContext(ctx)
	
	// Define PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memory.Name + "-storage",
			Namespace: namespace,
			Labels: map[string]string{
				"app":         "swarm-memory",
				"memory-name": memory.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(memory.Spec.StorageSize),
				},
			},
		},
	}
	
	if memory.Spec.StorageClass != "" {
		pvc.Spec.StorageClassName = &memory.Spec.StorageClass
	}
	
	// Check if PVC exists
	foundPVC := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating PVC", "Name", pvc.Name, "Namespace", pvc.Namespace)
		if err := r.Create(ctx, pvc); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	
	return nil
}

func (r *SwarmMemoryStoreReconciler) reconcileConfigMap(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore, namespace string) error {
	logger := log.FromContext(ctx)
	
	// Create ConfigMap with initialization scripts
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memory.Name + "-scripts",
			Namespace: namespace,
			Labels: map[string]string{
				"app":         "swarm-memory",
				"memory-name": memory.Name,
			},
		},
		Data: map[string]string{
			"init.sh": `#!/bin/bash
set -e

# Initialize SQLite database directory
mkdir -p /data/memory

# Create initial database if it doesn't exist
if [ ! -f /data/memory/swarm-memory.db ]; then
  echo "Initializing new SQLite database..."
  sqlite3 /data/memory/swarm-memory.db < /scripts/schema.sql
fi

echo "Database initialization complete"
`,
			"schema.sql": getEnhancedSchema(),
			"migrate.sh": `#!/bin/bash
set -e

# Migration script from legacy memory systems
if [ -f /legacy/memory-store.json ]; then
  echo "Migrating from legacy JSON store..."
  node /app/src/memory/migration.js --source=/legacy/memory-store.json --target=/data/memory/swarm-memory.db
fi

if [ -f /legacy/hive.db ]; then
  echo "Migrating from legacy hive database..."
  node /app/src/memory/migration.js --source=/legacy/hive.db --target=/data/memory/swarm-memory.db --type=sqlite
fi

echo "Migration complete"
`,
		},
	}
	
	// Check if ConfigMap exists
	foundCM := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, foundCM)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating ConfigMap", "Name", cm.Name, "Namespace", cm.Namespace)
		if err := r.Create(ctx, cm); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	
	return nil
}

func (r *SwarmMemoryStoreReconciler) reconcileStatefulSet(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore, namespace string) error {
	logger := log.FromContext(ctx)
	
	// Define StatefulSet
	replicas := int32(1)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memory.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":         "swarm-memory",
				"memory-name": memory.Name,
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: memory.Name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":         "swarm-memory",
					"memory-name": memory.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":         "swarm-memory",
						"memory-name": memory.Name,
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "init-db",
							Image: "alpine:3.18",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{"apk add --no-cache sqlite && /scripts/init.sh"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "scripts",
									MountPath: "/scripts",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "memory-service",
							Image: fmt.Sprintf("claudeflow/swarm-memory:%s", memory.Spec.Version),
							Env: []corev1.EnvVar{
								{
									Name:  "SWARM_ID",
									Value: memory.Spec.SwarmID,
								},
								{
									Name:  "DB_PATH",
									Value: "/data/memory/swarm-memory.db",
								},
								{
									Name:  "CACHE_SIZE",
									Value: fmt.Sprintf("%d", memory.Spec.CacheSize),
								},
								{
									Name:  "CACHE_MEMORY_MB",
									Value: fmt.Sprintf("%d", memory.Spec.CacheMemoryMB),
								},
								{
									Name:  "GC_INTERVAL",
									Value: memory.Spec.GCInterval,
								},
								{
									Name:  "COMPRESSION_THRESHOLD",
									Value: fmt.Sprintf("%d", memory.Spec.CompressionThreshold),
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "grpc",
									ContainerPort: 9090,
								},
								{
									Name:          "metrics",
									ContainerPort: 9091,
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
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: memory.Name + "-storage",
								},
							},
						},
						{
							Name: "scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: memory.Name + "-scripts",
									},
									DefaultMode: &[]int32{0755}[0],
								},
							},
						},
					},
				},
			},
		},
	}
	
	// Check if StatefulSet exists
	foundSts := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, foundSts)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating StatefulSet", "Name", sts.Name, "Namespace", sts.Namespace)
		if err := r.Create(ctx, sts); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	
	return nil
}

func (r *SwarmMemoryStoreReconciler) runMigration(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore, namespace string) error {
	logger := log.FromContext(ctx)
	
	// Check if migration has already been run
	if memory.Status.MigrationCompleted {
		return nil
	}
	
	// Create migration job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memory.Name + "-migration",
			Namespace: namespace,
			Labels: map[string]string{
				"app":         "swarm-memory",
				"memory-name": memory.Name,
				"job-type":    "migration",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:  "migrate",
							Image: fmt.Sprintf("claudeflow/swarm-memory:%s", memory.Spec.Version),
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{"/scripts/migrate.sh"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/data",
								},
								{
									Name:      "scripts",
									MountPath: "/scripts",
								},
								{
									Name:      "legacy-data",
									MountPath: "/legacy",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: memory.Name + "-storage",
								},
							},
						},
						{
							Name: "scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: memory.Name + "-scripts",
									},
									DefaultMode: &[]int32{0755}[0],
								},
							},
						},
						{
							Name: "legacy-data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: memory.Spec.LegacyDataPVC,
								},
							},
						},
					},
				},
			},
		},
	}
	
	// Check if job exists
	foundJob := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, foundJob)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating migration job", "Name", job.Name)
		if err := r.Create(ctx, job); err != nil {
			return err
		}
	} else if err == nil {
		// Check job status
		if foundJob.Status.Succeeded > 0 {
			memory.Status.MigrationCompleted = true
			memory.Status.MigrationTime = &metav1.Time{Time: time.Now()}
		}
	}
	
	return nil
}

func (r *SwarmMemoryStoreReconciler) handleDelete(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	
	// Check if finalizer is present
	if containsString(memory.GetFinalizers(), swarmMemoryFinalizer) {
		// Run cleanup logic
		logger.Info("Running SwarmMemory cleanup", "Name", memory.Name)
		
		// Create backup if configured
		if memory.Spec.BackupOnDelete {
			if err := r.createBackup(ctx, memory); err != nil {
				logger.Error(err, "Failed to create backup on delete")
				// Continue with deletion even if backup fails
			}
		}
		
		// Remove finalizer
		memory.SetFinalizers(removeString(memory.GetFinalizers(), swarmMemoryFinalizer))
		if err := r.Update(ctx, memory); err != nil {
			return ctrl.Result{}, err
		}
	}
	
	return ctrl.Result{}, nil
}

func (r *SwarmMemoryStoreReconciler) createBackup(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore) error {
	// Implementation would create a backup job
	// For now, just log
	logger := log.FromContext(ctx)
	logger.Info("Creating backup", "Memory", memory.Name)
	return nil
}

func (r *SwarmMemoryStoreReconciler) getDatabaseSize(ctx context.Context, memory *swarmv1alpha1.SwarmMemoryStore, namespace string) string {
	// In a real implementation, this would query the pod to get actual DB size
	// For now, return a placeholder
	return "0 MB"
}

func getEnhancedSchema() string {
	return `-- Enhanced SQLite schema for SwarmMemory
CREATE TABLE IF NOT EXISTS memory_store (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL,
    namespace TEXT NOT NULL,
    value TEXT NOT NULL,
    type TEXT DEFAULT 'json',
    metadata TEXT DEFAULT '{}',
    tags TEXT DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    access_count INTEGER DEFAULT 0,
    ttl INTEGER DEFAULT NULL,
    expires_at TIMESTAMP DEFAULT NULL,
    compressed BOOLEAN DEFAULT 0,
    size INTEGER DEFAULT 0,
    UNIQUE(key, namespace)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_namespace ON memory_store(namespace);
CREATE INDEX IF NOT EXISTS idx_expires_at ON memory_store(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tags ON memory_store(tags);
CREATE INDEX IF NOT EXISTS idx_created_at ON memory_store(created_at);
CREATE INDEX IF NOT EXISTS idx_accessed_at ON memory_store(accessed_at);

-- Trigger to update updated_at
CREATE TRIGGER IF NOT EXISTS update_timestamp 
AFTER UPDATE ON memory_store
BEGIN
    UPDATE memory_store SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Swarm-specific tables
CREATE TABLE IF NOT EXISTS swarm_agents (
    agent_id TEXT PRIMARY KEY,
    swarm_id TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT DEFAULT 'inactive',
    capabilities TEXT DEFAULT '[]',
    metadata TEXT DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_heartbeat TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS swarm_tasks (
    task_id TEXT PRIMARY KEY,
    swarm_id TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'pending',
    priority TEXT DEFAULT 'medium',
    assigned_agents TEXT DEFAULT '[]',
    result TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS swarm_patterns (
    pattern_id TEXT PRIMARY KEY,
    swarm_id TEXT NOT NULL,
    type TEXT NOT NULL,
    confidence REAL DEFAULT 0.0,
    data TEXT NOT NULL,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for swarm tables
CREATE INDEX IF NOT EXISTS idx_swarm_agents_swarm ON swarm_agents(swarm_id);
CREATE INDEX IF NOT EXISTS idx_swarm_tasks_swarm ON swarm_tasks(swarm_id);
CREATE INDEX IF NOT EXISTS idx_swarm_patterns_swarm ON swarm_patterns(swarm_id);
CREATE INDEX IF NOT EXISTS idx_swarm_patterns_confidence ON swarm_patterns(confidence DESC);
`
}

const swarmMemoryFinalizer = "swarm.claudeflow.io/memory-finalizer"

// Helper functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwarmMemoryStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swarmv1alpha1.SwarmMemoryStore{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}