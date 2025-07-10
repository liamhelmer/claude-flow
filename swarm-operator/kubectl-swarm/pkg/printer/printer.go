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

package printer

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// Printer interface for different output formats
type Printer interface {
	PrintSwarm(swarm *unstructured.Unstructured) error
	PrintSwarmList(swarms *unstructured.UnstructuredList) error
	PrintSwarmDetailed(swarm *unstructured.Unstructured, agents *unstructured.UnstructuredList) error
	PrintTask(task *unstructured.Unstructured) error
	PrintTaskList(tasks *unstructured.UnstructuredList) error
}

// TablePrinter prints resources in table format
type TablePrinter struct {
	out io.Writer
}

// NewTablePrinter creates a new table printer
func NewTablePrinter(out io.Writer) *TablePrinter {
	return &TablePrinter{out: out}
}

// PrintSwarm prints a single swarm in table format
func (p *TablePrinter) PrintSwarm(swarm *unstructured.Unstructured) error {
	w := tabwriter.NewWriter(p.out, 0, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tTOPOLOGY\tAGENTS\tSTATUS\tAGE")
	p.printSwarmRow(w, swarm)

	return nil
}

// PrintSwarmList prints a list of swarms in table format
func (p *TablePrinter) PrintSwarmList(swarms *unstructured.UnstructuredList) error {
	w := tabwriter.NewWriter(p.out, 0, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tTOPOLOGY\tAGENTS\tSTATUS\tAGE")
	
	for _, swarm := range swarms.Items {
		p.printSwarmRow(w, &swarm)
	}

	return nil
}

// PrintSwarmDetailed prints detailed swarm information
func (p *TablePrinter) PrintSwarmDetailed(swarm *unstructured.Unstructured, agents *unstructured.UnstructuredList) error {
	// Print swarm summary
	if err := p.PrintSwarm(swarm); err != nil {
		return err
	}

	fmt.Fprintln(p.out, "\nAgents:")
	
	if len(agents.Items) == 0 {
		fmt.Fprintln(p.out, "  No agents found")
		return nil
	}

	// Print agents table
	w := tabwriter.NewWriter(p.out, 2, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "  NAME\tTYPE\tSTATUS\tHEALTH\tTASKS\tAGE")
	
	for _, agent := range agents.Items {
		p.printAgentRow(w, &agent)
	}

	return nil
}

// PrintTask prints a single task
func (p *TablePrinter) PrintTask(task *unstructured.Unstructured) error {
	w := tabwriter.NewWriter(p.out, 0, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tDESCRIPTION\tSTATUS\tPROGRESS\tAGE")
	p.printTaskRow(w, task)

	// Print detailed status
	status, _ := task.Object["status"].(map[string]interface{})
	if message, ok := status["message"].(string); ok && message != "" {
		fmt.Fprintf(p.out, "\nMessage: %s\n", message)
	}

	// Print assigned agents
	if assignedAgents, ok := status["assignedAgents"].([]interface{}); ok && len(assignedAgents) > 0 {
		fmt.Fprintln(p.out, "\nAssigned Agents:")
		for _, agent := range assignedAgents {
			fmt.Fprintf(p.out, "  - %s\n", agent)
		}
	}

	return nil
}

// PrintTaskList prints a list of tasks
func (p *TablePrinter) PrintTaskList(tasks *unstructured.UnstructuredList) error {
	w := tabwriter.NewWriter(p.out, 0, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tDESCRIPTION\tSTATUS\tPROGRESS\tAGE")
	
	for _, task := range tasks.Items {
		p.printTaskRow(w, &task)
	}

	return nil
}

// Helper methods

func (p *TablePrinter) printSwarmRow(w io.Writer, swarm *unstructured.Unstructured) {
	name := swarm.GetName()
	
	spec, _ := swarm.Object["spec"].(map[string]interface{})
	topology, _ := spec["topology"].(string)
	
	status, _ := swarm.Object["status"].(map[string]interface{})
	phase, _ := status["phase"].(string)
	agents, _ := status["agents"].(map[string]interface{})
	active, _ := agents["active"].(int64)
	total, _ := agents["total"].(int64)
	
	age := p.getAge(swarm.GetCreationTimestamp().Time)
	
	fmt.Fprintf(w, "%s\t%s\t%d/%d\t%s\t%s\n", name, topology, active, total, phase, age)
}

func (p *TablePrinter) printAgentRow(w io.Writer, agent *unstructured.Unstructured) {
	name := agent.GetName()
	
	spec, _ := agent.Object["spec"].(map[string]interface{})
	agentType, _ := spec["type"].(string)
	
	status, _ := agent.Object["status"].(map[string]interface{})
	phase, _ := status["phase"].(string)
	health, _ := status["health"].(string)
	taskCount, _ := status["taskCount"].(int64)
	
	age := p.getAge(agent.GetCreationTimestamp().Time)
	
	fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%d\t%s\n", name, agentType, phase, health, taskCount, age)
}

func (p *TablePrinter) printTaskRow(w io.Writer, task *unstructured.Unstructured) {
	name := task.GetName()
	
	spec, _ := task.Object["spec"].(map[string]interface{})
	taskSpec, _ := spec["task"].(map[string]interface{})
	description, _ := taskSpec["description"].(string)
	
	// Truncate long descriptions
	if len(description) > 50 {
		description = description[:47] + "..."
	}
	
	status, _ := task.Object["status"].(map[string]interface{})
	phase, _ := status["phase"].(string)
	progress, _ := status["progress"].(int64)
	
	age := p.getAge(task.GetCreationTimestamp().Time)
	
	fmt.Fprintf(w, "%s\t%s\t%s\t%d%%\t%s\n", name, description, phase, progress, age)
}

func (p *TablePrinter) getAge(created time.Time) string {
	duration := time.Since(created)
	
	if duration.Hours() > 24 {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else if duration.Hours() > 1 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else if duration.Minutes() > 1 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	return fmt.Sprintf("%ds", int(duration.Seconds()))
}

// JSONPrinter prints resources in JSON format
type JSONPrinter struct {
	out io.Writer
}

// NewJSONPrinter creates a new JSON printer
func NewJSONPrinter(out io.Writer) *JSONPrinter {
	return &JSONPrinter{out: out}
}

// PrintSwarm prints a swarm in JSON format
func (p *JSONPrinter) PrintSwarm(swarm *unstructured.Unstructured) error {
	return p.printJSON(swarm.Object)
}

// PrintSwarmList prints a swarm list in JSON format
func (p *JSONPrinter) PrintSwarmList(swarms *unstructured.UnstructuredList) error {
	return p.printJSON(swarms.Object)
}

// PrintSwarmDetailed prints detailed swarm info in JSON format
func (p *JSONPrinter) PrintSwarmDetailed(swarm *unstructured.Unstructured, agents *unstructured.UnstructuredList) error {
	detailed := map[string]interface{}{
		"swarm":  swarm.Object,
		"agents": agents.Object,
	}
	return p.printJSON(detailed)
}

// PrintTask prints a task in JSON format
func (p *JSONPrinter) PrintTask(task *unstructured.Unstructured) error {
	return p.printJSON(task.Object)
}

// PrintTaskList prints a task list in JSON format
func (p *JSONPrinter) PrintTaskList(tasks *unstructured.UnstructuredList) error {
	return p.printJSON(tasks.Object)
}

func (p *JSONPrinter) printJSON(obj interface{}) error {
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(p.out, string(data))
	return nil
}

// YAMLPrinter prints resources in YAML format
type YAMLPrinter struct {
	out io.Writer
}

// NewYAMLPrinter creates a new YAML printer
func NewYAMLPrinter(out io.Writer) *YAMLPrinter {
	return &YAMLPrinter{out: out}
}

// PrintSwarm prints a swarm in YAML format
func (p *YAMLPrinter) PrintSwarm(swarm *unstructured.Unstructured) error {
	return p.printYAML(swarm.Object)
}

// PrintSwarmList prints a swarm list in YAML format
func (p *YAMLPrinter) PrintSwarmList(swarms *unstructured.UnstructuredList) error {
	return p.printYAML(swarms.Object)
}

// PrintSwarmDetailed prints detailed swarm info in YAML format
func (p *YAMLPrinter) PrintSwarmDetailed(swarm *unstructured.Unstructured, agents *unstructured.UnstructuredList) error {
	detailed := map[string]interface{}{
		"swarm":  swarm.Object,
		"agents": agents.Object,
	}
	return p.printYAML(detailed)
}

// PrintTask prints a task in YAML format
func (p *YAMLPrinter) PrintTask(task *unstructured.Unstructured) error {
	return p.printYAML(task.Object)
}

// PrintTaskList prints a task list in YAML format
func (p *YAMLPrinter) PrintTaskList(tasks *unstructured.UnstructuredList) error {
	return p.printYAML(tasks.Object)
}

func (p *YAMLPrinter) printYAML(obj interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}
	fmt.Fprint(p.out, string(data))
	return nil
}