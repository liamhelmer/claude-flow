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
	"time"

	"github.com/claude-flow/kubectl-swarm/pkg/client"
	"github.com/claude-flow/kubectl-swarm/pkg/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	statusExample = templates.Examples(`
		# Get status of all swarms
		kubectl swarm status

		# Get status of a specific swarm
		kubectl swarm status my-swarm

		# Get detailed status with agent information
		kubectl swarm status my-swarm --detailed

		# Watch status updates in real-time
		kubectl swarm status --watch

		# Output status in JSON format
		kubectl swarm status -o json

		# Output status in YAML format
		kubectl swarm status -o yaml`)
)

type StatusOptions struct {
	genericclioptions.IOStreams

	Name       string
	Namespace  string
	AllNamespaces bool
	Detailed   bool
	Watch      bool
	Output     string

	configFlags *genericclioptions.ConfigFlags
}

func NewStatusOptions(streams genericclioptions.IOStreams) *StatusOptions {
	return &StatusOptions{
		IOStreams:   streams,
		Output:      "table",
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdStatus(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewStatusOptions(streams)

	cmd := &cobra.Command{
		Use:     "status [NAME]",
		Short:   "View swarm status and agent health",
		Long:    templates.LongDesc(`Display the status of swarms including agent health and task progress.`),
		Example: statusExample,
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				o.Name = args[0]
			}
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

	cmd.Flags().BoolVarP(&o.AllNamespaces, "all-namespaces", "A", false, "List swarms across all namespaces")
	cmd.Flags().BoolVar(&o.Detailed, "detailed", false, "Show detailed information including agent status")
	cmd.Flags().BoolVarP(&o.Watch, "watch", "w", false, "Watch for status updates")
	cmd.Flags().StringVarP(&o.Output, "output", "o", o.Output, "Output format (table, json, yaml, wide)")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *StatusOptions) Complete(cmd *cobra.Command) error {
	var err error
	if !o.AllNamespaces {
		o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *StatusOptions) Validate() error {
	validOutputs := map[string]bool{
		"table": true,
		"json":  true,
		"yaml":  true,
		"wide":  true,
	}
	if !validOutputs[o.Output] {
		return fmt.Errorf("invalid output format: %s", o.Output)
	}
	return nil
}

func (o *StatusOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
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

	if o.Watch {
		return o.watchStatus(ctx, swarmClient, p)
	}

	return o.printStatus(ctx, swarmClient, p)
}

func (o *StatusOptions) printStatus(ctx context.Context, client *client.SwarmClient, p printer.Printer) error {
	if o.Name != "" {
		// Get specific swarm
		swarm, err := client.Get(ctx, o.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get swarm: %w", err)
		}

		if o.Detailed {
			// Also get agents for detailed view
			agents, err := client.ListAgents(ctx, o.Name, metav1.ListOptions{})
			if err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}
			return p.PrintSwarmDetailed(swarm, agents)
		}

		return p.PrintSwarm(swarm)
	}

	// List all swarms
	listOpts := metav1.ListOptions{}
	if o.AllNamespaces {
		o.Namespace = ""
	}
	
	swarms, err := client.List(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("failed to list swarms: %w", err)
	}

	return p.PrintSwarmList(swarms)
}

func (o *StatusOptions) watchStatus(ctx context.Context, client *client.SwarmClient, p printer.Printer) error {
	fmt.Fprintf(o.Out, "Watching swarm status (press Ctrl+C to stop)...\n\n")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Print initial status
	if err := o.printStatus(ctx, client, p); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Clear screen (simple approach)
			fmt.Fprint(o.Out, "\033[H\033[2J")
			fmt.Fprintf(o.Out, "Watching swarm status (press Ctrl+C to stop)...\n\n")
			
			// Print updated status
			if err := o.printStatus(ctx, client, p); err != nil {
				fmt.Fprintf(o.ErrOut, "Error updating status: %v\n", err)
			}
		}
	}
}