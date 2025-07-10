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

package topology

import (
	"fmt"
	"sort"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

// Manager handles topology configuration for swarm agents
type Manager struct {
	topology string
}

// NewManager creates a new topology manager
func NewManager(topology string) *Manager {
	return &Manager{
		topology: topology,
	}
}

// CalculatePeers determines peer connections based on topology
func (m *Manager) CalculatePeers(agents []swarmv1alpha1.Agent) map[string][]string {
	switch m.topology {
	case string(swarmv1alpha1.MeshTopology):
		return m.calculateMeshPeers(agents)
	case string(swarmv1alpha1.HierarchicalTopology):
		return m.calculateHierarchicalPeers(agents)
	case string(swarmv1alpha1.RingTopology):
		return m.calculateRingPeers(agents)
	case string(swarmv1alpha1.StarTopology):
		return m.calculateStarPeers(agents)
	default:
		// Default to mesh if unknown topology
		return m.calculateMeshPeers(agents)
	}
}

// calculateMeshPeers creates full mesh connectivity
func (m *Manager) calculateMeshPeers(agents []swarmv1alpha1.Agent) map[string][]string {
	peerMap := make(map[string][]string)
	
	// Sort agents by name for consistent peer ordering
	sortedAgents := make([]swarmv1alpha1.Agent, len(agents))
	copy(sortedAgents, agents)
	sort.Slice(sortedAgents, func(i, j int) bool {
		return sortedAgents[i].Name < sortedAgents[j].Name
	})
	
	// In mesh topology, every agent connects to every other agent
	for i, agent := range sortedAgents {
		peers := []string{}
		for j, peer := range sortedAgents {
			if i != j {
				peers = append(peers, m.formatPeerAddress(peer))
			}
		}
		peerMap[agent.Name] = peers
	}
	
	return peerMap
}

// calculateHierarchicalPeers creates a tree structure
func (m *Manager) calculateHierarchicalPeers(agents []swarmv1alpha1.Agent) map[string][]string {
	peerMap := make(map[string][]string)
	
	if len(agents) == 0 {
		return peerMap
	}
	
	// Sort agents to ensure consistent hierarchy
	sortedAgents := make([]swarmv1alpha1.Agent, len(agents))
	copy(sortedAgents, agents)
	sort.Slice(sortedAgents, func(i, j int) bool {
		// Coordinators first, then by name
		if sortedAgents[i].Spec.Type == swarmv1alpha1.CoordinatorAgent && 
		   sortedAgents[j].Spec.Type != swarmv1alpha1.CoordinatorAgent {
			return true
		}
		if sortedAgents[i].Spec.Type != swarmv1alpha1.CoordinatorAgent && 
		   sortedAgents[j].Spec.Type == swarmv1alpha1.CoordinatorAgent {
			return false
		}
		return sortedAgents[i].Name < sortedAgents[j].Name
	})
	
	// First agent is root
	root := sortedAgents[0]
	peerMap[root.Name] = []string{}
	
	// Binary tree structure: each agent connects to parent and children
	for i := 1; i < len(sortedAgents); i++ {
		agent := sortedAgents[i]
		peers := []string{}
		
		// Parent connection
		parentIdx := (i - 1) / 2
		peers = append(peers, m.formatPeerAddress(sortedAgents[parentIdx]))
		
		// Children connections
		leftChildIdx := 2*i + 1
		rightChildIdx := 2*i + 2
		
		if leftChildIdx < len(sortedAgents) {
			peers = append(peers, m.formatPeerAddress(sortedAgents[leftChildIdx]))
		}
		if rightChildIdx < len(sortedAgents) {
			peers = append(peers, m.formatPeerAddress(sortedAgents[rightChildIdx]))
		}
		
		peerMap[agent.Name] = peers
		
		// Update parent's peer list
		peerMap[sortedAgents[parentIdx].Name] = append(
			peerMap[sortedAgents[parentIdx].Name], 
			m.formatPeerAddress(agent),
		)
	}
	
	return peerMap
}

// calculateRingPeers creates a circular connection pattern
func (m *Manager) calculateRingPeers(agents []swarmv1alpha1.Agent) map[string][]string {
	peerMap := make(map[string][]string)
	
	if len(agents) == 0 {
		return peerMap
	}
	
	// Sort agents for consistent ring order
	sortedAgents := make([]swarmv1alpha1.Agent, len(agents))
	copy(sortedAgents, agents)
	sort.Slice(sortedAgents, func(i, j int) bool {
		return sortedAgents[i].Name < sortedAgents[j].Name
	})
	
	// Each agent connects to previous and next in the ring
	for i, agent := range sortedAgents {
		peers := []string{}
		
		// Previous peer
		prevIdx := (i - 1 + len(sortedAgents)) % len(sortedAgents)
		peers = append(peers, m.formatPeerAddress(sortedAgents[prevIdx]))
		
		// Next peer
		nextIdx := (i + 1) % len(sortedAgents)
		if nextIdx != prevIdx { // Avoid duplicate when only 2 agents
			peers = append(peers, m.formatPeerAddress(sortedAgents[nextIdx]))
		}
		
		peerMap[agent.Name] = peers
	}
	
	return peerMap
}

// calculateStarPeers creates a hub-and-spoke pattern
func (m *Manager) calculateStarPeers(agents []swarmv1alpha1.Agent) map[string][]string {
	peerMap := make(map[string][]string)
	
	if len(agents) == 0 {
		return peerMap
	}
	
	// Find coordinator or use first agent as hub
	var hub *swarmv1alpha1.Agent
	var spokes []swarmv1alpha1.Agent
	
	for i := range agents {
		if agents[i].Spec.Type == swarmv1alpha1.CoordinatorAgent {
			hub = &agents[i]
		} else {
			spokes = append(spokes, agents[i])
		}
	}
	
	// If no coordinator, use first agent as hub
	if hub == nil {
		hub = &agents[0]
		spokes = agents[1:]
	}
	
	// Hub connects to all spokes
	hubPeers := []string{}
	for _, spoke := range spokes {
		hubPeers = append(hubPeers, m.formatPeerAddress(spoke))
		// Each spoke only connects to hub
		peerMap[spoke.Name] = []string{m.formatPeerAddress(*hub)}
	}
	peerMap[hub.Name] = hubPeers
	
	return peerMap
}

// formatPeerAddress creates the peer connection string
func (m *Manager) formatPeerAddress(agent swarmv1alpha1.Agent) string {
	// Format: agent-name.namespace.svc.cluster.local:port
	return fmt.Sprintf("%s.%s.svc.cluster.local:%d", 
		agent.Name, 
		agent.Namespace, 
		agent.Spec.CommunicationEndpoints.Port)
}

// ValidateTopology checks if agents can form the requested topology
func (m *Manager) ValidateTopology(agentCount int) error {
	switch m.topology {
	case string(swarmv1alpha1.MeshTopology):
		// Mesh works with any number of agents
		return nil
	case string(swarmv1alpha1.HierarchicalTopology):
		// Hierarchical needs at least 2 agents
		if agentCount < 2 {
			return fmt.Errorf("hierarchical topology requires at least 2 agents, got %d", agentCount)
		}
	case string(swarmv1alpha1.RingTopology):
		// Ring needs at least 3 agents for proper circulation
		if agentCount < 3 {
			return fmt.Errorf("ring topology requires at least 3 agents, got %d", agentCount)
		}
	case string(swarmv1alpha1.StarTopology):
		// Star needs at least 2 agents (hub + spoke)
		if agentCount < 2 {
			return fmt.Errorf("star topology requires at least 2 agents, got %d", agentCount)
		}
	}
	return nil
}

// GetOptimalAgentCount returns the recommended agent count for the topology
func (m *Manager) GetOptimalAgentCount() int {
	switch m.topology {
	case string(swarmv1alpha1.MeshTopology):
		return 5 // Good balance of connectivity and overhead
	case string(swarmv1alpha1.HierarchicalTopology):
		return 7 // Perfect binary tree with 3 levels
	case string(swarmv1alpha1.RingTopology):
		return 6 // Even number for balanced communication
	case string(swarmv1alpha1.StarTopology):
		return 5 // 1 hub + 4 spokes
	default:
		return 3
	}
}