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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SwarmMemoryStoreSpec defines the desired state of SwarmMemoryStore
type SwarmMemoryStoreSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type is the memory backend type (now supports "sqlite" as primary)
	// +kubebuilder:validation:Enum=sqlite;redis;etcd;embedded
	// +kubebuilder:default=sqlite
	Type string `json:"type"`

	// SwarmID identifies the swarm this memory belongs to
	SwarmID string `json:"swarmId"`

	// SwarmClusterRef references the SwarmCluster this memory is for
	SwarmClusterRef string `json:"swarmClusterRef,omitempty"`

	// Namespace to deploy the memory service in (defaults based on cluster config)
	Namespace string `json:"namespace,omitempty"`

	// StorageSize is the persistent storage size for SQLite
	// +kubebuilder:default="10Gi"
	StorageSize string `json:"storageSize,omitempty"`

	// StorageClass for the PVC
	StorageClass string `json:"storageClass,omitempty"`

	// Version of the swarm-memory image to use
	// +kubebuilder:default="latest"
	Version string `json:"version,omitempty"`

	// CacheSize is the maximum number of entries to cache in memory
	// +kubebuilder:default=1000
	CacheSize int `json:"cacheSize,omitempty"`

	// CacheMemoryMB is the maximum memory to use for caching
	// +kubebuilder:default=50
	CacheMemoryMB int `json:"cacheMemoryMB,omitempty"`

	// CompressionThreshold is the size threshold for compression (bytes)
	// +kubebuilder:default=10240
	CompressionThreshold int `json:"compressionThreshold,omitempty"`

	// GCInterval is the garbage collection interval
	// +kubebuilder:default="5m"
	GCInterval string `json:"gcInterval,omitempty"`

	// BackupInterval for automatic backups
	BackupInterval string `json:"backupInterval,omitempty"`

	// BackupRetention is how many backups to keep
	// +kubebuilder:default=7
	BackupRetention int `json:"backupRetention,omitempty"`

	// MigrateFromLegacy enables migration from old memory systems
	MigrateFromLegacy bool `json:"migrateFromLegacy,omitempty"`

	// LegacyDataPVC is the PVC containing legacy data to migrate
	LegacyDataPVC string `json:"legacyDataPVC,omitempty"`

	// BackupOnDelete creates a backup before deletion
	// +kubebuilder:default=true
	BackupOnDelete bool `json:"backupOnDelete,omitempty"`

	// MCPMode enables MCP-specific features
	// +kubebuilder:default=true
	MCPMode bool `json:"mcpMode,omitempty"`

	// EnableWAL enables Write-Ahead Logging for SQLite
	// +kubebuilder:default=true
	EnableWAL bool `json:"enableWAL,omitempty"`

	// EnableVacuum enables automatic database vacuuming
	// +kubebuilder:default=true
	EnableVacuum bool `json:"enableVacuum,omitempty"`
}

// SwarmMemoryStoreStatus defines the observed state of SwarmMemoryStore
type SwarmMemoryStoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase represents the current phase of the memory system
	// +kubebuilder:validation:Enum=Initializing;Ready;Error;Migrating;BackingUp
	Phase string `json:"phase,omitempty"`

	// StorageReady indicates if the persistent storage is ready
	StorageReady bool `json:"storageReady,omitempty"`

	// DatabaseSize shows the current database size
	DatabaseSize string `json:"databaseSize,omitempty"`

	// EntryCount is the total number of entries stored
	EntryCount int64 `json:"entryCount,omitempty"`

	// AgentCount is the number of registered agents
	AgentCount int64 `json:"agentCount,omitempty"`

	// TaskCount is the number of tracked tasks
	TaskCount int64 `json:"taskCount,omitempty"`

	// PatternCount is the number of learned patterns
	PatternCount int64 `json:"patternCount,omitempty"`

	// CacheHitRate shows the cache effectiveness
	CacheHitRate string `json:"cacheHitRate,omitempty"`

	// LastBackup timestamp of the last successful backup
	LastBackup *metav1.Time `json:"lastBackup,omitempty"`

	// MigrationCompleted indicates if migration from legacy is done
	MigrationCompleted bool `json:"migrationCompleted,omitempty"`

	// MigrationTime when the migration completed
	MigrationTime *metav1.Time `json:"migrationTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Endpoints for accessing the memory service
	Endpoints SwarmMemoryEndpoints `json:"endpoints,omitempty"`
}

// SwarmMemoryEndpoints contains the service endpoints
type SwarmMemoryEndpoints struct {
	// GRPC endpoint for direct access
	GRPC string `json:"grpc,omitempty"`

	// HTTP endpoint for REST API (if enabled)
	HTTP string `json:"http,omitempty"`

	// Metrics endpoint for Prometheus
	Metrics string `json:"metrics,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=sms
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="SwarmID",type=string,JSONPath=`.spec.swarmId`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Storage",type=string,JSONPath=`.status.databaseSize`
//+kubebuilder:printcolumn:name="Entries",type=integer,JSONPath=`.status.entryCount`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// SwarmMemoryStore is the Schema for the swarmmemorystores API
type SwarmMemoryStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwarmMemoryStoreSpec   `json:"spec,omitempty"`
	Status SwarmMemoryStoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SwarmMemoryStoreList contains a list of SwarmMemoryStore
type SwarmMemoryStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwarmMemoryStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwarmMemoryStore{}, &SwarmMemoryStoreList{})
}