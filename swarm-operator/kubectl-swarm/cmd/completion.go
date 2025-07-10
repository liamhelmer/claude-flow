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
	"os"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	completionExample = templates.Examples(`
		# Load completions for bash
		source <(kubectl swarm completion bash)

		# Load completions for zsh
		source <(kubectl swarm completion zsh)

		# Load completions for fish
		kubectl swarm completion fish | source

		# Load completions for PowerShell
		kubectl swarm completion powershell | Out-String | Invoke-Expression

		# To load completions for each session, execute once:
		# Linux:
		kubectl swarm completion bash > /etc/bash_completion.d/kubectl-swarm

		# macOS:
		kubectl swarm completion bash > /usr/local/etc/bash_completion.d/kubectl-swarm

		# zsh:
		echo 'source <(kubectl swarm completion zsh)' >> ~/.zshrc`)
)

func NewCmdCompletion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: templates.LongDesc(`
			Generate completion script for kubectl-swarm for the specified shell.
			
			The shell completion scripts provide tab completion support for all
			kubectl-swarm commands, flags, and resource names.`),
		Example:               completionExample,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	return cmd
}