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
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	swarmExample = templates.Examples(`
		# Create a new swarm with mesh topology
		kubectl swarm create my-swarm --topology mesh --max-agents 5

		# Scale a swarm to 10 agents
		kubectl swarm scale my-swarm --replicas 10

		# Get the status of all swarms
		kubectl swarm status

		# Submit a task to a swarm
		kubectl swarm task submit my-swarm --task "Analyze codebase for security issues"

		# View logs from all agents in a swarm
		kubectl swarm logs my-swarm --follow

		# Debug a swarm
		kubectl swarm debug my-swarm --verbose`)
)

// NewCmdSwarm provides a cobra command for swarm operations
func NewCmdSwarm(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swarm",
		Short: "Manage AI agent swarms in Kubernetes",
		Long: templates.LongDesc(`
			Manage AI agent swarms in Kubernetes.

			This plugin provides commands to create, scale, monitor, and manage
			AI agent swarms that coordinate distributed task execution.`),
		Example: swarmExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewCmdCreate(streams))
	cmd.AddCommand(NewCmdScale(streams))
	cmd.AddCommand(NewCmdStatus(streams))
	cmd.AddCommand(NewCmdTask(streams))
	cmd.AddCommand(NewCmdLogs(streams))
	cmd.AddCommand(NewCmdDebug(streams))
	cmd.AddCommand(NewCmdDelete(streams))
	cmd.AddCommand(NewCmdCompletion())

	return cmd
}