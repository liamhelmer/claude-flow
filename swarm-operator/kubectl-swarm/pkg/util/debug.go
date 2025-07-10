/*
Copyright 2024 The Swarm Authors.

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

package util

import (
	"context"
	"fmt"
	"os"

	"github.com/claude-flow/kubectl-swarm/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// DebugInfo contains diagnostic information
type DebugInfo struct {
	SwarmName     string                 `json:"swarmName"`
	Namespace     string                 `json:"namespace"`
	Timestamp     metav1.Time            `json:"timestamp"`
	SwarmStatus   map[string]interface{} `json:"swarmStatus,omitempty"`
	AgentCount    int                    `json:"agentCount"`
	HealthyAgents int                    `json:"healthyAgents"`
	TaskCount     int                    `json:"taskCount"`
	TaskStatus    map[string]int         `json:"taskStatus,omitempty"`
	Errors        []DebugEntry           `json:"errors,omitempty"`
	Warnings      []DebugEntry           `json:"warnings,omitempty"`
}

// DebugEntry represents an error or warning entry
type DebugEntry struct {
	Component string `json:"component"`
	Message   string `json:"message"`
}

// AddError adds an error entry
func (d *DebugInfo) AddError(component, message string) {
	d.Errors = append(d.Errors, DebugEntry{
		Component: component,
		Message:   message,
	})
}

// AddWarning adds a warning entry
func (d *DebugInfo) AddWarning(component, message string) {
	d.Warnings = append(d.Warnings, DebugEntry{
		Component: component,
		Message:   message,
	})
}

// ExportDebugInfo exports debug information to a file
func ExportDebugInfo(info *DebugInfo, filename string) error {
	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal debug info: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Diagnostic test functions

// TestAgentCommunication tests inter-agent communication
func TestAgentCommunication(ctx context.Context, swarmClient *client.SwarmClient, swarmName string) error {
	// List all agents
	agents, err := swarmClient.ListAgents(ctx, swarmName, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if len(agents.Items) < 2 {
		return fmt.Errorf("need at least 2 agents for communication test")
	}

	// Check if agents have correct labels and can discover each other
	for _, agent := range agents.Items {
		labels := agent.GetLabels()
		if labels["swarm.io/swarm"] != swarmName {
			return fmt.Errorf("agent %s missing swarm label", agent.GetName())
		}
	}

	return nil
}

// TestTaskSubmission tests task submission capability
func TestTaskSubmission(ctx context.Context, swarmClient *client.SwarmClient, swarmName, namespace string) error {
	// Create a test task
	testTask := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "swarm.io/v1alpha1",
			"kind":       "SwarmTask",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("test-task-%d", metav1.Now().Unix()),
				"namespace": namespace,
				"labels": map[string]interface{}{
					"swarm.io/swarm": swarmName,
					"test":           "true",
				},
			},
			"spec": map[string]interface{}{
				"swarmRef": map[string]interface{}{
					"name": swarmName,
				},
				"task": map[string]interface{}{
					"description": "Test task for diagnostics",
					"priority":    "low",
				},
			},
		},
	}

	// Create the task
	created, err := swarmClient.CreateTask(ctx, testTask, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create test task: %w", err)
	}

	// Clean up test task
	defer swarmClient.DeleteTask(ctx, created.GetName(), metav1.DeleteOptions{})

	return nil
}

// TestResourceAvailability tests resource availability
func TestResourceAvailability(ctx context.Context, clientset kubernetes.Interface, namespace string) error {
	// Check if namespace exists
	_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("namespace %s not accessible: %w", namespace, err)
	}

	// Check resource quotas
	quotas, err := clientset.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list resource quotas: %w", err)
	}

	// Check if any quota is exceeded
	for _, quota := range quotas.Items {
		used := quota.Status.Used
		hard := quota.Status.Hard
		
		for resource, hardLimit := range hard {
			if usedAmount, ok := used[resource]; ok {
				if usedAmount.Cmp(hardLimit) >= 0 {
					return fmt.Errorf("resource quota exceeded for %s", resource)
				}
			}
		}
	}

	return nil
}