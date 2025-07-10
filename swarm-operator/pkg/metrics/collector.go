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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// SwarmCluster metrics
	swarmClusterTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_cluster_total",
			Help: "Total number of SwarmCluster resources",
		},
		[]string{"namespace"},
	)

	swarmClusterPhase = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_cluster_phase",
			Help: "Current phase of SwarmCluster (1 for the current phase, 0 for others)",
		},
		[]string{"namespace", "name", "phase"},
	)

	swarmClusterAgents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_cluster_agents",
			Help: "Number of agents in the swarm cluster",
		},
		[]string{"namespace", "name", "status"},
	)

	// Agent metrics
	agentTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_agent_total",
			Help: "Total number of Agent resources",
		},
		[]string{"namespace", "swarm_cluster", "type"},
	)

	agentPhase = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_agent_phase",
			Help: "Current phase of Agent (1 for the current phase, 0 for others)",
		},
		[]string{"namespace", "name", "phase", "type"},
	)

	agentTasks = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_agent_tasks_current",
			Help: "Current number of tasks being processed by the agent",
		},
		[]string{"namespace", "name", "type"},
	)

	agentTasksCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "swarm_agent_tasks_completed_total",
			Help: "Total number of tasks completed by the agent",
		},
		[]string{"namespace", "name", "type", "status"},
	)

	agentCPUUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_agent_cpu_usage_percent",
			Help: "CPU usage percentage of the agent",
		},
		[]string{"namespace", "name", "type"},
	)

	agentMemoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_agent_memory_usage_bytes",
			Help: "Memory usage in bytes of the agent",
		},
		[]string{"namespace", "name", "type"},
	)

	// Task metrics
	taskQueueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_task_queue_size",
			Help: "Current size of the task queue",
		},
		[]string{"namespace", "swarm_cluster"},
	)

	taskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "swarm_task_duration_seconds",
			Help:    "Duration of task execution in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~100s
		},
		[]string{"namespace", "swarm_cluster", "agent_type", "task_type"},
	)

	taskSuccessRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_task_success_rate",
			Help: "Success rate of tasks (0-1)",
		},
		[]string{"namespace", "swarm_cluster"},
	)

	// Topology metrics
	topologyPeerConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_topology_peer_connections",
			Help: "Number of peer connections per agent",
		},
		[]string{"namespace", "name", "topology"},
	)

	topologyCommunicationLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "swarm_topology_communication_latency_ms",
			Help:    "Communication latency between peers in milliseconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1ms to ~1s
		},
		[]string{"namespace", "from_agent", "to_agent"},
	)

	// Autoscaling metrics
	autoscalingEvents = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "swarm_autoscaling_events_total",
			Help: "Total number of autoscaling events",
		},
		[]string{"namespace", "swarm_cluster", "direction"},
	)

	autoscalingTargetAgents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "swarm_autoscaling_target_agents",
			Help: "Target number of agents based on autoscaling calculations",
		},
		[]string{"namespace", "swarm_cluster"},
	)

	// Controller metrics
	reconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "swarm_controller_reconcile_total",
			Help: "Total number of reconciliations",
		},
		[]string{"controller", "result"},
	)

	reconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "swarm_controller_reconcile_duration_seconds",
			Help:    "Duration of reconciliation in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s
		},
		[]string{"controller"},
	)
)

func init() {
	// Register metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		// SwarmCluster metrics
		swarmClusterTotal,
		swarmClusterPhase,
		swarmClusterAgents,
		
		// Agent metrics
		agentTotal,
		agentPhase,
		agentTasks,
		agentTasksCompleted,
		agentCPUUsage,
		agentMemoryUsage,
		
		// Task metrics
		taskQueueSize,
		taskDuration,
		taskSuccessRate,
		
		// Topology metrics
		topologyPeerConnections,
		topologyCommunicationLatency,
		
		// Autoscaling metrics
		autoscalingEvents,
		autoscalingTargetAgents,
		
		// Controller metrics
		reconcileTotal,
		reconcileDuration,
	)
}

// MetricsRecorder provides methods to record metrics
type MetricsRecorder struct{}

// NewMetricsRecorder creates a new metrics recorder
func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{}
}

// RecordSwarmClusterPhase records the current phase of a SwarmCluster
func (m *MetricsRecorder) RecordSwarmClusterPhase(namespace, name, phase string) {
	phases := []string{"Pending", "Initializing", "Running", "Scaling", "Terminating", "Failed"}
	for _, p := range phases {
		value := 0.0
		if p == phase {
			value = 1.0
		}
		swarmClusterPhase.WithLabelValues(namespace, name, p).Set(value)
	}
}

// RecordSwarmClusterAgents records agent counts
func (m *MetricsRecorder) RecordSwarmClusterAgents(namespace, name string, active, ready int32) {
	swarmClusterAgents.WithLabelValues(namespace, name, "active").Set(float64(active))
	swarmClusterAgents.WithLabelValues(namespace, name, "ready").Set(float64(ready))
}

// RecordAgentPhase records the current phase of an Agent
func (m *MetricsRecorder) RecordAgentPhase(namespace, name, agentType, phase string) {
	phases := []string{"Pending", "Initializing", "Ready", "Busy", "Terminating", "Failed"}
	for _, p := range phases {
		value := 0.0
		if p == phase {
			value = 1.0
		}
		agentPhase.WithLabelValues(namespace, name, p, agentType).Set(value)
	}
}

// RecordAgentTasks records current task count for an agent
func (m *MetricsRecorder) RecordAgentTasks(namespace, name, agentType string, count int) {
	agentTasks.WithLabelValues(namespace, name, agentType).Set(float64(count))
}

// RecordAgentTaskCompleted records a completed task
func (m *MetricsRecorder) RecordAgentTaskCompleted(namespace, name, agentType, status string) {
	agentTasksCompleted.WithLabelValues(namespace, name, agentType, status).Inc()
}

// RecordAgentResourceUsage records agent resource usage
func (m *MetricsRecorder) RecordAgentResourceUsage(namespace, name, agentType string, cpu float64, memory int64) {
	agentCPUUsage.WithLabelValues(namespace, name, agentType).Set(cpu)
	agentMemoryUsage.WithLabelValues(namespace, name, agentType).Set(float64(memory))
}

// RecordTaskQueueSize records the task queue size
func (m *MetricsRecorder) RecordTaskQueueSize(namespace, swarmCluster string, size int32) {
	taskQueueSize.WithLabelValues(namespace, swarmCluster).Set(float64(size))
}

// RecordTaskDuration records task execution duration
func (m *MetricsRecorder) RecordTaskDuration(namespace, swarmCluster, agentType, taskType string, duration float64) {
	taskDuration.WithLabelValues(namespace, swarmCluster, agentType, taskType).Observe(duration)
}

// RecordTaskSuccessRate records the task success rate
func (m *MetricsRecorder) RecordTaskSuccessRate(namespace, swarmCluster string, rate float64) {
	taskSuccessRate.WithLabelValues(namespace, swarmCluster).Set(rate)
}

// RecordPeerConnections records the number of peer connections
func (m *MetricsRecorder) RecordPeerConnections(namespace, name, topology string, connections int) {
	topologyPeerConnections.WithLabelValues(namespace, name, topology).Set(float64(connections))
}

// RecordCommunicationLatency records latency between peers
func (m *MetricsRecorder) RecordCommunicationLatency(namespace, fromAgent, toAgent string, latencyMs float64) {
	topologyCommunicationLatency.WithLabelValues(namespace, fromAgent, toAgent).Observe(latencyMs)
}

// RecordAutoscalingEvent records an autoscaling event
func (m *MetricsRecorder) RecordAutoscalingEvent(namespace, swarmCluster, direction string) {
	autoscalingEvents.WithLabelValues(namespace, swarmCluster, direction).Inc()
}

// RecordAutoscalingTarget records the target agent count
func (m *MetricsRecorder) RecordAutoscalingTarget(namespace, swarmCluster string, target int) {
	autoscalingTargetAgents.WithLabelValues(namespace, swarmCluster).Set(float64(target))
}

// RecordReconciliation records reconciliation metrics
func (m *MetricsRecorder) RecordReconciliation(controller string, duration float64, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	reconcileTotal.WithLabelValues(controller, result).Inc()
	reconcileDuration.WithLabelValues(controller).Observe(duration)
}