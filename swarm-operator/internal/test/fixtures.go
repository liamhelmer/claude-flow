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

package test

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/pointer"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

// SwarmClusterFixture creates a test SwarmCluster
func SwarmClusterFixture(name, namespace string) *swarmv1alpha1.SwarmCluster {
	return &swarmv1alpha1.SwarmCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: swarmv1alpha1.SwarmClusterSpec{
			Topology: swarmv1alpha1.MeshTopology,
			Size:     3,
			Strategy: swarmv1alpha1.StrategySpec{
				Type:               swarmv1alpha1.BalancedStrategy,
				MaxConcurrentTasks: 5,
			},
		},
	}
}

// SwarmClusterWithTopology creates a test SwarmCluster with specific topology
func SwarmClusterWithTopology(name, namespace string, topology swarmv1alpha1.TopologyType, size int32) *swarmv1alpha1.SwarmCluster {
	cluster := SwarmClusterFixture(name, namespace)
	cluster.Spec.Topology = topology
	cluster.Spec.Size = size
	return cluster
}

// ReadySwarmCluster creates a SwarmCluster in Ready state
func ReadySwarmCluster(name, namespace string) *swarmv1alpha1.SwarmCluster {
	cluster := SwarmClusterFixture(name, namespace)
	cluster.Status = swarmv1alpha1.SwarmClusterStatus{
		State:        swarmv1alpha1.ClusterReady,
		ReadyAgents:  cluster.Spec.Size,
		ActiveAgents: cluster.Spec.Size - 1,
		Health:       swarmv1alpha1.HealthyCondition,
		Topology:     cluster.Spec.Topology,
		LastUpdated:  metav1.NewTime(time.Now()),
	}
	return cluster
}

// AgentFixture creates a test Agent
func AgentFixture(name, namespace, clusterName string, agentType swarmv1alpha1.AgentType) *swarmv1alpha1.Agent {
	return &swarmv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"swarm.claudeflow.io/cluster": clusterName,
				"swarm.claudeflow.io/type":    string(agentType),
			},
		},
		Spec: swarmv1alpha1.AgentSpec{
			Type:         agentType,
			ClusterRef:   clusterName,
			Capabilities: DefaultCapabilities(agentType),
			Resources: swarmv1alpha1.ResourceRequirements{
				Requests: swarmv1alpha1.ResourceList{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Limits: swarmv1alpha1.ResourceList{
					CPU:    "1",
					Memory: "1Gi",
				},
			},
		},
	}
}

// ReadyAgent creates an Agent in Ready state
func ReadyAgent(name, namespace, clusterName string, agentType swarmv1alpha1.AgentType) *swarmv1alpha1.Agent {
	agent := AgentFixture(name, namespace, clusterName, agentType)
	agent.Status = swarmv1alpha1.AgentStatus{
		State:       swarmv1alpha1.AgentReady,
		Health:      swarmv1alpha1.HealthyCondition,
		Workload:    10,
		Capacity:    100,
		TaskCount:   1,
		Connections: []string{},
		LastUpdated: metav1.NewTime(time.Now()),
	}
	return agent
}

// BusyAgent creates an Agent in Busy state
func BusyAgent(name, namespace, clusterName string, agentType swarmv1alpha1.AgentType) *swarmv1alpha1.Agent {
	agent := ReadyAgent(name, namespace, clusterName, agentType)
	agent.Status.State = swarmv1alpha1.AgentBusy
	agent.Status.Workload = 85
	agent.Status.TaskCount = 5
	return agent
}

// SwarmTaskFixture creates a test SwarmTask
func SwarmTaskFixture(name, namespace, clusterRef string) *swarmv1alpha1.SwarmTask {
	return &swarmv1alpha1.SwarmTask{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: swarmv1alpha1.SwarmTaskSpec{
			ClusterRef: clusterRef,
			Task: swarmv1alpha1.TaskSpec{
				Type:        "research",
				Description: "Test research task",
				Priority:    swarmv1alpha1.MediumPriority,
			},
			Strategy: swarmv1alpha1.StrategySpec{
				Type:               swarmv1alpha1.ParallelStrategy,
				MaxConcurrentTasks: 3,
			},
		},
	}
}

// RunningSwarmTask creates a SwarmTask in Running state
func RunningSwarmTask(name, namespace, clusterRef string, assignedAgents []string) *swarmv1alpha1.SwarmTask {
	task := SwarmTaskFixture(name, namespace, clusterRef)
	task.Status = swarmv1alpha1.SwarmTaskStatus{
		State:          swarmv1alpha1.TaskRunning,
		AssignedAgents: assignedAgents,
		StartTime:      &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
	}
	return task
}

// CompletedSwarmTask creates a SwarmTask in Completed state
func CompletedSwarmTask(name, namespace, clusterRef string) *swarmv1alpha1.SwarmTask {
	task := SwarmTaskFixture(name, namespace, clusterRef)
	startTime := time.Now().Add(-10 * time.Minute)
	completionTime := time.Now().Add(-1 * time.Minute)
	task.Status = swarmv1alpha1.SwarmTaskStatus{
		State:          swarmv1alpha1.TaskCompleted,
		AssignedAgents: []string{"agent-1", "agent-2"},
		StartTime:      &metav1.Time{Time: startTime},
		CompletionTime: &metav1.Time{Time: completionTime},
		Result: &swarmv1alpha1.TaskResult{
			Success: true,
			Message: "Task completed successfully",
			Data:    map[string]string{"result": "success"},
		},
	}
	return task
}

// DefaultCapabilities returns default capabilities for an agent type
func DefaultCapabilities(agentType swarmv1alpha1.AgentType) []string {
	switch agentType {
	case swarmv1alpha1.ResearcherAgent:
		return []string{"research", "analysis", "documentation"}
	case swarmv1alpha1.CoderAgent:
		return []string{"coding", "testing", "debugging"}
	case swarmv1alpha1.AnalystAgent:
		return []string{"analysis", "optimization", "metrics"}
	case swarmv1alpha1.ArchitectAgent:
		return []string{"design", "planning", "architecture"}
	case swarmv1alpha1.TesterAgent:
		return []string{"testing", "validation", "qa"}
	case swarmv1alpha1.ReviewerAgent:
		return []string{"review", "feedback", "quality"}
	case swarmv1alpha1.OptimizerAgent:
		return []string{"optimization", "performance", "tuning"}
	case swarmv1alpha1.DocumenterAgent:
		return []string{"documentation", "writing", "guides"}
	case swarmv1alpha1.MonitorAgent:
		return []string{"monitoring", "alerting", "observability"}
	case swarmv1alpha1.CoordinatorAgent:
		return []string{"coordination", "planning", "management"}
	default:
		return []string{"general"}
	}
}

// CreateAgentList creates a list of agents for a cluster
func CreateAgentList(clusterName, namespace string, count int, agentTypes ...swarmv1alpha1.AgentType) []swarmv1alpha1.Agent {
	agents := make([]swarmv1alpha1.Agent, count)
	
	// Default to a mix of agent types if none specified
	if len(agentTypes) == 0 {
		agentTypes = []swarmv1alpha1.AgentType{
			swarmv1alpha1.ResearcherAgent,
			swarmv1alpha1.CoderAgent,
			swarmv1alpha1.AnalystAgent,
		}
	}

	for i := 0; i < count; i++ {
		agentType := agentTypes[i%len(agentTypes)]
		name := fmt.Sprintf("%s-agent-%d", clusterName, i)
		agents[i] = *AgentFixture(name, namespace, clusterName, agentType)
	}

	return agents
}

// CreateOwnerReference creates an owner reference for the given object
func CreateOwnerReference(owner metav1.Object, gvk schema.GroupVersionKind) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		Controller:         pointer.Bool(true),
		BlockOwnerDeletion: pointer.Bool(true),
	}
}