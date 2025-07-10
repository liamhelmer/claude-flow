package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemoryType defines the type of memory entry
type MemoryType string

const (
	MemoryTypeKnowledge   MemoryType = "knowledge"
	MemoryTypeExperience  MemoryType = "experience"
	MemoryTypePattern     MemoryType = "pattern"
	MemoryTypeDecision    MemoryType = "decision"
	MemoryTypeCheckpoint  MemoryType = "checkpoint"
)

// SwarmMemorySpec defines the desired state of SwarmMemory
type SwarmMemorySpec struct {
	// ClusterRef references the parent SwarmCluster
	ClusterRef string `json:"clusterRef"`

	// Namespace for memory isolation
	Namespace string `json:"namespace"`

	// Type of memory entry
	Type MemoryType `json:"type,omitempty"`

	// Key for the memory entry
	Key string `json:"key"`

	// Value stored in memory (base64 encoded for binary data)
	Value string `json:"value"`

	// TTL time-to-live in seconds (0 = permanent)
	TTL int32 `json:"ttl,omitempty"`

	// Tags for categorization and search
	Tags []string `json:"tags,omitempty"`

	// AccessPattern expected (sequential, random, frequent)
	AccessPattern string `json:"accessPattern,omitempty"`

	// Compression enabled for this entry
	Compression bool `json:"compression,omitempty"`

	// Encryption enabled for sensitive data
	Encryption bool `json:"encryption,omitempty"`

	// SharedWith specific agents (empty = all agents)
	SharedWith []string `json:"sharedWith,omitempty"`

	// Priority for cache retention (0-100)
	Priority int32 `json:"priority,omitempty"`
}

// SwarmMemoryStatus defines the observed state of SwarmMemory
type SwarmMemoryStatus struct {
	// Phase of the memory entry
	Phase string `json:"phase,omitempty"`

	// Size of the stored value in bytes
	Size int64 `json:"size,omitempty"`

	// CompressedSize if compression is enabled
	CompressedSize int64 `json:"compressedSize,omitempty"`

	// AccessCount number of times accessed
	AccessCount int64 `json:"accessCount,omitempty"`

	// LastAccessTime
	LastAccessTime *metav1.Time `json:"lastAccessTime,omitempty"`

	// CreatedBy agent that created this entry
	CreatedBy string `json:"createdBy,omitempty"`

	// ModifiedBy agent that last modified this entry
	ModifiedBy string `json:"modifiedBy,omitempty"`

	// ExpiresAt calculated expiration time
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Replicas count for durability
	Replicas int32 `json:"replicas,omitempty"`

	// StorageBackend where this is stored
	StorageBackend string `json:"storageBackend,omitempty"`

	// Conditions for the memory entry
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=sm
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Key",type=string,JSONPath=`.spec.key`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Size",type=integer,JSONPath=`.status.size`
// +kubebuilder:printcolumn:name="Accesses",type=integer,JSONPath=`.status.accessCount`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef`

// SwarmMemory is the Schema for the swarmmemories API
type SwarmMemory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwarmMemorySpec   `json:"spec,omitempty"`
	Status SwarmMemoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SwarmMemoryList contains a list of SwarmMemory
type SwarmMemoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwarmMemory `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwarmMemory{}, &SwarmMemoryList{})
}