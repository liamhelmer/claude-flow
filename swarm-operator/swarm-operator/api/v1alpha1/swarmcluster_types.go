package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SwarmTopology defines the coordination topology
type SwarmTopology string

const (
	TopologyMesh         SwarmTopology = "mesh"
	TopologyHierarchical SwarmTopology = "hierarchical"
	TopologyRing         SwarmTopology = "ring"
	TopologyStar         SwarmTopology = "star"
)

// QueenMode defines the coordination mode
type QueenMode string

const (
	QueenModeCentralized QueenMode = "centralized"
	QueenModeDistributed QueenMode = "distributed"
)

// ExecutionStrategy defines how tasks are executed
type ExecutionStrategy string

const (
	StrategyParallel   ExecutionStrategy = "parallel"
	StrategySequential ExecutionStrategy = "sequential"
	StrategyAdaptive   ExecutionStrategy = "adaptive"
	StrategyConsensus  ExecutionStrategy = "consensus"
)

// SwarmClusterSpec defines the desired state of SwarmCluster
type SwarmClusterSpec struct {
	// Topology defines the swarm coordination topology
	Topology SwarmTopology `json:"topology"`

	// QueenMode defines centralized or distributed coordination
	QueenMode QueenMode `json:"queenMode,omitempty"`

	// Strategy defines the default execution strategy
	Strategy ExecutionStrategy `json:"strategy,omitempty"`

	// ConsensusThreshold for decision making (0.0-1.0)
	ConsensusThreshold float64 `json:"consensusThreshold,omitempty"`

	// HiveMind configuration
	HiveMind HiveMindSpec `json:"hiveMind,omitempty"`

	// Autoscaling configuration
	Autoscaling AutoscalingSpec `json:"autoscaling,omitempty"`

	// AgentTemplate defines the template for spawning agents
	AgentTemplate AgentTemplateSpec `json:"agentTemplate,omitempty"`

	// Memory configuration for distributed memory
	Memory MemorySpec `json:"memory,omitempty"`

	// Neural configuration for ML capabilities
	Neural NeuralSpec `json:"neural,omitempty"`

	// Monitoring configuration
	Monitoring MonitoringSpec `json:"monitoring,omitempty"`

	// NamespaceConfig defines which namespaces to use for different agent types
	NamespaceConfig NamespaceConfig `json:"namespaceConfig,omitempty"`

	// GitHubApp configuration for the swarm
	GitHubApp *GitHubAppConfig `json:"githubApp,omitempty"`
}

// HiveMindSpec defines hive-mind configuration
type HiveMindSpec struct {
	// Enabled activates hive-mind coordination
	Enabled bool `json:"enabled,omitempty"`

	// DatabaseSize for hive-mind storage
	DatabaseSize string `json:"databaseSize,omitempty"`

	// SyncInterval for agent synchronization
	SyncInterval string `json:"syncInterval,omitempty"`

	// BackupEnabled for hive-mind state
	BackupEnabled bool `json:"backupEnabled,omitempty"`

	// BackupInterval for automatic backups
	BackupInterval string `json:"backupInterval,omitempty"`
}

// AutoscalingSpec defines autoscaling configuration
type AutoscalingSpec struct {
	// Enabled activates autoscaling
	Enabled bool `json:"enabled,omitempty"`

	// MinAgents minimum number of agents
	MinAgents int32 `json:"minAgents,omitempty"`

	// MaxAgents maximum number of agents
	MaxAgents int32 `json:"maxAgents,omitempty"`

	// TargetUtilization triggers scaling (0-100)
	TargetUtilization int32 `json:"targetUtilization,omitempty"`

	// ScaleUpThreshold for adding agents
	ScaleUpThreshold int32 `json:"scaleUpThreshold,omitempty"`

	// ScaleDownThreshold for removing agents
	ScaleDownThreshold int32 `json:"scaleDownThreshold,omitempty"`

	// StabilizationWindow prevents flapping
	StabilizationWindow string `json:"stabilizationWindow,omitempty"`

	// TopologyRatios maintains agent type ratios
	TopologyRatios map[string]int32 `json:"topologyRatios,omitempty"`

	// Metrics defines custom metrics for scaling
	Metrics []AutoscalingMetric `json:"metrics,omitempty"`
}

// AutoscalingMetric defines a custom metric for autoscaling
type AutoscalingMetric struct {
	// Type of metric (cpu, memory, queue, custom)
	Type string `json:"type"`

	// Name of the metric
	Name string `json:"name,omitempty"`

	// Target value for the metric
	Target string `json:"target"`
}

// AgentTemplateSpec defines the template for agents
type AgentTemplateSpec struct {
	// Image for agent containers
	Image string `json:"image,omitempty"`

	// Resources for agent containers
	Resources ResourceRequirements `json:"resources,omitempty"`

	// SecurityContext for containers
	SecurityContext *SecurityContext `json:"securityContext,omitempty"`

	// NodeSelector for agent placement
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations for agent scheduling
	Tolerations []Toleration `json:"tolerations,omitempty"`

	// Affinity rules for agent placement
	Affinity *Affinity `json:"affinity,omitempty"`
}

// MemorySpec defines distributed memory configuration
type MemorySpec struct {
	// Type of memory backend (redis, hazelcast, etcd)
	Type string `json:"type,omitempty"`

	// Size of memory allocation
	Size string `json:"size,omitempty"`

	// Replication factor for durability
	Replication int32 `json:"replication,omitempty"`

	// Persistence enables durable storage
	Persistence bool `json:"persistence,omitempty"`

	// CachePolicy (LRU, LFU, ARC)
	CachePolicy string `json:"cachePolicy,omitempty"`

	// Compression enables memory compression
	Compression bool `json:"compression,omitempty"`
}

// NeuralSpec defines neural network configuration
type NeuralSpec struct {
	// Enabled activates neural capabilities
	Enabled bool `json:"enabled,omitempty"`

	// Models to deploy
	Models []NeuralModel `json:"models,omitempty"`

	// Acceleration (cpu, gpu, wasm-simd)
	Acceleration string `json:"acceleration,omitempty"`

	// TrainingEnabled allows model updates
	TrainingEnabled bool `json:"trainingEnabled,omitempty"`
}

// NeuralModel defines a neural model deployment
type NeuralModel struct {
	// Name of the model
	Name string `json:"name"`

	// Type (pattern-recognition, optimization, prediction)
	Type string `json:"type"`

	// Path to model artifacts
	Path string `json:"path"`

	// Resources for model serving
	Resources ResourceRequirements `json:"resources,omitempty"`
}

// MonitoringSpec defines monitoring configuration
type MonitoringSpec struct {
	// Enabled activates monitoring
	Enabled bool `json:"enabled,omitempty"`

	// MetricsPort for Prometheus scraping
	MetricsPort int32 `json:"metricsPort,omitempty"`

	// TracingEnabled for distributed tracing
	TracingEnabled bool `json:"tracingEnabled,omitempty"`

	// DashboardEnabled for Grafana dashboards
	DashboardEnabled bool `json:"dashboardEnabled,omitempty"`

	// AlertRules for monitoring alerts
	AlertRules []AlertRule `json:"alertRules,omitempty"`
}

// AlertRule defines a monitoring alert
type AlertRule struct {
	// Name of the alert
	Name string `json:"name"`

	// Expression in PromQL
	Expression string `json:"expression"`

	// Duration before firing
	Duration string `json:"duration"`

	// Severity level
	Severity string `json:"severity"`
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

// NamespaceConfig defines namespace allocation for different components
type NamespaceConfig struct {
	// SwarmNamespace for general swarm agents (default: claude-flow-swarm)
	SwarmNamespace string `json:"swarmNamespace,omitempty"`

	// HiveMindNamespace for hive-mind components (default: claude-flow-hivemind)
	HiveMindNamespace string `json:"hiveMindNamespace,omitempty"`

	// AllowedNamespaces that the operator will watch
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`

	// CreateNamespaces if they don't exist
	CreateNamespaces bool `json:"createNamespaces,omitempty"`
}

// SwarmClusterStatus defines the observed state of SwarmCluster
type SwarmClusterStatus struct {
	// Phase of the swarm cluster
	Phase string `json:"phase,omitempty"`

	// ReadyAgents count
	ReadyAgents int32 `json:"readyAgents,omitempty"`

	// TotalAgents count
	TotalAgents int32 `json:"totalAgents,omitempty"`

	// AgentTypes breakdown
	AgentTypes map[string]int32 `json:"agentTypes,omitempty"`

	// ActiveTasks count
	ActiveTasks int32 `json:"activeTasks,omitempty"`

	// CompletedTasks count
	CompletedTasks int32 `json:"completedTasks,omitempty"`

	// HiveMindStatus
	HiveMindStatus HiveMindStatus `json:"hiveMindStatus,omitempty"`

	// MemoryStatus
	MemoryStatus MemoryStatus `json:"memoryStatus,omitempty"`

	// Conditions for the cluster
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastScaleTime for autoscaling
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// ObservedGeneration for tracking updates
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// HiveMindStatus defines hive-mind operational status
type HiveMindStatus struct {
	// Connected agents to hive-mind
	Connected int32 `json:"connected,omitempty"`

	// SyncStatus of hive-mind
	SyncStatus string `json:"syncStatus,omitempty"`

	// LastSyncTime
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// DatabaseSize current usage
	DatabaseSize string `json:"databaseSize,omitempty"`
}

// MemoryStatus defines memory system status
type MemoryStatus struct {
	// Available memory
	Available string `json:"available,omitempty"`

	// Used memory
	Used string `json:"used,omitempty"`

	// HitRate for cache
	HitRate float64 `json:"hitRate,omitempty"`

	// Evictions count
	Evictions int64 `json:"evictions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=sc
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.autoscaling.minAgents,statuspath=.status.totalAgents
// +kubebuilder:printcolumn:name="Topology",type=string,JSONPath=`.spec.topology`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyAgents`
// +kubebuilder:printcolumn:name="Total",type=integer,JSONPath=`.status.totalAgents`
// +kubebuilder:printcolumn:name="Tasks",type=integer,JSONPath=`.status.activeTasks`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

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

// ResourceRequirements simplified resource definition
type ResourceRequirements struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	GPU    string `json:"gpu,omitempty"`
}

// SecurityContext simplified security context
type SecurityContext struct {
	RunAsNonRoot bool  `json:"runAsNonRoot,omitempty"`
	RunAsUser    int64 `json:"runAsUser,omitempty"`
	FSGroup      int64 `json:"fsGroup,omitempty"`
}

// Toleration simplified toleration
type Toleration struct {
	Key      string `json:"key,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect,omitempty"`
}

// Affinity simplified affinity rules
type Affinity struct {
	NodeAffinity *NodeAffinity `json:"nodeAffinity,omitempty"`
	PodAffinity  *PodAffinity  `json:"podAffinity,omitempty"`
}

// NodeAffinity simplified node affinity
type NodeAffinity struct {
	RequiredDuringScheduling []NodeSelectorTerm `json:"requiredDuringScheduling,omitempty"`
}

// NodeSelectorTerm simplified selector
type NodeSelectorTerm struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// PodAffinity simplified pod affinity
type PodAffinity struct {
	PreferredDuringScheduling []WeightedPodAffinityTerm `json:"preferredDuringScheduling,omitempty"`
}

// WeightedPodAffinityTerm simplified weighted term
type WeightedPodAffinityTerm struct {
	Weight          int32             `json:"weight"`
	PodAffinityTerm PodAffinityTerm   `json:"podAffinityTerm"`
}

// PodAffinityTerm simplified term
type PodAffinityTerm struct {
	LabelSelector map[string]string `json:"labelSelector,omitempty"`
	TopologyKey   string            `json:"topologyKey"`
}

func init() {
	SchemeBuilder.Register(&SwarmCluster{}, &SwarmClusterList{})
}