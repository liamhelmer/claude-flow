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

package client

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
)

// SwarmClient provides access to swarm resources
type SwarmClient struct {
	dynamicClient dynamic.Interface
	namespace     string
}

var (
	swarmGVR = schema.GroupVersionResource{
		Group:    "swarm.io",
		Version:  "v1alpha1",
		Resource: "swarms",
	}

	swarmAgentGVR = schema.GroupVersionResource{
		Group:    "swarm.io",
		Version:  "v1alpha1",
		Resource: "swarmagents",
	}

	swarmTaskGVR = schema.GroupVersionResource{
		Group:    "swarm.io",
		Version:  "v1alpha1",
		Resource: "swarmtasks",
	}
)

// NewSwarmClient creates a new swarm client
func NewSwarmClient(configFlags *genericclioptions.ConfigFlags) (*SwarmClient, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get REST config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	namespace, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return &SwarmClient{
		dynamicClient: dynamicClient,
		namespace:     namespace,
	}, nil
}

// Swarm operations

// Create creates a new swarm
func (c *SwarmClient) Create(ctx context.Context, swarm *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmGVR).Namespace(c.namespace).Create(ctx, swarm, opts)
}

// Get retrieves a swarm by name
func (c *SwarmClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmGVR).Namespace(c.namespace).Get(ctx, name, opts)
}

// List lists all swarms
func (c *SwarmClient) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(swarmGVR).Namespace(c.namespace).List(ctx, opts)
}

// Update updates a swarm
func (c *SwarmClient) Update(ctx context.Context, swarm *unstructured.Unstructured, opts metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmGVR).Namespace(c.namespace).Update(ctx, swarm, opts)
}

// Patch patches a swarm
func (c *SwarmClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmGVR).Namespace(c.namespace).Patch(ctx, name, pt, data, opts)
}

// Delete deletes a swarm
func (c *SwarmClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.dynamicClient.Resource(swarmGVR).Namespace(c.namespace).Delete(ctx, name, opts)
}

// Agent operations

// ListAgents lists all agents for a swarm
func (c *SwarmClient) ListAgents(ctx context.Context, swarmName string, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	if opts.LabelSelector != "" {
		opts.LabelSelector = fmt.Sprintf("%s,swarm.io/swarm=%s", opts.LabelSelector, swarmName)
	} else {
		opts.LabelSelector = fmt.Sprintf("swarm.io/swarm=%s", swarmName)
	}
	return c.dynamicClient.Resource(swarmAgentGVR).Namespace(c.namespace).List(ctx, opts)
}

// GetAgent retrieves an agent by name
func (c *SwarmClient) GetAgent(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmAgentGVR).Namespace(c.namespace).Get(ctx, name, opts)
}

// Task operations

// CreateTask creates a new task
func (c *SwarmClient) CreateTask(ctx context.Context, task *unstructured.Unstructured, opts metav1.CreateOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmTaskGVR).Namespace(c.namespace).Create(ctx, task, opts)
}

// GetTask retrieves a task by name
func (c *SwarmClient) GetTask(ctx context.Context, name string, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmTaskGVR).Namespace(c.namespace).Get(ctx, name, opts)
}

// ListTasks lists all tasks
func (c *SwarmClient) ListTasks(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(swarmTaskGVR).Namespace(c.namespace).List(ctx, opts)
}

// PatchTaskStatus patches a task's status
func (c *SwarmClient) PatchTaskStatus(ctx context.Context, name string, data []byte, opts metav1.PatchOptions) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(swarmTaskGVR).Namespace(c.namespace).Patch(ctx, name, types.MergePatchType, data, opts, "status")
}

// DeleteTask deletes a task
func (c *SwarmClient) DeleteTask(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.dynamicClient.Resource(swarmTaskGVR).Namespace(c.namespace).Delete(ctx, name, opts)
}