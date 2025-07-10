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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SwarmTopology defines the communication topology for the swarm
type SwarmTopology string

const (
	// MeshTopology allows all agents to communicate with each other
	MeshTopology SwarmTopology = "mesh"
	// HierarchicalTopology creates a tree-like structure with parent-child relationships
	HierarchicalTopology SwarmTopology = "hierarchical"
	// RingTopology arranges agents in a circular communication pattern
	RingTopology SwarmTopology = "ring"
	// StarTopology has a central coordinator with all agents connecting to it
	StarTopology SwarmTopology = "star"
)

// SwarmClusterSpec defines the desired state of SwarmCluster
type SwarmClusterSpec struct {
	// Topology defines the communication pattern between agents
	// +kubebuilder:validation:Enum=mesh;hierarchical;ring;star
	// +kubebuilder:default=mesh
	Topology SwarmTopology `json:"topology"`

	// MaxAgents is the maximum number of agents in the swarm
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=5
	MaxAgents int32 `json:"maxAgents"`

	// MinAgents is the minimum number of agents in the swarm
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=1
	MinAgents int32 `json:"minAgents,omitempty"`

	// Strategy defines how agents are selected and distributed
	// +kubebuilder:validation:Enum=balanced;specialized;adaptive
	// +kubebuilder:default=balanced
	Strategy string `json:"strategy,omitempty"`

	// AgentTemplate defines the template for creating agents
	AgentTemplate AgentTemplateSpec `json:"agentTemplate,omitempty"`

	// TaskDistribution defines how tasks are distributed among agents
	TaskDistribution TaskDistributionSpec `json:"taskDistribution,omitempty"`

	// AutoScaling defines auto-scaling behavior
	AutoScaling *AutoScalingSpec `json:"autoScaling,omitempty"`
}

// AgentTemplateSpec defines the template for creating agents
type AgentTemplateSpec struct {
	// Capabilities that agents in this swarm should have
	Capabilities []string `json:"capabilities,omitempty"`

	// Resources defines resource requirements for agents
	Resources ResourceRequirements `json:"resources,omitempty"`

	// CognitivePatterns defines the thinking patterns for agents
	CognitivePatterns []string `json:"cognitivePatterns,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	// CPU requirement in millicores
	CPU string `json:"cpu,omitempty"`

	// Memory requirement
	Memory string `json:"memory,omitempty"`

	// Storage requirement
	Storage string `json:"storage,omitempty"`
}

// TaskDistributionSpec defines how tasks are distributed
type TaskDistributionSpec struct {
	// Algorithm for task distribution
	// +kubebuilder:validation:Enum=round-robin;least-loaded;capability-based;priority-based
	// +kubebuilder:default=capability-based
	Algorithm string `json:"algorithm"`

	// MaxTasksPerAgent limits tasks per agent
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=10
	MaxTasksPerAgent int32 `json:"maxTasksPerAgent,omitempty"`

	// TaskTimeout in seconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=300
	TaskTimeout int32 `json:"taskTimeout,omitempty"`
}

// AutoScalingSpec defines auto-scaling configuration
type AutoScalingSpec struct {
	// Enabled indicates if auto-scaling is enabled
	Enabled bool `json:"enabled"`

	// Metrics to use for scaling decisions
	Metrics []ScalingMetric `json:"metrics,omitempty"`

	// ScaleUpThreshold percentage (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=80
	ScaleUpThreshold int32 `json:"scaleUpThreshold,omitempty"`

	// ScaleDownThreshold percentage (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=20
	ScaleDownThreshold int32 `json:"scaleDownThreshold,omitempty"`
}

// ScalingMetric defines a metric for auto-scaling
type ScalingMetric struct {
	// Type of metric
	// +kubebuilder:validation:Enum=cpu;memory;task-queue;custom
	Type string `json:"type"`

	// Target value for the metric
	Target string `json:"target"`
}

// SwarmClusterStatus defines the observed state of SwarmCluster
type SwarmClusterStatus struct {
	// Phase represents the current phase of the swarm
	// +kubebuilder:validation:Enum=Pending;Initializing;Running;Scaling;Terminating;Failed
	Phase string `json:"phase,omitempty"`

	// ActiveAgents is the current number of active agents
	ActiveAgents int32 `json:"activeAgents"`

	// ReadyAgents is the number of agents ready to process tasks
	ReadyAgents int32 `json:"readyAgents"`

	// Conditions represent the latest available observations of the swarm's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastScaleTime is the last time the swarm was scaled
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// TaskStats contains task execution statistics
	TaskStats TaskStatistics `json:"taskStats,omitempty"`

	// TopologyStatus contains topology-specific status information
	TopologyStatus map[string]string `json:"topologyStatus,omitempty"`
}

// TaskStatistics contains task execution statistics
type TaskStatistics struct {
	// Total number of tasks processed
	TotalTasks int64 `json:"totalTasks"`

	// Number of successful tasks
	SuccessfulTasks int64 `json:"successfulTasks"`

	// Number of failed tasks
	FailedTasks int64 `json:"failedTasks"`

	// Average task completion time in milliseconds
	AverageCompletionTime int64 `json:"averageCompletionTime,omitempty"`

	// Current queue size
	QueueSize int32 `json:"queueSize"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.maxAgents,statuspath=.status.activeAgents
// +kubebuilder:printcolumn:name="Topology",type="string",JSONPath=".spec.topology"
// +kubebuilder:printcolumn:name="Active",type="integer",JSONPath=".status.activeAgents"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyAgents"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SwarmCluster is the Schema for the swarmclusters API
type SwarmCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwarmClusterSpec   `json:"spec,omitempty"`
	Status SwarmClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SwarmClusterList contains a list of SwarmCluster
type SwarmClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwarmCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwarmCluster{}, &SwarmClusterList{})
}