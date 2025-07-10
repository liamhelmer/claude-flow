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
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	deleteExample = templates.Examples(`
		# Delete a swarm
		kubectl swarm delete my-swarm

		# Delete multiple swarms
		kubectl swarm delete swarm1 swarm2 swarm3

		# Delete a swarm and all associated resources
		kubectl swarm delete my-swarm --cascade

		# Delete without confirmation prompt
		kubectl swarm delete my-swarm --force

		# Delete all swarms in a namespace
		kubectl swarm delete --all`)
)

type DeleteOptions struct {
	genericclioptions.IOStreams

	Names      []string
	Namespace  string
	All        bool
	Cascade    bool
	Force      bool

	configFlags *genericclioptions.ConfigFlags
}

func NewDeleteOptions(streams genericclioptions.IOStreams) *DeleteOptions {
	return &DeleteOptions{
		IOStreams:   streams,
		Cascade:     true,
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmdDelete(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewDeleteOptions(streams)

	cmd := &cobra.Command{
		Use:     "delete NAME [NAME...]",
		Short:   "Delete swarms",
		Long:    templates.LongDesc(`Delete one or more swarms and their associated resources.`),
		Example: deleteExample,
		Args: func(cmd *cobra.Command, args []string) error {
			if !o.All && len(args) == 0 {
				return fmt.Errorf("requires at least one swarm name or --all flag")
			}
			return nil
		},
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

	cmd.Flags().BoolVar(&o.All, "all", false, "Delete all swarms in the namespace")
	cmd.Flags().BoolVar(&o.Cascade, "cascade", o.Cascade, "Delete associated resources (agents, tasks)")
	cmd.Flags().BoolVarP(&o.Force, "force", "f", false, "Skip confirmation prompt")

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *DeleteOptions) Complete(cmd *cobra.Command) error {
	var err error
	o.Namespace, _, err = o.configFlags.ToRawKubeConfigLoader().Namespace()
	return err
}

func (o *DeleteOptions) Validate() error {
	if !o.All && len(o.Names) == 0 {
		return fmt.Errorf("specify swarm names or use --all flag")
	}
	return nil
}

func (o *DeleteOptions) Run(ctx context.Context) error {
	// Create Kubernetes client
	swarmClient, err := client.NewSwarmClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Get list of swarms to delete
	swarmsToDelete := o.Names

	if o.All {
		// List all swarms in namespace
		swarms, err := swarmClient.List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list swarms: %w", err)
		}

		swarmsToDelete = []string{}
		for _, swarm := range swarms.Items {
			name, _ := swarm.Object["metadata"].(map[string]interface{})["name"].(string)
			swarmsToDelete = append(swarmsToDelete, name)
		}

		if len(swarmsToDelete) == 0 {
			fmt.Fprintf(o.Out, "No swarms found in namespace %s\n", o.Namespace)
			return nil
		}
	}

	// Confirm deletion
	if !o.Force {
		fmt.Fprintf(o.Out, "You are about to delete the following swarms:\n")
		for _, name := range swarmsToDelete {
			fmt.Fprintf(o.Out, "  - %s\n", name)
		}
		if o.Cascade {
			fmt.Fprintf(o.Out, "\nThis will also delete all associated agents and tasks.\n")
		}
		fmt.Fprintf(o.Out, "\nContinue? (y/N): ")

		var response string
		fmt.Fscanln(o.In, &response)
		if !strings.HasPrefix(strings.ToLower(response), "y") {
			fmt.Fprintf(o.Out, "Deletion cancelled.\n")
			return nil
		}
	}

	// Delete swarms
	deleteOpts := metav1.DeleteOptions{}
	if o.Cascade {
		propagation := metav1.DeletePropagationForeground
		deleteOpts.PropagationPolicy = &propagation
	}

	successCount := 0
	for _, name := range swarmsToDelete {
		fmt.Fprintf(o.Out, "Deleting swarm %s...\n", name)
		
		if err := swarmClient.Delete(ctx, name, deleteOpts); err != nil {
			fmt.Fprintf(o.ErrOut, "Error deleting %s: %v\n", name, err)
			continue
		}

		successCount++
		fmt.Fprintf(o.Out, "swarm.swarm.io/%s deleted\n", name)
	}

	if successCount == len(swarmsToDelete) {
		fmt.Fprintf(o.Out, "\n✅ Successfully deleted %d swarm(s)\n", successCount)
	} else {
		fmt.Fprintf(o.Out, "\n⚠️  Deleted %d of %d swarm(s)\n", successCount, len(swarmsToDelete))
	}

	return nil
}