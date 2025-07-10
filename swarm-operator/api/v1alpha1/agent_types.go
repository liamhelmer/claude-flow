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

// AgentType defines the type of agent
type AgentType string

const (
	ResearcherAgent   AgentType = "researcher"
	CoderAgent        AgentType = "coder"
	AnalystAgent      AgentType = "analyst"
	OptimizerAgent    AgentType = "optimizer"
	CoordinatorAgent  AgentType = "coordinator"
	ArchitectAgent    AgentType = "architect"
	TesterAgent       AgentType = "tester"
	ReviewerAgent     AgentType = "reviewer"
	DocumenterAgent   AgentType = "documenter"
	MonitorAgent      AgentType = "monitor"
	SpecialistAgent   AgentType = "specialist"
)

// CognitivePattern defines thinking patterns for agents
type CognitivePattern string

const (
	ConvergentPattern  CognitivePattern = "convergent"
	DivergentPattern   CognitivePattern = "divergent"
	LateralPattern     CognitivePattern = "lateral"
	SystemsPattern     CognitivePattern = "systems"
	CriticalPattern    CognitivePattern = "critical"
	AdaptivePattern    CognitivePattern = "adaptive"
)

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	// Type defines the agent type
	// +kubebuilder:validation:Enum=researcher;coder;analyst;optimizer;coordinator;architect;tester;reviewer;documenter;monitor;specialist
	Type AgentType `json:"type"`

	// SwarmCluster reference
	SwarmCluster string `json:"swarmCluster"`

	// Capabilities that this agent has
	Capabilities []string `json:"capabilities,omitempty"`

	// CognitivePattern defines the thinking pattern
	// +kubebuilder:validation:Enum=convergent;divergent;lateral;systems;critical;adaptive
	// +kubebuilder:default=adaptive
	CognitivePattern CognitivePattern `json:"cognitivePattern,omitempty"`

	// Resources defines resource requirements
	Resources ResourceRequirements `json:"resources,omitempty"`

	// TaskAffinity defines task preferences
	TaskAffinity []TaskAffinityRule `json:"taskAffinity,omitempty"`

	// CommunicationEndpoints for inter-agent communication
	CommunicationEndpoints CommunicationSpec `json:"communication,omitempty"`
}

// TaskAffinityRule defines task affinity rules
type TaskAffinityRule struct {
	// TaskType that this rule applies to
	TaskType string `json:"taskType"`

	// Priority for this task type (0-100)
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Priority int32 `json:"priority"`

	// Required capabilities for this task
	RequiredCapabilities []string `json:"requiredCapabilities,omitempty"`
}

// CommunicationSpec defines communication endpoints
type CommunicationSpec struct {
	// Protocol for communication
	// +kubebuilder:validation:Enum=grpc;http;websocket
	// +kubebuilder:default=grpc
	Protocol string `json:"protocol,omitempty"`

	// Port for communication
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8080
	Port int32 `json:"port,omitempty"`

	// Peers that this agent can communicate with
	Peers []string `json:"peers,omitempty"`

	// BroadcastEnabled allows broadcasting to all peers
	BroadcastEnabled bool `json:"broadcastEnabled,omitempty"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// Phase represents the current phase of the agent
	// +kubebuilder:validation:Enum=Pending;Initializing;Ready;Busy;Terminating;Failed
	Phase string `json:"phase,omitempty"`

	// CurrentTasks being processed
	CurrentTasks []TaskReference `json:"currentTasks,omitempty"`

	// CompletedTasks count
	CompletedTasks int64 `json:"completedTasks"`

	// FailedTasks count
	FailedTasks int64 `json:"failedTasks"`

	// LastHeartbeat time
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Metrics contains agent performance metrics
	Metrics AgentMetrics `json:"metrics,omitempty"`

	// CommunicationStatus with peers
	CommunicationStatus map[string]PeerStatus `json:"communicationStatus,omitempty"`
}

// TaskReference references a task being processed
type TaskReference struct {
	// Name of the task
	Name string `json:"name"`

	// Type of the task
	Type string `json:"type"`

	// StartTime when the task started
	StartTime metav1.Time `json:"startTime"`

	// Progress percentage (0-100)
	Progress int32 `json:"progress,omitempty"`
}

// AgentMetrics contains performance metrics
type AgentMetrics struct {
	// CPU usage percentage
	CPUUsage float64 `json:"cpuUsage,omitempty"`

	// Memory usage in bytes
	MemoryUsage int64 `json:"memoryUsage,omitempty"`

	// Task throughput per minute
	TaskThroughput float64 `json:"taskThroughput,omitempty"`

	// Average task completion time in ms
	AverageTaskTime int64 `json:"averageTaskTime,omitempty"`

	// Success rate percentage
	SuccessRate float64 `json:"successRate,omitempty"`
}

// PeerStatus represents communication status with a peer
type PeerStatus struct {
	// Connected indicates if peer is connected
	Connected bool `json:"connected"`

	// LastContact time with the peer
	LastContact *metav1.Time `json:"lastContact,omitempty"`

	// Latency in milliseconds
	Latency int32 `json:"latency,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Swarm",type="string",JSONPath=".spec.swarmCluster"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Tasks",type="integer",JSONPath=".status.completedTasks"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}