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

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/claude-flow/kubectl-swarm/pkg/client"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	logsExample = templates.Examples(`
		# View logs from all agents in a swarm
		kubectl swarm logs my-swarm

		# Follow logs in real-time
		kubectl swarm logs my-swarm --follow

		# View logs for a specific task
		kubectl swarm logs my-swarm --task task-123

		# View logs from specific agent types
		kubectl swarm logs my-swarm --agent-type researcher,coder

		# View logs with timestamps
		kubectl swarm logs my-swarm --timestamps

		# Tail last 100 lines
		kubectl swarm logs my-swarm --tail 100`)
)

type LogsOptions struct {
	genericclioptions.IOStreams

	SwarmName  string
	Namespace  string
	Follow     bool
	Tail       int64
	Timestamps bool
	Task       string
	AgentTypes []string
	Since      string

	configFlags *genericclioptions.ConfigFlags
}

func NewLogsOptions(streams genericclioptions.IOStreams) *LogsOptions {
	return &LogsOptions{
		IOStreams:   streams,
		Tail:        -1,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdLogs(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewLogsOptions(streams)

	cmd := &cobra.Command{
		Use:     "logs SWARM-NAME",
		Short:   "View aggregated logs from agents",
		Long:    templates.LongDesc(`View logs from all agents in a swarm with optional filtering.`),
		Example: logsExample,
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.SwarmName = args[0]
			if err := o.Complete(cmd); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
			if err := o.Validate(); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
			if err := o.Run(cmd.Context()); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
		},
	}

	cmd.Flags().BoolVarP(&o.Follow, "follow", "f", false, "Follow log output")
	cmd.Flags().Int64Var(&o.Tail, "tail", o.Tail, "Lines of recent log to display (-1 for all)")
	cmd.Flags().BoolVar(&o.Timestamps, "timestamps", false, "Include timestamps in log output")
	cmd.Flags().StringVar(&o.Task, "task", "", "Filter logs by task ID")
	cmd.Flags().StringSliceVar(&o.AgentTypes, "agent-type", nil, "Filter logs by agent type")
	cmd.Flags().StringVar(&o.Since, "since", "", "Only return logs newer than a relative duration (e.g., 5m, 2h)")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *LogsOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *LogsOptions) Validate() error {
	if o.SwarmName == "" {
		return fmt.Errorf("swarm name is required")
	}
	return nil
}

func (o *LogsOptions) Run(ctx context.Context) error {
	// Create Kubernetes clients
	config, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create swarm client: %w", err)
	}

	// Get agents for the swarm
	agents, err := swarmClient.ListAgents(ctx, o.SwarmName, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if len(agents.Items) == 0 {
		fmt.Fprintf(o.Out, "No agents found for swarm %s\n", o.SwarmName)
		return nil
	}

	// Filter agents if needed
	filteredAgents := o.filterAgents(agents.Items)
	if len(filteredAgents) == 0 {
		fmt.Fprintf(o.Out, "No agents match the specified filters\n")
		return nil
	}

	// Create log options
	podLogOpts := &corev1.PodLogOptions{
		Follow:     o.Follow,
		Timestamps: o.Timestamps,
	}

	if o.Tail > 0 {
		podLogOpts.TailLines = &o.Tail
	}

	if o.Since != "" {
		podLogOpts.SinceTime = &metav1.Time{}
		// Parse duration and set SinceTime
	}

	// Stream logs from all agents
	var wg sync.WaitGroup
	for _, agent := range filteredAgents {
		agentName, _ := agent.Object["metadata"].(map[string]interface{})["name"].(string)
		agentType := o.getAgentType(agent)

		wg.Add(1)
		go func(name, agentType string) {
			defer wg.Done()
			o.streamAgentLogs(ctx, clientset, name, agentType, podLogOpts)
		}(agentName, agentType)
	}

	wg.Wait()
	return nil
}

func (o *LogsOptions) filterAgents(agents []unstructured.Unstructured) []unstructured.Unstructured {
	if len(o.AgentTypes) == 0 && o.Task == "" {
		return agents
	}

	var filtered []unstructured.Unstructured
	agentTypeMap := make(map[string]bool)
	for _, at := range o.AgentTypes {
		agentTypeMap[at] = true
	}

	for _, agent := range agents {
		// Check agent type filter
		if len(o.AgentTypes) > 0 {
			agentType := o.getAgentType(agent)
			if !agentTypeMap[agentType] {
				continue
			}
		}

		// Check task filter
		if o.Task != "" {
			labels, _ := agent.Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
			taskLabel, _ := labels["swarm.io/task"].(string)
			if taskLabel != o.Task {
				continue
			}
		}

		filtered = append(filtered, agent)
	}

	return filtered
}

func (o *LogsOptions) getAgentType(agent unstructured.Unstructured) string {
	spec, _ := agent.Object["spec"].(map[string]interface{})
	agentType, _ := spec["type"].(string)
	return agentType
}

func (o *LogsOptions) streamAgentLogs(ctx context.Context, clientset kubernetes.Interface, agentName, agentType string, opts *corev1.PodLogOptions) {
	// Find pod for this agent
	pods, err := clientset.CoreV1().Pods(o.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("swarm.io/agent=%s", agentName),
	})
	if err != nil {
		fmt.Fprintf(o.ErrOut, "[%s] Error listing pods: %v\n", agentName, err)
		return
	}

	if len(pods.Items) == 0 {
		fmt.Fprintf(o.ErrOut, "[%s] No pods found\n", agentName)
		return
	}

	// Stream logs from the first pod
	pod := pods.Items[0]
	req := clientset.CoreV1().Pods(o.Namespace).GetLogs(pod.Name, opts)
	
	stream, err := req.Stream(ctx)
	if err != nil {
		fmt.Fprintf(o.ErrOut, "[%s] Error streaming logs: %v\n", agentName, err)
		return
	}
	defer stream.Close()

	// Add prefix to each log line
	prefix := fmt.Sprintf("[%s/%s]", agentType, agentName)
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		fmt.Fprintf(o.Out, "%s %s\n", o.colorizePrefix(prefix, agentType), scanner.Text())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(o.ErrOut, "[%s] Error reading logs: %v\n", agentName, err)
	}
}

func (o *LogsOptions) colorizePrefix(prefix, agentType string) string {
	// Color codes for different agent types
	colors := map[string]string{
		"researcher":  "\033[34m", // Blue
		"coder":       "\033[32m", // Green
		"analyst":     "\033[33m", // Yellow
		"tester":      "\033[35m", // Magenta
		"coordinator": "\033[36m", // Cyan
		"default":     "\033[37m", // White
	}

	color, ok := colors[agentType]
	if !ok {
		color = colors["default"]
	}

	return fmt.Sprintf("%s%s\033[0m", color, prefix)
}