package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AgentType defines the type of agent
type AgentType string

const (
	AgentTypeCoordinator AgentType = "coordinator"
	AgentTypeResearcher  AgentType = "researcher"
	AgentTypeCoder       AgentType = "coder"
	AgentTypeAnalyst     AgentType = "analyst"
	AgentTypeTester      AgentType = "tester"
	AgentTypeReviewer    AgentType = "reviewer"
	AgentTypeOptimizer   AgentType = "optimizer"
	AgentTypeDocumenter  AgentType = "documenter"
	AgentTypeMonitor     AgentType = "monitor"
	AgentTypeSpecialist  AgentType = "specialist"
	AgentTypeArchitect   AgentType = "architect"
)

// AgentStatus defines the agent operational status
type AgentStatus string

const (
	AgentStatusPending     AgentStatus = "pending"
	AgentStatusInitializing AgentStatus = "initializing"
	AgentStatusReady       AgentStatus = "ready"
	AgentStatusBusy        AgentStatus = "busy"
	AgentStatusIdle        AgentStatus = "idle"
	AgentStatusTerminating AgentStatus = "terminating"
	AgentStatusError       AgentStatus = "error"
)

// CognitivePattern defines agent thinking patterns
type CognitivePattern string

const (
	PatternConvergent  CognitivePattern = "convergent"
	PatternDivergent   CognitivePattern = "divergent"
	PatternLateral     CognitivePattern = "lateral"
	PatternSystems     CognitivePattern = "systems"
	PatternCritical    CognitivePattern = "critical"
	PatternAbstract    CognitivePattern = "abstract"
	PatternAdaptive    CognitivePattern = "adaptive"
)

// SwarmAgentSpec defines the desired state of SwarmAgent
type SwarmAgentSpec struct {
	// Type of the agent
	Type AgentType `json:"type"`

	// ClusterRef references the parent SwarmCluster
	ClusterRef string `json:"clusterRef"`

	// Capabilities of the agent
	Capabilities []string `json:"capabilities,omitempty"`

	// CognitivePattern for the agent
	CognitivePattern CognitivePattern `json:"cognitivePattern,omitempty"`

	// Priority for task assignment (0-100)
	Priority int32 `json:"priority,omitempty"`

	// MaxConcurrentTasks this agent can handle
	MaxConcurrentTasks int32 `json:"maxConcurrentTasks,omitempty"`

	// Specialization areas for the agent
	Specialization []string `json:"specialization,omitempty"`

	// Resources for the agent pod
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Image override for this specific agent
	Image string `json:"image,omitempty"`

	// Environment variables for the agent
	Environment []EnvVar `json:"environment,omitempty"`

	// HiveMindRole in the collective
	HiveMindRole string `json:"hiveMindRole,omitempty"`

	// MemoryAllocation for agent-specific memory
	MemoryAllocation string `json:"memoryAllocation,omitempty"`

	// NeuralModels assigned to this agent
	NeuralModels []string `json:"neuralModels,omitempty"`

	// GitHubTokenSecret references the secret containing the GitHub token for this agent
	GitHubTokenSecret string `json:"githubTokenSecret,omitempty"`

	// AllowedRepositories lists the repositories this agent can access
	AllowedRepositories []string `json:"allowedRepositories,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

// EnvVarSource represents a source for an environment variable
type EnvVarSource struct {
	SecretKeyRef    *SecretKeySelector    `json:"secretKeyRef,omitempty"`
	ConfigMapKeyRef *ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
}

// SecretKeySelector selects a key from a Secret
type SecretKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ConfigMapKeySelector selects a key from a ConfigMap
type ConfigMapKeySelector struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// SwarmAgentStatus defines the observed state of SwarmAgent
type SwarmAgentStatus struct {
	// Status of the agent
	Status AgentStatus `json:"status,omitempty"`

	// PodName running this agent
	PodName string `json:"podName,omitempty"`

	// NodeName where the agent is running
	NodeName string `json:"nodeName,omitempty"`

	// AssignedTasks currently being processed
	AssignedTasks []string `json:"assignedTasks,omitempty"`

	// CompletedTasks count
	CompletedTasks int32 `json:"completedTasks,omitempty"`

	// FailedTasks count
	FailedTasks int32 `json:"failedTasks,omitempty"`

	// Utilization percentage (0-100)
	Utilization int32 `json:"utilization,omitempty"`

	// LastTaskTime when the agent last processed a task
	LastTaskTime *metav1.Time `json:"lastTaskTime,omitempty"`

	// HiveMindConnected status
	HiveMindConnected bool `json:"hiveMindConnected,omitempty"`

	// MemoryUsage by this agent
	MemoryUsage string `json:"memoryUsage,omitempty"`

	// Performance metrics
	Performance PerformanceMetrics `json:"performance,omitempty"`

	// Conditions for the agent
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// StartTime when the agent started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// GitHubTokenStatus contains information about the GitHub token
	GitHubTokenStatus *GitHubTokenStatus `json:"githubTokenStatus,omitempty"`
}

// GitHubTokenStatus contains GitHub token status information
type GitHubTokenStatus struct {
	// Created indicates if the token has been created
	Created bool `json:"created"`

	// ExpiresAt is when the token expires
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// Repositories that the token has access to
	Repositories []string `json:"repositories,omitempty"`

	// Permissions granted to the token
	Permissions map[string]string `json:"permissions,omitempty"`

	// LastRotated is when the token was last rotated
	LastRotated *metav1.Time `json:"lastRotated,omitempty"`
}

// PerformanceMetrics tracks agent performance
type PerformanceMetrics struct {
	// TasksPerMinute processing rate
	TasksPerMinute float64 `json:"tasksPerMinute,omitempty"`

	// AverageTaskDuration in seconds
	AverageTaskDuration float64 `json:"averageTaskDuration,omitempty"`

	// SuccessRate percentage (0-100)
	SuccessRate float64 `json:"successRate,omitempty"`

	// ResponseTime average in milliseconds
	ResponseTime float64 `json:"responseTime,omitempty"`

	// CPUUsage percentage
	CPUUsage float64 `json:"cpuUsage,omitempty"`

	// MemoryUsage percentage
	MemoryUsage float64 `json:"memoryUsage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=sa
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
// +kubebuilder:printcolumn:name="Tasks",type=integer,JSONPath=`.status.completedTasks`
// +kubebuilder:printcolumn:name="Utilization",type=integer,JSONPath=`.status.utilization`
// +kubebuilder:printcolumn:name="Cluster",type=string,JSONPath=`.spec.clusterRef`

// SwarmAgent is the Schema for the swarmagents API
type SwarmAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwarmAgentSpec   `json:"spec,omitempty"`
	Status SwarmAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SwarmAgentList contains a list of SwarmAgent
type SwarmAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwarmAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwarmAgent{}, &SwarmAgentList{})
}