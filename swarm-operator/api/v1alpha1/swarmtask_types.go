/*
Copyright 2025 Claude Flow Contributors.

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

// TaskPriority defines the priority level of a task
type TaskPriority string

const (
	LowPriority      TaskPriority = "low"
	MediumPriority   TaskPriority = "medium"
	HighPriority     TaskPriority = "high"
	CriticalPriority TaskPriority = "critical"
)

// TaskStrategy defines the execution strategy
type TaskStrategy string

const (
	ParallelStrategy   TaskStrategy = "parallel"
	SequentialStrategy TaskStrategy = "sequential"
	AdaptiveStrategy   TaskStrategy = "adaptive"
	BalancedStrategy   TaskStrategy = "balanced"
)

// SwarmTaskSpec defines the desired state of SwarmTask
type SwarmTaskSpec struct {
	// SwarmCluster reference
	SwarmCluster string `json:"swarmCluster"`

	// Description of the task
	Description string `json:"description"`

	// Type of task (e.g., "research", "development", "analysis")
	Type string `json:"type"`

	// Priority of the task
	// +kubebuilder:validation:Enum=low;medium;high;critical
	// +kubebuilder:default=medium
	Priority TaskPriority `json:"priority,omitempty"`

	// Strategy for task execution
	// +kubebuilder:validation:Enum=parallel;sequential;adaptive;balanced
	// +kubebuilder:default=adaptive
	Strategy TaskStrategy `json:"strategy,omitempty"`

	// RequiredCapabilities that agents must have to process this task
	RequiredCapabilities []string `json:"requiredCapabilities,omitempty"`

	// PreferredAgentTypes for this task
	PreferredAgentTypes []AgentType `json:"preferredAgentTypes,omitempty"`

	// Subtasks that compose this task
	Subtasks []SubtaskSpec `json:"subtasks,omitempty"`

	// Dependencies between subtasks
	Dependencies []TaskDependency `json:"dependencies,omitempty"`

	// Parameters for task execution
	Parameters map[string]string `json:"parameters,omitempty"`

	// Timeout in seconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=300
	Timeout int32 `json:"timeout,omitempty"`

	// RetryPolicy for failed tasks
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	// ResultStorage configuration
	ResultStorage ResultStorageSpec `json:"resultStorage,omitempty"`

	// Repositories is a list of GitHub repositories this task needs access to
	// Format: owner/repo (e.g., "claude-flow/swarm-operator")
	Repositories []string `json:"repositories,omitempty"`

	// GitHubApp configuration for repository access
	GitHubApp *GitHubAppConfig `json:"githubApp,omitempty"`

	// Namespace to run this task in (defaults based on task type)
	Namespace string `json:"namespace,omitempty"`
}

// SubtaskSpec defines a subtask
type SubtaskSpec struct {
	// Name of the subtask
	Name string `json:"name"`

	// Type of subtask
	Type string `json:"type"`

	// Description of what this subtask does
	Description string `json:"description,omitempty"`

	// RequiredCapabilities for this subtask
	RequiredCapabilities []string `json:"requiredCapabilities,omitempty"`

	// EstimatedDuration in seconds
	EstimatedDuration int32 `json:"estimatedDuration,omitempty"`

	// Parameters specific to this subtask
	Parameters map[string]string `json:"parameters,omitempty"`
}

// TaskDependency defines dependencies between subtasks
type TaskDependency struct {
	// From subtask name
	From string `json:"from"`

	// To subtask name
	To string `json:"to"`

	// Type of dependency
	// +kubebuilder:validation:Enum=completion;data;conditional
	// +kubebuilder:default=completion
	Type string `json:"type,omitempty"`

	// Condition for conditional dependencies
	Condition string `json:"condition,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	// MaxRetries allowed
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=3
	MaxRetries int32 `json:"maxRetries"`

	// BackoffSeconds between retries
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=30
	BackoffSeconds int32 `json:"backoffSeconds,omitempty"`

	// BackoffMultiplier for exponential backoff
	// +kubebuilder:default=2
	BackoffMultiplier float64 `json:"backoffMultiplier,omitempty"`
}

// GitHubAppConfig defines GitHub App configuration for repository access
type GitHubAppConfig struct {
	// AppID is the GitHub App ID
	AppID int64 `json:"appID"`

	// PrivateKeyRef references a Secret containing the GitHub App private key
	PrivateKeyRef SecretKeyRef `json:"privateKeyRef"`

	// InstallationID for the GitHub App (optional, will be auto-discovered if not provided)
	InstallationID int64 `json:"installationID,omitempty"`

	// TokenTTL is the duration for which generated tokens are valid
	// +kubebuilder:default="1h"
	TokenTTL string `json:"tokenTTL,omitempty"`
}

// SecretKeyRef references a key in a Secret
type SecretKeyRef struct {
	// Name of the Secret
	Name string `json:"name"`

	// Key within the Secret
	Key string `json:"key"`

	// Namespace of the Secret (defaults to same namespace as the resource)
	Namespace string `json:"namespace,omitempty"`
}

// ResultStorageSpec defines where to store results
type ResultStorageSpec struct {
	// Type of storage
	// +kubebuilder:validation:Enum=configmap;secret;s3;pvc
	// +kubebuilder:default=configmap
	Type string `json:"type"`

	// Name of the storage resource
	Name string `json:"name,omitempty"`

	// Path within the storage
	Path string `json:"path,omitempty"`

	// TTL for result storage in seconds
	TTL int32 `json:"ttl,omitempty"`
}

// SwarmTaskStatus defines the observed state of SwarmTask
type SwarmTaskStatus struct {
	// Phase of the task
	// +kubebuilder:validation:Enum=Pending;Scheduled;Running;Completed;Failed;Cancelled
	Phase string `json:"phase,omitempty"`

	// StartTime when the task started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime when the task completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// AssignedAgents working on this task
	AssignedAgents []AssignedAgent `json:"assignedAgents,omitempty"`

	// SubtaskStatuses for each subtask
	SubtaskStatuses []SubtaskStatus `json:"subtaskStatuses,omitempty"`

	// Progress percentage (0-100)
	Progress int32 `json:"progress"`

	// Result of the task execution
	Result *TaskResult `json:"result,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// RetryCount tracks retry attempts
	RetryCount int32 `json:"retryCount"`

	// Message provides additional information
	Message string `json:"message,omitempty"`
}

// AssignedAgent represents an agent assigned to the task
type AssignedAgent struct {
	// Name of the agent
	Name string `json:"name"`

	// Type of the agent
	Type AgentType `json:"type"`

	// Subtasks assigned to this agent
	AssignedSubtasks []string `json:"assignedSubtasks,omitempty"`

	// Status of this agent's work
	Status string `json:"status,omitempty"`
}

// SubtaskStatus represents the status of a subtask
type SubtaskStatus struct {
	// Name of the subtask
	Name string `json:"name"`

	// Phase of the subtask
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed;Skipped
	Phase string `json:"phase"`

	// AssignedAgent for this subtask
	AssignedAgent string `json:"assignedAgent,omitempty"`

	// StartTime of the subtask
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime of the subtask
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Progress percentage (0-100)
	Progress int32 `json:"progress"`

	// Result of the subtask
	Result map[string]string `json:"result,omitempty"`

	// Error message if failed
	Error string `json:"error,omitempty"`
}

// TaskResult contains the final result of the task
type TaskResult struct {
	// Success indicates if the task completed successfully
	Success bool `json:"success"`

	// Data contains the result data
	Data map[string]string `json:"data,omitempty"`

	// Summary of the task execution
	Summary string `json:"summary,omitempty"`

	// Metrics collected during execution
	Metrics TaskMetrics `json:"metrics,omitempty"`

	// StorageRef points to where full results are stored
	StorageRef string `json:"storageRef,omitempty"`
}

// TaskMetrics contains execution metrics
type TaskMetrics struct {
	// ExecutionTime in seconds
	ExecutionTime int64 `json:"executionTime"`

	// AgentsUsed count
	AgentsUsed int32 `json:"agentsUsed"`

	// SubtasksCompleted count
	SubtasksCompleted int32 `json:"subtasksCompleted"`

	// TokensConsumed if applicable
	TokensConsumed int64 `json:"tokensConsumed,omitempty"`

	// CostEstimate if applicable
	CostEstimate float64 `json:"costEstimate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Swarm",type="string",JSONPath=".spec.swarmCluster"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Priority",type="string",JSONPath=".spec.priority"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Progress",type="integer",JSONPath=".status.progress"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SwarmTask is the Schema for the swarmtasks API
type SwarmTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwarmTaskSpec   `json:"spec,omitempty"`
	Status SwarmTaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SwarmTaskList contains a list of SwarmTask
type SwarmTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwarmTask `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwarmTask{}, &SwarmTaskList{})
}
