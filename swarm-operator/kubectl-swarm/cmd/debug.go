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
	"github.com/claude-flow/kubectl-swarm/pkg/util"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	debugExample = templates.Examples(`
		# Debug a swarm
		kubectl swarm debug my-swarm

		# Debug with verbose output
		kubectl swarm debug my-swarm --verbose

		# Debug specific components
		kubectl swarm debug my-swarm --component agents

		# Export debug information to file
		kubectl swarm debug my-swarm --export debug-report.yaml

		# Run diagnostic tests
		kubectl swarm debug my-swarm --run-tests`)
)

type DebugOptions struct {
	genericclioptions.IOStreams

	SwarmName  string
	Namespace  string
	Verbose    bool
	Component  string
	Export     string
	RunTests   bool

	configFlags *genericclioptions.ConfigFlags
}

func NewDebugOptions(streams genericclioptions.IOStreams) *DebugOptions {
	return &DebugOptions{
		IOStreams:   streams,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdDebug(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDebugOptions(streams)

	cmd := &cobra.Command{
		Use:     "debug SWARM-NAME",
		Short:   "Debug swarm issues",
		Long:    templates.LongDesc(`Debug swarm issues by collecting diagnostic information and running tests.`),
		Example: debugExample,
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

	cmd.Flags().BoolVarP(&o.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().StringVar(&o.Component, "component", "", "Debug specific component (agents, tasks, network)")
	cmd.Flags().StringVar(&o.Export, "export", "", "Export debug information to file")
	cmd.Flags().BoolVar(&o.RunTests, "run-tests", false, "Run diagnostic tests")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DebugOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *DebugOptions) Validate() error {
	if o.Component != "" {
		validComponents := map[string]bool{
			"agents":  true,
			"tasks":   true,
			"network": true,
			"storage": true,
			"all":     true,
		}
		if !validComponents[o.Component] {
			return fmt.Errorf("invalid component: %s", o.Component)
		}
	}
	return nil
}

func (o *DebugOptions) Run(ctx context.Context) error {
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

	fmt.Fprintf(o.Out, "🔍 Debugging swarm: %s\n\n", o.SwarmName)

	// Collect debug information
	debugInfo := &util.DebugInfo{
		SwarmName: o.SwarmName,
		Namespace: o.Namespace,
		Timestamp: metav1.Now(),
	}

	// 1. Check swarm status
	fmt.Fprintf(o.Out, "📊 Checking swarm status...\n")
	if err := o.checkSwarmStatus(ctx, swarmClient, debugInfo); err != nil {
		fmt.Fprintf(o.ErrOut, "  ❌ Error: %v\n", err)
	}

	// 2. Check agents
	if o.Component == "" || o.Component == "agents" || o.Component == "all" {
		fmt.Fprintf(o.Out, "\n🤖 Checking agents...\n")
		if err := o.checkAgents(ctx, swarmClient, clientset, debugInfo); err != nil {
			fmt.Fprintf(o.ErrOut, "  ❌ Error: %v\n", err)
		}
	}

	// 3. Check tasks
	if o.Component == "" || o.Component == "tasks" || o.Component == "all" {
		fmt.Fprintf(o.Out, "\n📋 Checking tasks...\n")
		if err := o.checkTasks(ctx, swarmClient, debugInfo); err != nil {
			fmt.Fprintf(o.ErrOut, "  ❌ Error: %v\n", err)
		}
	}

	// 4. Check network connectivity
	if o.Component == "" || o.Component == "network" || o.Component == "all" {
		fmt.Fprintf(o.Out, "\n🌐 Checking network...\n")
		if err := o.checkNetwork(ctx, clientset, debugInfo); err != nil {
			fmt.Fprintf(o.ErrOut, "  ❌ Error: %v\n", err)
		}
	}

	// 5. Run diagnostic tests
	if o.RunTests {
		fmt.Fprintf(o.Out, "\n🧪 Running diagnostic tests...\n")
		if err := o.runDiagnosticTests(ctx, swarmClient, clientset, debugInfo); err != nil {
			fmt.Fprintf(o.ErrOut, "  ❌ Error: %v\n", err)
		}
	}

	// Print summary
	o.printSummary(debugInfo)

	// Export if requested
	if o.Export != "" {
		if err := util.ExportDebugInfo(debugInfo, o.Export); err != nil {
			return fmt.Errorf("failed to export debug info: %w", err)
		}
		fmt.Fprintf(o.Out, "\n📁 Debug information exported to: %s\n", o.Export)
	}

	return nil
}

func (o *DebugOptions) checkSwarmStatus(ctx context.Context, client *client.SwarmClient, info *util.DebugInfo) error {
	swarm, err := client.Get(ctx, o.SwarmName, metav1.GetOptions{})
	if err != nil {
		info.AddError("swarm", fmt.Sprintf("Failed to get swarm: %v", err))
		return err
	}

	status, _ := swarm.Object["status"].(map[string]interface{})
	phase, _ := status["phase"].(string)
	message, _ := status["message"].(string)

	fmt.Fprintf(o.Out, "  Status: %s\n", phase)
	if message != "" {
		fmt.Fprintf(o.Out, "  Message: %s\n", message)
	}

	if phase != "Active" {
		info.AddWarning("swarm", fmt.Sprintf("Swarm is not active: %s", phase))
	}

	info.SwarmStatus = status
	return nil
}

func (o *DebugOptions) checkAgents(ctx context.Context, swarmClient *client.SwarmClient, clientset kubernetes.Interface, info *util.DebugInfo) error {
	agents, err := swarmClient.ListAgents(ctx, o.SwarmName, metav1.ListOptions{})
	if err != nil {
		info.AddError("agents", fmt.Sprintf("Failed to list agents: %v", err))
		return err
	}

	fmt.Fprintf(o.Out, "  Total agents: %d\n", len(agents.Items))

	healthyCount := 0
	unhealthyAgents := []string{}

	for _, agent := range agents.Items {
		name, _ := agent.Object["metadata"].(map[string]interface{})["name"].(string)
		status, _ := agent.Object["status"].(map[string]interface{})
		health, _ := status["health"].(string)

		if health == "healthy" {
			healthyCount++
		} else {
			unhealthyAgents = append(unhealthyAgents, name)
		}

		// Check pod status
		pods, err := clientset.CoreV1().Pods(o.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("swarm.io/agent=%s", name),
		})
		if err != nil {
			info.AddWarning("agents", fmt.Sprintf("Failed to list pods for agent %s: %v", name, err))
			continue
		}

		if len(pods.Items) == 0 {
			info.AddError("agents", fmt.Sprintf("No pod found for agent %s", name))
		} else {
			pod := pods.Items[0]
			if pod.Status.Phase != "Running" {
				info.AddWarning("agents", fmt.Sprintf("Pod %s is not running: %s", pod.Name, pod.Status.Phase))
			}
		}
	}

	fmt.Fprintf(o.Out, "  Healthy agents: %d\n", healthyCount)
	if len(unhealthyAgents) > 0 {
		fmt.Fprintf(o.Out, "  Unhealthy agents: %s\n", strings.Join(unhealthyAgents, ", "))
		info.AddWarning("agents", fmt.Sprintf("%d unhealthy agents", len(unhealthyAgents)))
	}

	info.AgentCount = len(agents.Items)
	info.HealthyAgents = healthyCount
	return nil
}

func (o *DebugOptions) checkTasks(ctx context.Context, client *client.SwarmClient, info *util.DebugInfo) error {
	tasks, err := client.ListTasks(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("swarm.io/swarm=%s", o.SwarmName),
	})
	if err != nil {
		info.AddError("tasks", fmt.Sprintf("Failed to list tasks: %v", err))
		return err
	}

	statusCount := make(map[string]int)
	for _, task := range tasks.Items {
		status, _ := task.Object["status"].(map[string]interface{})
		phase, _ := status["phase"].(string)
		statusCount[phase]++
	}

	fmt.Fprintf(o.Out, "  Total tasks: %d\n", len(tasks.Items))
	for status, count := range statusCount {
		fmt.Fprintf(o.Out, "  %s: %d\n", status, count)
	}

	if failed := statusCount["Failed"]; failed > 0 {
		info.AddWarning("tasks", fmt.Sprintf("%d failed tasks", failed))
	}

	info.TaskCount = len(tasks.Items)
	info.TaskStatus = statusCount
	return nil
}

func (o *DebugOptions) checkNetwork(ctx context.Context, clientset kubernetes.Interface, info *util.DebugInfo) error {
	// Check if swarm service exists
	svc, err := clientset.CoreV1().Services(o.Namespace).Get(ctx, o.SwarmName, metav1.GetOptions{})
	if err != nil {
		info.AddError("network", fmt.Sprintf("Swarm service not found: %v", err))
		return err
	}

	fmt.Fprintf(o.Out, "  Service: %s\n", svc.Name)
	fmt.Fprintf(o.Out, "  Type: %s\n", svc.Spec.Type)
	if svc.Spec.ClusterIP != "" {
		fmt.Fprintf(o.Out, "  ClusterIP: %s\n", svc.Spec.ClusterIP)
	}

	// Check endpoints
	endpoints, err := clientset.CoreV1().Endpoints(o.Namespace).Get(ctx, o.SwarmName, metav1.GetOptions{})
	if err != nil {
		info.AddWarning("network", "No endpoints found")
	} else {
		endpointCount := 0
		for _, subset := range endpoints.Subsets {
			endpointCount += len(subset.Addresses)
		}
		fmt.Fprintf(o.Out, "  Active endpoints: %d\n", endpointCount)
		
		if endpointCount == 0 {
			info.AddWarning("network", "No active endpoints")
		}
	}

	return nil
}

func (o *DebugOptions) runDiagnosticTests(ctx context.Context, swarmClient *client.SwarmClient, clientset kubernetes.Interface, info *util.DebugInfo) error {
	tests := []struct {
		name string
		test func() error
	}{
		{
			name: "Agent communication test",
			test: func() error {
				// Test inter-agent communication
				return util.TestAgentCommunication(ctx, swarmClient, o.SwarmName)
			},
		},
		{
			name: "Task submission test",
			test: func() error {
				// Test task submission
				return util.TestTaskSubmission(ctx, swarmClient, o.SwarmName, o.Namespace)
			},
		},
		{
			name: "Resource availability test",
			test: func() error {
				// Test resource availability
				return util.TestResourceAvailability(ctx, clientset, o.Namespace)
			},
		},
	}

	for _, test := range tests {
		fmt.Fprintf(o.Out, "  Running %s... ", test.name)
		if err := test.test(); err != nil {
			fmt.Fprintf(o.Out, "❌ Failed: %v\n", err)
			info.AddError("tests", fmt.Sprintf("%s failed: %v", test.name, err))
		} else {
			fmt.Fprintf(o.Out, "✅ Passed\n")
		}
	}

	return nil
}

func (o *DebugOptions) printSummary(info *util.DebugInfo) {
	fmt.Fprintf(o.Out, "\n📊 Debug Summary\n")
	fmt.Fprintf(o.Out, "================\n")
	
	if len(info.Errors) == 0 && len(info.Warnings) == 0 {
		fmt.Fprintf(o.Out, "✅ No issues found!\n")
	} else {
		if len(info.Errors) > 0 {
			fmt.Fprintf(o.Out, "\n❌ Errors (%d):\n", len(info.Errors))
			for _, err := range info.Errors {
				fmt.Fprintf(o.Out, "  - [%s] %s\n", err.Component, err.Message)
			}
		}
		
		if len(info.Warnings) > 0 {
			fmt.Fprintf(o.Out, "\n⚠️  Warnings (%d):\n", len(info.Warnings))
			for _, warn := range info.Warnings {
				fmt.Fprintf(o.Out, "  - [%s] %s\n", warn.Component, warn.Message)
			}
		}
	}

	if o.Verbose {
		fmt.Fprintf(o.Out, "\n📈 Metrics:\n")
		fmt.Fprintf(o.Out, "  Agents: %d (healthy: %d)\n", info.AgentCount, info.HealthyAgents)
		fmt.Fprintf(o.Out, "  Tasks: %d\n", info.TaskCount)
		if info.TaskStatus != nil {
			fmt.Fprintf(o.Out, "  Task distribution:\n")
			for status, count := range info.TaskStatus {
				fmt.Fprintf(o.Out, "    %s: %d\n", status, count)
			}
		}
	}
}