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
	"context"
	"fmt"
	"strings"

	"github.com/claude-flow/kubectl-swarm/pkg/client"
	"github.com/claude-flow/kubectl-swarm/pkg/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	taskExample = templates.Examples(`
		# Submit a task to a swarm
		kubectl swarm task submit my-swarm --task "Analyze security vulnerabilities"

		# Submit a task with high priority
		kubectl swarm task submit my-swarm --task "Critical bug fix" --priority high

		# Submit a task with dependencies
		kubectl swarm task submit my-swarm --task "Deploy application" --depends-on task-123,task-456

		# List all tasks for a swarm
		kubectl swarm task list my-swarm

		# Get task status
		kubectl swarm task status task-789

		# Cancel a running task
		kubectl swarm task cancel task-789`)
)

type TaskOptions struct {
	genericclioptions.IOStreams

	configFlags *genericclioptions.ConfigFlags
}

func NewTaskOptions(streams genericclioptions.IOStreams) *TaskOptions {
	return &TaskOptions{
		IOStreams:   streams,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdTask(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewTaskOptions(streams)

	cmd := &cobra.Command{
		Use:     "task",
		Short:   "Submit and monitor tasks",
		Long:    templates.LongDesc(`Submit tasks to swarms and monitor their execution.`),
		Example: taskExample,
	}

	// Add subcommands
	cmd.AddCommand(NewCmdTaskSubmit(streams))
	cmd.AddCommand(NewCmdTaskList(streams))
	cmd.AddCommand(NewCmdTaskStatus(streams))
	cmd.AddCommand(NewCmdTaskCancel(streams))

	return cmd
}

// Submit subcommand
type TaskSubmitOptions struct {
	genericclioptions.IOStreams

	SwarmName   string
	Task        string
	Priority    string
	DependsOn   []string
	Namespace   string
	Strategy    string
	MaxRetries  int

	configFlags *genericclioptions.ConfigFlags
}

func NewTaskSubmitOptions(streams genericclioptions.IOStreams) *TaskSubmitOptions {
	return &TaskSubmitOptions{
		IOStreams:   streams,
		Priority:    "medium",
		Strategy:    "adaptive",
		MaxRetries:  3,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdTaskSubmit(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewTaskSubmitOptions(streams)

	cmd := &cobra.Command{
		Use:   "submit SWARM-NAME",
		Short: "Submit a task to a swarm",
		Args:  cobra.ExactArgs(1),
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

	cmd.Flags().StringVar(&o.Task, "task", "", "Task description")
	cmd.MarkFlagRequired("task")
	cmd.Flags().StringVar(&o.Priority, "priority", o.Priority, "Task priority (low, medium, high, critical)")
	cmd.Flags().StringSliceVar(&o.DependsOn, "depends-on", nil, "Comma-separated list of task IDs this task depends on")
	cmd.Flags().StringVar(&o.Strategy, "strategy", o.Strategy, "Execution strategy (parallel, sequential, adaptive)")
	cmd.Flags().IntVar(&o.MaxRetries, "max-retries", o.MaxRetries, "Maximum number of retries on failure")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *TaskSubmitOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *TaskSubmitOptions) Validate() error {
	if o.Task == "" {
		return fmt.Errorf("task description is required")
	}

	validPriorities := map[string]bool{
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}
	if !validPriorities[o.Priority] {
		return fmt.Errorf("invalid priority: %s", o.Priority)
	}

	return nil
}

func (o *TaskSubmitOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Generate task name
	taskName := fmt.Sprintf("%s-task-%d", o.SwarmName, metav1.Now().Unix())

	// Create SwarmTask object
	task := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "swarm.io/v1alpha1",
			"kind":       "SwarmTask",
			"metadata": map[string]interface{}{
				"name":      taskName,
				"namespace": o.Namespace,
				"labels": map[string]interface{}{
					"swarm.io/swarm": o.SwarmName,
				},
			},
			"spec": map[string]interface{}{
				"swarmRef": map[string]interface{}{
					"name": o.SwarmName,
				},
				"task": map[string]interface{}{
					"description": o.Task,
					"priority":    o.Priority,
					"strategy":    o.Strategy,
					"maxRetries":  o.MaxRetries,
				},
			},
		},
	}

	// Add dependencies if specified
	if len(o.DependsOn) > 0 {
		spec := task.Object["spec"].(map[string]interface{})
		spec["dependencies"] = o.DependsOn
	}

	// Create the task
	created, err := swarmClient.CreateTask(ctx, task, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	fmt.Fprintf(o.Out, "Task submitted successfully!\n")
	fmt.Fprintf(o.Out, "Task ID: %s\n", created.GetName())
	fmt.Fprintf(o.Out, "\nMonitor progress:\n")
	fmt.Fprintf(o.Out, "  kubectl swarm task status %s\n", created.GetName())
	fmt.Fprintf(o.Out, "  kubectl swarm logs %s --task %s\n", o.SwarmName, created.GetName())

	return nil
}

// List subcommand
type TaskListOptions struct {
	genericclioptions.IOStreams

	SwarmName string
	Namespace string
	Status    string
	Output    string

	configFlags *genericclioptions.ConfigFlags
}

func NewTaskListOptions(streams genericclioptions.IOStreams) *TaskListOptions {
	return &TaskListOptions{
		IOStreams:   streams,
		Output:      "table",
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdTaskList(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewTaskListOptions(streams)

	cmd := &cobra.Command{
		Use:   "list SWARM-NAME",
		Short: "List tasks for a swarm",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.SwarmName = args[0]
			if err := o.Complete(cmd); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
			if err := o.Run(cmd.Context()); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
		},
	}

	cmd.Flags().StringVar(&o.Status, "status", "", "Filter by status (pending, running, completed, failed)")
	cmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "Output format (table, json, yaml)")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *TaskListOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *TaskListOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// List tasks
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("swarm.io/swarm=%s", o.SwarmName),
	}
	
	if o.Status != "" {
		listOpts.FieldSelector = fmt.Sprintf("status.phase=%s", o.Status)
	}

	tasks, err := swarmClient.ListTasks(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Create printer based on output format
	var p printer.Printer
	switch o.Output {
	case "json":
		p = printer.NewJSONPrinter(o.Out)
	case "yaml":
		p = printer.NewYAMLPrinter(o.Out)
	default:
		p = printer.NewTablePrinter(o.Out)
	}

	return p.PrintTaskList(tasks)
}

// Status subcommand
type TaskStatusOptions struct {
	genericclioptions.IOStreams

	TaskName  string
	Namespace string
	Watch     bool
	Output    string

	configFlags *genericclioptions.ConfigFlags
}

func NewTaskStatusOptions(streams genericclioptions.IOStreams) *TaskStatusOptions {
	return &TaskStatusOptions{
		IOStreams:   streams,
		Output:      "table",
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdTaskStatus(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewTaskStatusOptions(streams)

	cmd := &cobra.Command{
		Use:   "status TASK-ID",
		Short: "Get task status",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.TaskName = args[0]
			if err := o.Complete(cmd); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
			if err := o.Run(cmd.Context()); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
		},
	}

	cmd.Flags().BoolVarP(&o.Watch, "watch", "w", false, "Watch for status updates")
	cmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "Output format (table, json, yaml)")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *TaskStatusOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *TaskStatusOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get task
	task, err := swarmClient.GetTask(ctx, o.TaskName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Create printer based on output format
	var p printer.Printer
	switch o.Output {
	case "json":
		p = printer.NewJSONPrinter(o.Out)
	case "yaml":
		p = printer.NewYAMLPrinter(o.Out)
	default:
		p = printer.NewTablePrinter(o.Out)
	}

	return p.PrintTask(task)
}

// Cancel subcommand
type TaskCancelOptions struct {
	genericclioptions.IOStreams

	TaskName  string
	Namespace string

	configFlags *genericclioptions.ConfigFlags
}

func NewTaskCancelOptions(streams genericclioptions.IOStreams) *TaskCancelOptions {
	return &TaskCancelOptions{
		IOStreams:   streams,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdTaskCancel(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewTaskCancelOptions(streams)

	cmd := &cobra.Command{
		Use:   "cancel TASK-ID",
		Short: "Cancel a running task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.TaskName = args[0]
			if err := o.Complete(cmd); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
			if err := o.Run(cmd.Context()); err != nil {
				fmt.Fprintf(o.ErrOut, "Error: %v\n", err)
				return
			}
		},
	}

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *TaskCancelOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *TaskCancelOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Update task status to cancelled
	patch := []byte(`{"status":{"phase":"Cancelled"}}`)
	_, err = swarmClient.PatchTaskStatus(ctx, o.TaskName, patch, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	fmt.Fprintf(o.Out, "Task %s cancelled successfully\n", o.TaskName)
	return nil
}