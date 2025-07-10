/*
Copyright 2025 The Claude Flow Authors.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
	"github.com/claude-flow/swarm-operator/pkg/topology"
)

const (
	swarmClusterFinalizer = "swarm.claudeflow.io/finalizer"
	
	// Condition types
	ConditionTypeReady       = "Ready"
	ConditionTypeProgressing = "Progressing"
	ConditionTypeDegraded    = "Degraded"
	
	// Reason codes
	ReasonInitializing     = "Initializing"
	ReasonScaling          = "Scaling"
	ReasonReady            = "Ready"
	ReasonAgentsFailed     = "AgentsFailed"
	ReasonInsufficientAgents = "InsufficientAgents"
)

// SwarmClusterReconciler reconciles a SwarmCluster object
type SwarmClusterReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	Recorder          record.EventRecorder
	SwarmNamespace    string
	HiveMindNamespace string
}

// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *SwarmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the SwarmCluster instance
	swarmCluster := &swarmv1alpha1.SwarmCluster{}
	err := r.Get(ctx, req.NamespacedName, swarmCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("SwarmCluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get SwarmCluster")
		return ctrl.Result{}, err
	}

	// Check if the swarmCluster instance is marked to be deleted
	if swarmCluster.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(swarmCluster, swarmClusterFinalizer) {
			// Run finalization logic
			if err := r.finalizeSwarmCluster(ctx, swarmCluster); err != nil {
				log.Error(err, "Failed to finalize SwarmCluster")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(swarmCluster, swarmClusterFinalizer)
			err := r.Update(ctx, swarmCluster)
			if err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(swarmCluster, swarmClusterFinalizer) {
		controllerutil.AddFinalizer(swarmCluster, swarmClusterFinalizer)
		err = r.Update(ctx, swarmCluster)
		if err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Initialize status if needed
	if swarmCluster.Status.Phase == "" {
		swarmCluster.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, swarmCluster); err != nil {
			log.Error(err, "Failed to update SwarmCluster status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the swarm based on current phase
	switch swarmCluster.Status.Phase {
	case "Pending":
		return r.handlePendingPhase(ctx, swarmCluster)
	case "Initializing":
		return r.handleInitializingPhase(ctx, swarmCluster)
	case "Running":
		return r.handleRunningPhase(ctx, swarmCluster)
	case "Scaling":
		return r.handleScalingPhase(ctx, swarmCluster)
	case "Failed":
		return r.handleFailedPhase(ctx, swarmCluster)
	default:
		log.Info("Unknown phase, setting to Pending", "phase", swarmCluster.Status.Phase)
		swarmCluster.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, swarmCluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
}

// handlePendingPhase transitions from Pending to Initializing
func (r *SwarmClusterReconciler) handlePendingPhase(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Pending phase")

	// Update phase to Initializing
	swarmCluster.Status.Phase = "Initializing"
	swarmCluster.Status.ActiveAgents = 0
	swarmCluster.Status.ReadyAgents = 0

	// Set initial conditions
	meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeProgressing,
		Status:             metav1.ConditionTrue,
		Reason:             ReasonInitializing,
		Message:            "SwarmCluster is being initialized",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, swarmCluster); err != nil {
		log.Error(err, "Failed to update status to Initializing")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(swarmCluster, corev1.EventTypeNormal, "Initializing", "SwarmCluster initialization started")
	return ctrl.Result{Requeue: true}, nil
}

// handleInitializingPhase creates initial agents and sets up topology
func (r *SwarmClusterReconciler) handleInitializingPhase(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Initializing phase")

	// Get current agents
	agentList := &swarmv1alpha1.AgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(swarmCluster.Namespace), 
		client.MatchingLabels{"swarm-cluster": swarmCluster.Name}); err != nil {
		log.Error(err, "Failed to list agents")
		return ctrl.Result{}, err
	}

	// Calculate desired agent count (start with minimum)
	desiredAgents := int(swarmCluster.Spec.MinAgents)
	if desiredAgents == 0 {
		desiredAgents = 1
	}

	currentAgents := len(agentList.Items)
	log.Info("Agent count", "current", currentAgents, "desired", desiredAgents)

	// Create missing agents
	if currentAgents < desiredAgents {
		for i := currentAgents; i < desiredAgents; i++ {
			agent := r.constructAgentForSwarmCluster(swarmCluster, i)
			if err := controllerutil.SetControllerReference(swarmCluster, agent, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, agent); err != nil {
				log.Error(err, "Failed to create agent", "agent", agent.Name)
				return ctrl.Result{}, err
			}
			log.Info("Created agent", "agent", agent.Name)
		}
		
		// Requeue to check agent status
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Check if all agents are ready
	readyAgents := 0
	for _, agent := range agentList.Items {
		if agent.Status.Phase == "Ready" {
			readyAgents++
		}
	}

	// Update status
	swarmCluster.Status.ActiveAgents = int32(currentAgents)
	swarmCluster.Status.ReadyAgents = int32(readyAgents)

	// If all initial agents are ready, transition to Running
	if readyAgents >= desiredAgents {
		swarmCluster.Status.Phase = "Running"
		
		meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonReady,
			Message:            fmt.Sprintf("SwarmCluster is ready with %d agents", readyAgents),
			LastTransitionTime: metav1.Now(),
		})
		
		meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeProgressing,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonReady,
			Message:            "SwarmCluster initialization complete",
			LastTransitionTime: metav1.Now(),
		})

		// Initialize topology
		if err := r.setupTopology(ctx, swarmCluster, agentList.Items); err != nil {
			log.Error(err, "Failed to setup topology")
			return ctrl.Result{}, err
		}

		r.Recorder.Event(swarmCluster, corev1.EventTypeNormal, "Ready", 
			fmt.Sprintf("SwarmCluster is ready with %d agents", readyAgents))
	}

	if err := r.Status().Update(ctx, swarmCluster); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Requeue to check again
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// handleRunningPhase monitors the swarm and handles auto-scaling
func (r *SwarmClusterReconciler) handleRunningPhase(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Running phase")

	// Get current agents
	agentList := &swarmv1alpha1.AgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(swarmCluster.Namespace),
		client.MatchingLabels{"swarm-cluster": swarmCluster.Name}); err != nil {
		log.Error(err, "Failed to list agents")
		return ctrl.Result{}, err
	}

	// Count ready and active agents
	readyAgents := 0
	activeAgents := 0
	var taskStats swarmv1alpha1.TaskStatistics

	for _, agent := range agentList.Items {
		if agent.Status.Phase == "Ready" || agent.Status.Phase == "Busy" {
			readyAgents++
		}
		if agent.Status.Phase != "Failed" && agent.Status.Phase != "Terminating" {
			activeAgents++
		}
		
		// Aggregate task statistics
		taskStats.SuccessfulTasks += agent.Status.CompletedTasks
		taskStats.FailedTasks += agent.Status.FailedTasks
		taskStats.QueueSize += int32(len(agent.Status.CurrentTasks))
	}
	taskStats.TotalTasks = taskStats.SuccessfulTasks + taskStats.FailedTasks

	// Update status
	swarmCluster.Status.ActiveAgents = int32(activeAgents)
	swarmCluster.Status.ReadyAgents = int32(readyAgents)
	swarmCluster.Status.TaskStats = taskStats

	// Check if we need to scale
	if swarmCluster.Spec.AutoScaling != nil && swarmCluster.Spec.AutoScaling.Enabled {
		shouldScale, scaleDirection := r.evaluateScaling(swarmCluster, agentList.Items)
		if shouldScale {
			swarmCluster.Status.Phase = "Scaling"
			swarmCluster.Status.LastScaleTime = &metav1.Time{Time: time.Now()}
			
			meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
				Type:               ConditionTypeProgressing,
				Status:             metav1.ConditionTrue,
				Reason:             ReasonScaling,
				Message:            fmt.Sprintf("Scaling %s", scaleDirection),
				LastTransitionTime: metav1.Now(),
			})
			
			if err := r.Status().Update(ctx, swarmCluster); err != nil {
				return ctrl.Result{}, err
			}
			
			r.Recorder.Event(swarmCluster, corev1.EventTypeNormal, "Scaling",
				fmt.Sprintf("Auto-scaling %s triggered", scaleDirection))
			
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// Check health
	if readyAgents < int(swarmCluster.Spec.MinAgents) {
		meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
			Type:               ConditionTypeDegraded,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonInsufficientAgents,
			Message:            fmt.Sprintf("Only %d/%d agents are ready", readyAgents, swarmCluster.Spec.MinAgents),
			LastTransitionTime: metav1.Now(),
		})
		
		r.Recorder.Event(swarmCluster, corev1.EventTypeWarning, "Degraded",
			fmt.Sprintf("Insufficient ready agents: %d/%d", readyAgents, swarmCluster.Spec.MinAgents))
	} else {
		meta.RemoveStatusCondition(&swarmCluster.Status.Conditions, ConditionTypeDegraded)
	}

	if err := r.Status().Update(ctx, swarmCluster); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Regular reconciliation interval
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// handleScalingPhase performs scaling operations
func (r *SwarmClusterReconciler) handleScalingPhase(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Scaling phase")

	// Get current agents
	agentList := &swarmv1alpha1.AgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(swarmCluster.Namespace),
		client.MatchingLabels{"swarm-cluster": swarmCluster.Name}); err != nil {
		log.Error(err, "Failed to list agents")
		return ctrl.Result{}, err
	}

	currentCount := len(agentList.Items)
	targetCount := r.calculateTargetAgentCount(swarmCluster, agentList.Items)
	
	log.Info("Scaling swarm", "current", currentCount, "target", targetCount)

	if currentCount < targetCount {
		// Scale up
		for i := currentCount; i < targetCount; i++ {
			agent := r.constructAgentForSwarmCluster(swarmCluster, i)
			if err := controllerutil.SetControllerReference(swarmCluster, agent, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, agent); err != nil {
				log.Error(err, "Failed to create agent", "agent", agent.Name)
				return ctrl.Result{}, err
			}
			log.Info("Created agent for scale-up", "agent", agent.Name)
		}
	} else if currentCount > targetCount {
		// Scale down - remove agents gracefully
		agentsToRemove := currentCount - targetCount
		removed := 0
		
		// Sort agents by task count and remove idle ones first
		for _, agent := range agentList.Items {
			if removed >= agentsToRemove {
				break
			}
			
			if agent.Status.Phase == "Ready" && len(agent.Status.CurrentTasks) == 0 {
				if err := r.Delete(ctx, &agent); err != nil {
					log.Error(err, "Failed to delete agent", "agent", agent.Name)
					continue
				}
				log.Info("Deleted agent for scale-down", "agent", agent.Name)
				removed++
			}
		}
	}

	// Transition back to Running
	swarmCluster.Status.Phase = "Running"
	meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeProgressing,
		Status:             metav1.ConditionFalse,
		Reason:             ReasonReady,
		Message:            "Scaling complete",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, swarmCluster); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(swarmCluster, corev1.EventTypeNormal, "ScalingComplete",
		fmt.Sprintf("Scaled from %d to %d agents", currentCount, targetCount))

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// handleFailedPhase attempts to recover from failures
func (r *SwarmClusterReconciler) handleFailedPhase(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Failed phase")

	// Attempt recovery by transitioning to Initializing
	swarmCluster.Status.Phase = "Initializing"
	
	meta.SetStatusCondition(&swarmCluster.Status.Conditions, metav1.Condition{
		Type:               ConditionTypeProgressing,
		Status:             metav1.ConditionTrue,
		Reason:             ReasonInitializing,
		Message:            "Attempting recovery",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, swarmCluster); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(swarmCluster, corev1.EventTypeNormal, "Recovery", "Attempting to recover failed SwarmCluster")
	return ctrl.Result{Requeue: true}, nil
}

// constructAgentForSwarmCluster creates an Agent resource for the SwarmCluster
func (r *SwarmClusterReconciler) constructAgentForSwarmCluster(swarmCluster *swarmv1alpha1.SwarmCluster, index int) *swarmv1alpha1.Agent {
	agentType := r.selectAgentType(swarmCluster, index)
	name := fmt.Sprintf("%s-%s-%d", swarmCluster.Name, agentType, index)

	agent := &swarmv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: swarmCluster.Namespace,
			Labels: map[string]string{
				"swarm-cluster": swarmCluster.Name,
				"agent-type":    string(agentType),
				"topology":      string(swarmCluster.Spec.Topology),
			},
		},
		Spec: swarmv1alpha1.AgentSpec{
			Type:             agentType,
			SwarmCluster:     swarmCluster.Name,
			Capabilities:     swarmCluster.Spec.AgentTemplate.Capabilities,
			CognitivePattern: r.selectCognitivePattern(swarmCluster, index),
			Resources:        swarmCluster.Spec.AgentTemplate.Resources,
		},
	}

	// Set communication spec based on topology
	agent.Spec.CommunicationEndpoints = swarmv1alpha1.CommunicationSpec{
		Protocol:         "grpc",
		Port:             8080 + int32(index),
		BroadcastEnabled: swarmCluster.Spec.Topology == swarmv1alpha1.MeshTopology,
	}

	return agent
}

// selectAgentType determines the type of agent to create based on strategy
func (r *SwarmClusterReconciler) selectAgentType(swarmCluster *swarmv1alpha1.SwarmCluster, index int) swarmv1alpha1.AgentType {
	// For specialized strategy, create different types
	if swarmCluster.Spec.Strategy == "specialized" {
		types := []swarmv1alpha1.AgentType{
			swarmv1alpha1.CoordinatorAgent,
			swarmv1alpha1.ResearcherAgent,
			swarmv1alpha1.CoderAgent,
			swarmv1alpha1.AnalystAgent,
			swarmv1alpha1.TesterAgent,
		}
		return types[index%len(types)]
	}
	
	// For balanced strategy, create a mix
	if index == 0 {
		return swarmv1alpha1.CoordinatorAgent // First agent is always coordinator
	}
	
	// Default to coder agents
	return swarmv1alpha1.CoderAgent
}

// selectCognitivePattern selects a cognitive pattern for the agent
func (r *SwarmClusterReconciler) selectCognitivePattern(swarmCluster *swarmv1alpha1.SwarmCluster, index int) swarmv1alpha1.CognitivePattern {
	if len(swarmCluster.Spec.AgentTemplate.CognitivePatterns) > 0 {
		pattern := swarmCluster.Spec.AgentTemplate.CognitivePatterns[index%len(swarmCluster.Spec.AgentTemplate.CognitivePatterns)]
		return swarmv1alpha1.CognitivePattern(pattern)
	}
	
	// Default pattern based on agent index
	patterns := []swarmv1alpha1.CognitivePattern{
		swarmv1alpha1.AdaptivePattern,
		swarmv1alpha1.SystemsPattern,
		swarmv1alpha1.ConvergentPattern,
		swarmv1alpha1.DivergentPattern,
	}
	return patterns[index%len(patterns)]
}

// setupTopology configures agent communication based on topology
func (r *SwarmClusterReconciler) setupTopology(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster, agents []swarmv1alpha1.Agent) error {
	log := log.FromContext(ctx)
	
	// Create topology manager
	topologyManager := topology.NewManager(string(swarmCluster.Spec.Topology))
	
	// Configure peer connections for each agent
	peerMap := topologyManager.CalculatePeers(agents)
	
	for i := range agents {
		agent := &agents[i]
		peers := peerMap[agent.Name]
		
		// Update agent's peer list
		agent.Spec.CommunicationEndpoints.Peers = peers
		
		if err := r.Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update agent peers", "agent", agent.Name)
			return err
		}
	}
	
	// Update topology status
	if swarmCluster.Status.TopologyStatus == nil {
		swarmCluster.Status.TopologyStatus = make(map[string]string)
	}
	swarmCluster.Status.TopologyStatus["configured"] = "true"
	swarmCluster.Status.TopologyStatus["type"] = string(swarmCluster.Spec.Topology)
	swarmCluster.Status.TopologyStatus["lastUpdate"] = time.Now().Format(time.RFC3339)
	
	return nil
}

// evaluateScaling determines if scaling is needed
func (r *SwarmClusterReconciler) evaluateScaling(swarmCluster *swarmv1alpha1.SwarmCluster, agents []swarmv1alpha1.Agent) (bool, string) {
	if swarmCluster.Spec.AutoScaling == nil || !swarmCluster.Spec.AutoScaling.Enabled {
		return false, ""
	}
	
	// Calculate average metrics
	var totalCPU float64
	var totalTasks int
	activeAgents := 0
	
	for _, agent := range agents {
		if agent.Status.Phase == "Ready" || agent.Status.Phase == "Busy" {
			activeAgents++
			totalCPU += agent.Status.Metrics.CPUUsage
			totalTasks += len(agent.Status.CurrentTasks)
		}
	}
	
	if activeAgents == 0 {
		return false, ""
	}
	
	avgCPU := totalCPU / float64(activeAgents)
	avgTasksPerAgent := float64(totalTasks) / float64(activeAgents)
	
	// Check scale up conditions
	if avgCPU > float64(swarmCluster.Spec.AutoScaling.ScaleUpThreshold) {
		if int32(activeAgents) < swarmCluster.Spec.MaxAgents {
			return true, "up"
		}
	}
	
	// Check scale down conditions
	if avgCPU < float64(swarmCluster.Spec.AutoScaling.ScaleDownThreshold) &&
		avgTasksPerAgent < 1.0 {
		if int32(activeAgents) > swarmCluster.Spec.MinAgents {
			return true, "down"
		}
	}
	
	return false, ""
}

// calculateTargetAgentCount determines the target number of agents
func (r *SwarmClusterReconciler) calculateTargetAgentCount(swarmCluster *swarmv1alpha1.SwarmCluster, agents []swarmv1alpha1.Agent) int {
	currentCount := len(agents)
	
	// Simple scaling logic - scale by 1 agent at a time
	_, direction := r.evaluateScaling(swarmCluster, agents)
	
	switch direction {
	case "up":
		targetCount := currentCount + 1
		if int32(targetCount) > swarmCluster.Spec.MaxAgents {
			return int(swarmCluster.Spec.MaxAgents)
		}
		return targetCount
	case "down":
		targetCount := currentCount - 1
		if int32(targetCount) < swarmCluster.Spec.MinAgents {
			return int(swarmCluster.Spec.MinAgents)
		}
		return targetCount
	default:
		return currentCount
	}
}

// finalizeSwarmCluster handles cleanup when SwarmCluster is deleted
func (r *SwarmClusterReconciler) finalizeSwarmCluster(ctx context.Context, swarmCluster *swarmv1alpha1.SwarmCluster) error {
	log := log.FromContext(ctx)
	
	// Delete all agents
	agentList := &swarmv1alpha1.AgentList{}
	if err := r.List(ctx, agentList, client.InNamespace(swarmCluster.Namespace),
		client.MatchingLabels{"swarm-cluster": swarmCluster.Name}); err != nil {
		log.Error(err, "Failed to list agents for cleanup")
		return err
	}
	
	for _, agent := range agentList.Items {
		if err := r.Delete(ctx, &agent); err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete agent", "agent", agent.Name)
			return err
		}
	}
	
	r.Recorder.Event(swarmCluster, corev1.EventTypeNormal, "Finalized", "SwarmCluster finalization complete")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwarmClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swarmv1alpha1.SwarmCluster{}).
		Owns(&swarmv1alpha1.Agent{}).
		Complete(r)
}