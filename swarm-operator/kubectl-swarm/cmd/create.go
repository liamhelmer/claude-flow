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

	"github.com/claude-flow/kubectl-swarm/pkg/client"
	"github.com/claude-flow/kubectl-swarm/pkg/printer"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	createExample = templates.Examples(`
		# Create a new swarm with default settings
		kubectl swarm create my-swarm

		# Create a swarm with specific topology
		kubectl swarm create my-swarm --topology hierarchical

		# Create a swarm with custom agent limits
		kubectl swarm create my-swarm --max-agents 10 --min-agents 3

		# Create a swarm in a specific namespace
		kubectl swarm create my-swarm -n production

		# Create a swarm with interactive prompts
		kubectl swarm create --interactive`)
)

type CreateOptions struct {
	genericclioptions.IOStreams

	Name        string
	Namespace   string
	Topology    string
	MaxAgents   int32
	MinAgents   int32
	Strategy    string
	Interactive bool

	configFlags *genericclioptions.ConfigFlags
}

func NewCreateOptions(streams genericclioptions.IOStreams) *CreateOptions {
	return &CreateOptions{
		IOStreams:   streams,
		Topology:    "mesh",
		MaxAgents:   5,
		MinAgents:   1,
		Strategy:    "balanced",
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdCreate(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewCreateOptions(streams)

	cmd := &cobra.Command{
		Use:     "create [NAME]",
		Short:   "Create a new swarm",
		Long:    templates.LongDesc(`Create a new AI agent swarm with specified configuration.`),
		Example: createExample,
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

	cmd.Flags().StringVar(&o.Topology, "topology", o.Topology, "Swarm topology (mesh, hierarchical, ring, star)")
	cmd.Flags().Int32Var(&o.MaxAgents, "max-agents", o.MaxAgents, "Maximum number of agents")
	cmd.Flags().Int32Var(&o.MinAgents, "min-agents", o.MinAgents, "Minimum number of agents")
	cmd.Flags().StringVar(&o.Strategy, "strategy", o.Strategy, "Distribution strategy (balanced, specialized, adaptive)")
	cmd.Flags().BoolVarP(&o.Interactive, "interactive", "i", false, "Use interactive mode with prompts")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *CreateOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	if o.Interactive && o.Name == "" {
		// Interactive mode - prompt for values
		fmt.Fprint(o.Out, "Swarm name: ")
		fmt.Fscanln(o.In, &o.Name)
		
		fmt.Fprintf(o.Out, "Topology [%s]: ", o.Topology)
		var topology string
		fmt.Fscanln(o.In, &topology)
		if topology != "" {
			o.Topology = topology
		}
		
		fmt.Fprintf(o.Out, "Maximum agents [%d]: ", o.MaxAgents)
		var maxAgents int32
		fmt.Fscanln(o.In, &maxAgents)
		if maxAgents > 0 {
			o.MaxAgents = maxAgents
		}
	}

	return nil
}

func (o *CreateOptions) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("swarm name is required")
	}

	validTopologies := map[string]bool{
		"mesh":         true,
		"hierarchical": true,
		"ring":         true,
		"star":         true,
	}
	if !validTopologies[o.Topology] {
		return fmt.Errorf("invalid topology: %s", o.Topology)
	}

	if o.MinAgents < 1 {
		return fmt.Errorf("minimum agents must be at least 1")
	}
	if o.MaxAgents < o.MinAgents {
		return fmt.Errorf("maximum agents must be greater than or equal to minimum agents")
	}

	return nil
}

func (o *CreateOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create swarm object
	swarm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "swarm.io/v1alpha1",
			"kind":       "Swarm",
			"metadata": map[string]interface{}{
				"name":      o.Name,
				"namespace": o.Namespace,
			},
			"spec": map[string]interface{}{
				"topology": o.Topology,
				"agents": map[string]interface{}{
					"min": o.MinAgents,
					"max": o.MaxAgents,
				},
				"strategy": o.Strategy,
			},
		},
	}

	// Create the swarm
	created, err := swarmClient.Create(ctx, swarm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create swarm: %w", err)
	}

	// Print result
	p := printer.NewTablePrinter(o.Out)
	fmt.Fprintf(o.Out, "swarm.swarm.io/%s created\n", created.GetName())
	
	if o.Interactive {
		fmt.Fprintf(o.Out, "\nSwarm Details:\n")
		p.PrintSwarm(created)
		fmt.Fprintf(o.Out, "\nNext steps:\n")
		fmt.Fprintf(o.Out, "  - Check status: kubectl swarm status %s\n", o.Name)
		fmt.Fprintf(o.Out, "  - Submit task: kubectl swarm task submit %s --task \"Your task here\"\n", o.Name)
		fmt.Fprintf(o.Out, "  - View logs: kubectl swarm logs %s\n", o.Name)
	}

	return nil
}