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

package utils

import (
	"fmt"
	"sort"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

// TaskDistributor handles task assignment to agents
type TaskDistributor struct {
	algorithm        string
	maxTasksPerAgent int32
}

// NewTaskDistributor creates a new task distributor
func NewTaskDistributor(spec swarmv1alpha1.TaskDistributionSpec) *TaskDistributor {
	return &TaskDistributor{
		algorithm:        spec.Algorithm,
		maxTasksPerAgent: spec.MaxTasksPerAgent,
	}
}

// Task represents a task to be distributed
type Task struct {
	Name         string
	Type         string
	Priority     int
	Capabilities []string
}

// AssignTask assigns a task to the most suitable agent
func (td *TaskDistributor) AssignTask(task Task, agents []swarmv1alpha1.Agent) (*swarmv1alpha1.Agent, error) {
	// Filter out agents that are at capacity or not ready
	availableAgents := td.filterAvailableAgents(agents)
	
	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no available agents")
	}

	switch td.algorithm {
	case "round-robin":
		return td.roundRobinAssignment(availableAgents)
	case "least-loaded":
		return td.leastLoadedAssignment(availableAgents)
	case "capability-based":
		return td.capabilityBasedAssignment(task, availableAgents)
	case "priority-based":
		return td.priorityBasedAssignment(task, availableAgents)
	default:
		// Default to capability-based
		return td.capabilityBasedAssignment(task, availableAgents)
	}
}

// filterAvailableAgents returns agents that can accept new tasks
func (td *TaskDistributor) filterAvailableAgents(agents []swarmv1alpha1.Agent) []*swarmv1alpha1.Agent {
	available := []*swarmv1alpha1.Agent{}
	
	for i := range agents {
		agent := &agents[i]
		// Check if agent is ready and not at capacity
		if agent.Status.Phase == "Ready" || agent.Status.Phase == "Busy" {
			if int32(len(agent.Status.CurrentTasks)) < td.maxTasksPerAgent {
				available = append(available, agent)
			}
		}
	}
	
	return available
}

// roundRobinAssignment selects agents in round-robin fashion
func (td *TaskDistributor) roundRobinAssignment(agents []*swarmv1alpha1.Agent) (*swarmv1alpha1.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	// Sort by completed tasks to ensure even distribution
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Status.CompletedTasks < agents[j].Status.CompletedTasks
	})
	
	return agents[0], nil
}

// leastLoadedAssignment selects the agent with fewest current tasks
func (td *TaskDistributor) leastLoadedAssignment(agents []*swarmv1alpha1.Agent) (*swarmv1alpha1.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	// Sort by current task count
	sort.Slice(agents, func(i, j int) bool {
		return len(agents[i].Status.CurrentTasks) < len(agents[j].Status.CurrentTasks)
	})
	
	return agents[0], nil
}

// capabilityBasedAssignment selects agent based on capability match
func (td *TaskDistributor) capabilityBasedAssignment(task Task, agents []*swarmv1alpha1.Agent) (*swarmv1alpha1.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	// Score agents based on capability match
	type scoredAgent struct {
		agent *swarmv1alpha1.Agent
		score int
	}
	
	scored := []scoredAgent{}
	for _, agent := range agents {
		score := td.calculateCapabilityScore(task.Capabilities, agent.Spec.Capabilities)
		
		// Bonus for agent type matching task type
		if td.isAgentTypeMatch(agent.Spec.Type, task.Type) {
			score += 10
		}
		
		scored = append(scored, scoredAgent{agent: agent, score: score})
	}
	
	// Sort by score (highest first)
	sort.Slice(scored, func(i, j int) bool {
		// If scores are equal, prefer less loaded agent
		if scored[i].score == scored[j].score {
			return len(scored[i].agent.Status.CurrentTasks) < len(scored[j].agent.Status.CurrentTasks)
		}
		return scored[i].score > scored[j].score
	})
	
	if len(scored) > 0 && scored[0].score > 0 {
		return scored[0].agent, nil
	}
	
	// Fallback to least loaded if no capability match
	return td.leastLoadedAssignment(agents)
}

// priorityBasedAssignment considers task priority and agent capabilities
func (td *TaskDistributor) priorityBasedAssignment(task Task, agents []*swarmv1alpha1.Agent) (*swarmv1alpha1.Agent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	// For high priority tasks, find the best agent even if slightly loaded
	if task.Priority > 7 {
		// Find agents with matching capabilities
		capableAgents := []*swarmv1alpha1.Agent{}
		for _, agent := range agents {
			if td.calculateCapabilityScore(task.Capabilities, agent.Spec.Capabilities) > 0 {
				capableAgents = append(capableAgents, agent)
			}
		}
		
		if len(capableAgents) > 0 {
			return td.leastLoadedAssignment(capableAgents)
		}
	}
	
	// For normal priority, use capability-based assignment
	return td.capabilityBasedAssignment(task, agents)
}

// calculateCapabilityScore calculates how well agent capabilities match task requirements
func (td *TaskDistributor) calculateCapabilityScore(required, available []string) int {
	score := 0
	capMap := make(map[string]bool)
	
	for _, cap := range available {
		capMap[cap] = true
	}
	
	for _, req := range required {
		if capMap[req] {
			score++
		}
	}
	
	return score
}

// isAgentTypeMatch checks if agent type matches task type
func (td *TaskDistributor) isAgentTypeMatch(agentType swarmv1alpha1.AgentType, taskType string) bool {
	matches := map[swarmv1alpha1.AgentType][]string{
		swarmv1alpha1.ResearcherAgent:  {"research", "analysis", "investigation"},
		swarmv1alpha1.CoderAgent:       {"coding", "development", "implementation"},
		swarmv1alpha1.AnalystAgent:     {"analysis", "metrics", "reporting"},
		swarmv1alpha1.TesterAgent:      {"testing", "validation", "qa"},
		swarmv1alpha1.ArchitectAgent:   {"design", "architecture", "planning"},
		swarmv1alpha1.OptimizerAgent:   {"optimization", "performance", "tuning"},
		swarmv1alpha1.DocumenterAgent:  {"documentation", "writing", "guides"},
		swarmv1alpha1.ReviewerAgent:    {"review", "audit", "verification"},
		swarmv1alpha1.CoordinatorAgent: {"coordination", "management", "orchestration"},
	}
	
	if taskTypes, ok := matches[agentType]; ok {
		for _, t := range taskTypes {
			if t == taskType {
				return true
			}
		}
	}
	
	return false
}

// RebalanceTasks redistributes tasks among agents for better load distribution
func (td *TaskDistributor) RebalanceTasks(agents []swarmv1alpha1.Agent) []TaskMigration {
	migrations := []TaskMigration{}
	
	// Calculate average load
	totalTasks := 0
	for _, agent := range agents {
		totalTasks += len(agent.Status.CurrentTasks)
	}
	
	if len(agents) == 0 {
		return migrations
	}
	
	avgLoad := float64(totalTasks) / float64(len(agents))
	threshold := avgLoad * 0.2 // 20% threshold
	
	// Find overloaded and underloaded agents
	overloaded := []*swarmv1alpha1.Agent{}
	underloaded := []*swarmv1alpha1.Agent{}
	
	for i := range agents {
		agent := &agents[i]
		load := float64(len(agent.Status.CurrentTasks))
		
		if load > avgLoad+threshold {
			overloaded = append(overloaded, agent)
		} else if load < avgLoad-threshold {
			underloaded = append(underloaded, agent)
		}
	}
	
	// Create migrations from overloaded to underloaded
	for _, source := range overloaded {
		if len(underloaded) == 0 {
			break
		}
		
		excessTasks := int(float64(len(source.Status.CurrentTasks)) - avgLoad)
		for i := 0; i < excessTasks && len(underloaded) > 0; i++ {
			// Find best target
			target := underloaded[0]
			
			// Create migration
			if len(source.Status.CurrentTasks) > 0 {
				task := source.Status.CurrentTasks[len(source.Status.CurrentTasks)-1]
				migrations = append(migrations, TaskMigration{
					Task:       task,
					FromAgent:  source.Name,
					ToAgent:    target.Name,
					Reason:     "Load balancing",
				})
			}
			
			// Update target load
			if float64(len(target.Status.CurrentTasks)+1) >= avgLoad {
				underloaded = underloaded[1:]
			}
		}
	}
	
	return migrations
}

// TaskMigration represents a task migration between agents
type TaskMigration struct {
	Task      swarmv1alpha1.TaskReference
	FromAgent string
	ToAgent   string
	Reason    string
}