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
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	scaleExample = templates.Examples(`
		# Scale a swarm to 10 agents
		kubectl swarm scale my-swarm --replicas 10

		# Scale multiple swarms
		kubectl swarm scale swarm1 swarm2 --replicas 5

		# Scale with auto-adjust based on workload
		kubectl swarm scale my-swarm --replicas 10 --auto-adjust

		# Scale down to minimum agents
		kubectl swarm scale my-swarm --replicas 1`)
)

type ScaleOptions struct {
	genericclioptions.IOStreams

	Names      []string
	Namespace  string
	Replicas   int32
	AutoAdjust bool

	configFlags *genericclioptions.ConfigFlags
}

func NewScaleOptions(streams genericclioptions.IOStreams) *ScaleOptions {
	return &ScaleOptions{
		IOStreams:   streams,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdScale(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewScaleOptions(streams)

	cmd := &cobra.Command{
		Use:     "scale NAME [NAME...] --replicas=COUNT",
		Short:   "Scale swarm agents up or down",
		Long:    templates.LongDesc(`Scale the number of agents in one or more swarms.`),
		Example: scaleExample,
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			o.Names = args
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

	cmd.Flags().Int32Var(&o.Replicas, "replicas", 0, "The new number of agents")
	cmd.MarkFlagRequired("replicas")
	cmd.Flags().BoolVar(&o.AutoAdjust, "auto-adjust", false, "Enable auto-adjustment based on workload")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *ScaleOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *ScaleOptions) Validate() error {
	if len(o.Names) == 0 {
		return fmt.Errorf("at least one swarm name is required")
	}
	if o.Replicas < 0 {
		return fmt.Errorf("replicas cannot be negative")
	}
	return nil
}

func (o *ScaleOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Scale each swarm
	for _, name := range o.Names {
		if err := o.scaleSwarm(ctx, swarmClient, name); err != nil {
			fmt.Fprintf(o.ErrOut, "Error scaling %s: %v\n", name, err)
			continue
		}
		fmt.Fprintf(o.Out, "swarm.swarm.io/%s scaled to %d agents\n", name, o.Replicas)
	}

	return nil
}

func (o *ScaleOptions) scaleSwarm(ctx context.Context, client *client.SwarmClient, name string) error {
	// Create patch to update the agent count
	patch := []byte(fmt.Sprintf(`{"spec":{"agents":{"current":%d,"autoAdjust":%t}}}`, o.Replicas, o.AutoAdjust))

	_, err := client.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}