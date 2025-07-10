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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
	"github.com/claude-flow/swarm-operator/pkg/metrics"
	"github.com/claude-flow/swarm-operator/pkg/utils"
)

const (
	agentFinalizer = "agent.swarm.claudeflow.io/finalizer"
	
	// Heartbeat interval
	heartbeatInterval = 30 * time.Second
	heartbeatTimeout  = 2 * time.Minute
)

// AgentReconciler reconciles an Agent object
type AgentReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	MetricsRecorder *metrics.MetricsRecorder
	SwarmNamespace  string
}

// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=agents/finalizers,verbs=update
// +kubebuilder:rbac:groups=swarm.claudeflow.io,resources=swarmclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	startTime := time.Now()

	// Fetch the Agent instance
	agent := &swarmv1alpha1.Agent{}
	err := r.Get(ctx, req.NamespacedName, agent)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Agent resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Agent")
		return ctrl.Result{}, err
	}

	// Record reconciliation metrics
	defer func() {
		duration := time.Since(startTime).Seconds()
		r.MetricsRecorder.RecordReconciliation("agent", duration, err)
	}()

	// Check if the agent instance is marked to be deleted
	if agent.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(agent, agentFinalizer) {
			// Run finalization logic
			if err := r.finalizeAgent(ctx, agent); err != nil {
				log.Error(err, "Failed to finalize Agent")
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(agent, agentFinalizer)
			err := r.Update(ctx, agent)
			if err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(agent, agentFinalizer) {
		controllerutil.AddFinalizer(agent, agentFinalizer)
		err = r.Update(ctx, agent)
		if err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Initialize status if needed
	if agent.Status.Phase == "" {
		agent.Status.Phase = "Pending"
		agent.Status.CompletedTasks = 0
		agent.Status.FailedTasks = 0
		agent.Status.Metrics = swarmv1alpha1.AgentMetrics{}
		
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update Agent status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Get parent SwarmCluster
	swarmCluster := &swarmv1alpha1.SwarmCluster{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      agent.Spec.SwarmCluster,
		Namespace: agent.Namespace,
	}, swarmCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "SwarmCluster not found", "swarmCluster", agent.Spec.SwarmCluster)
			return r.markAgentFailed(ctx, agent, "SwarmClusterNotFound", 
				fmt.Sprintf("SwarmCluster %s not found", agent.Spec.SwarmCluster))
		}
		return ctrl.Result{}, err
	}

	// Check if SwarmCluster is ready
	if swarmCluster.Status.Phase != "Running" && swarmCluster.Status.Phase != "Scaling" {
		log.Info("SwarmCluster not ready", "phase", swarmCluster.Status.Phase)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Reconcile the agent based on current phase
	switch agent.Status.Phase {
	case "Pending":
		return r.handlePendingPhase(ctx, agent, swarmCluster)
	case "Initializing":
		return r.handleInitializingPhase(ctx, agent, swarmCluster)
	case "Ready", "Busy":
		return r.handleActivePhase(ctx, agent, swarmCluster)
	case "Failed":
		return r.handleFailedPhase(ctx, agent, swarmCluster)
	default:
		log.Info("Unknown phase, setting to Pending", "phase", agent.Status.Phase)
		agent.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, agent); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
}

// handlePendingPhase transitions from Pending to Initializing
func (r *AgentReconciler) handlePendingPhase(ctx context.Context, agent *swarmv1alpha1.Agent, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Pending phase")

	// Update phase to Initializing
	agent.Status.Phase = "Initializing"
	agent.Status.LastHeartbeat = &metav1.Time{Time: time.Now()}

	// Initialize conditions
	condHelper := utils.NewConditionHelper(&agent.Status.Conditions)
	condHelper.MarkProgressing(utils.ReasonInitializing, "Agent is being initialized")

	// Initialize communication status if needed
	if agent.Status.CommunicationStatus == nil {
		agent.Status.CommunicationStatus = make(map[string]swarmv1alpha1.PeerStatus)
	}

	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update status to Initializing")
		return ctrl.Result{}, err
	}

	// Record metrics
	r.MetricsRecorder.RecordAgentPhase(agent.Namespace, agent.Name, string(agent.Spec.Type), agent.Status.Phase)

	r.Recorder.Event(agent, corev1.EventTypeNormal, "Initializing", "Agent initialization started")
	return ctrl.Result{Requeue: true}, nil
}

// handleInitializingPhase performs agent initialization
func (r *AgentReconciler) handleInitializingPhase(ctx context.Context, agent *swarmv1alpha1.Agent, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Initializing phase")

	// Simulate initialization steps
	// In a real implementation, this would:
	// 1. Set up communication channels
	// 2. Initialize neural models
	// 3. Load cognitive patterns
	// 4. Establish peer connections

	// Check if we have peer connections configured
	if len(agent.Spec.CommunicationEndpoints.Peers) == 0 {
		log.Info("No peers configured yet, waiting for topology setup")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Initialize peer connections
	for _, peer := range agent.Spec.CommunicationEndpoints.Peers {
		agent.Status.CommunicationStatus[peer] = swarmv1alpha1.PeerStatus{
			Connected:   false,
			LastContact: nil,
			Latency:     0,
		}
	}

	// Transition to Ready
	agent.Status.Phase = "Ready"
	agent.Status.LastHeartbeat = &metav1.Time{Time: time.Now()}

	// Update conditions
	condHelper := utils.NewConditionHelper(&agent.Status.Conditions)
	condHelper.MarkReady("Agent is ready to process tasks")

	// Initialize metrics
	agent.Status.Metrics = swarmv1alpha1.AgentMetrics{
		CPUUsage:        0.0,
		MemoryUsage:     0,
		TaskThroughput:  0.0,
		AverageTaskTime: 0,
		SuccessRate:     100.0,
	}

	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update status to Ready")
		return ctrl.Result{}, err
	}

	// Record metrics
	r.MetricsRecorder.RecordAgentPhase(agent.Namespace, agent.Name, string(agent.Spec.Type), agent.Status.Phase)
	r.MetricsRecorder.RecordPeerConnections(agent.Namespace, agent.Name, 
		string(swarmCluster.Spec.Topology), len(agent.Spec.CommunicationEndpoints.Peers))

	r.Recorder.Event(agent, corev1.EventTypeNormal, "Ready", "Agent is ready to process tasks")
	return ctrl.Result{RequeueAfter: heartbeatInterval}, nil
}

// handleActivePhase manages Ready and Busy agents
func (r *AgentReconciler) handleActivePhase(ctx context.Context, agent *swarmv1alpha1.Agent, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Active phase", "phase", agent.Status.Phase)

	// Check heartbeat timeout
	if agent.Status.LastHeartbeat != nil {
		lastHeartbeat := agent.Status.LastHeartbeat.Time
		if time.Since(lastHeartbeat) > heartbeatTimeout {
			log.Info("Agent heartbeat timeout", "lastHeartbeat", lastHeartbeat)
			return r.markAgentFailed(ctx, agent, "HeartbeatTimeout", 
				fmt.Sprintf("No heartbeat for %v", time.Since(lastHeartbeat)))
		}
	}

	// Update heartbeat
	agent.Status.LastHeartbeat = &metav1.Time{Time: time.Now()}

	// Simulate task processing
	if agent.Status.Phase == "Ready" && len(agent.Status.CurrentTasks) > 0 {
		agent.Status.Phase = "Busy"
	} else if agent.Status.Phase == "Busy" && len(agent.Status.CurrentTasks) == 0 {
		agent.Status.Phase = "Ready"
	}

	// Update peer connection status
	for peer := range agent.Status.CommunicationStatus {
		// Simulate peer connectivity
		status := agent.Status.CommunicationStatus[peer]
		status.Connected = true
		status.LastContact = &metav1.Time{Time: time.Now()}
		status.Latency = int32(5 + (time.Now().UnixNano() % 20)) // Random latency 5-25ms
		agent.Status.CommunicationStatus[peer] = status

		// Record latency metric
		r.MetricsRecorder.RecordCommunicationLatency(agent.Namespace, agent.Name, peer, float64(status.Latency))
	}

	// Update metrics (simulated)
	agent.Status.Metrics.CPUUsage = float64(20 + (time.Now().UnixNano() % 60)) // 20-80%
	agent.Status.Metrics.MemoryUsage = 100 * 1024 * 1024 // 100MB
	agent.Status.Metrics.TaskThroughput = float64(len(agent.Status.CurrentTasks)) * 60 / 5 // tasks per minute
	if agent.Status.CompletedTasks > 0 {
		agent.Status.Metrics.SuccessRate = float64(agent.Status.CompletedTasks) / 
			float64(agent.Status.CompletedTasks + agent.Status.FailedTasks) * 100
	}

	// Record metrics
	r.MetricsRecorder.RecordAgentPhase(agent.Namespace, agent.Name, string(agent.Spec.Type), agent.Status.Phase)
	r.MetricsRecorder.RecordAgentTasks(agent.Namespace, agent.Name, string(agent.Spec.Type), len(agent.Status.CurrentTasks))
	r.MetricsRecorder.RecordAgentResourceUsage(agent.Namespace, agent.Name, string(agent.Spec.Type), 
		agent.Status.Metrics.CPUUsage, agent.Status.Metrics.MemoryUsage)

	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update agent status")
		return ctrl.Result{}, err
	}

	// Regular heartbeat interval
	return ctrl.Result{RequeueAfter: heartbeatInterval}, nil
}

// handleFailedPhase attempts to recover failed agents
func (r *AgentReconciler) handleFailedPhase(ctx context.Context, agent *swarmv1alpha1.Agent, swarmCluster *swarmv1alpha1.SwarmCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Handling Failed phase")

	// Check if we should attempt recovery
	condHelper := utils.NewConditionHelper(&agent.Status.Conditions)
	failedCondition := condHelper.GetCondition(utils.ConditionReady)
	
	if failedCondition != nil && time.Since(failedCondition.LastTransitionTime.Time) > 5*time.Minute {
		// Attempt recovery after 5 minutes
		log.Info("Attempting agent recovery")
		
		agent.Status.Phase = "Initializing"
		agent.Status.CurrentTasks = []swarmv1alpha1.TaskReference{}
		condHelper.MarkProgressing(utils.ReasonInitializing, "Attempting recovery")
		
		if err := r.Status().Update(ctx, agent); err != nil {
			log.Error(err, "Failed to update status for recovery")
			return ctrl.Result{}, err
		}
		
		r.Recorder.Event(agent, corev1.EventTypeNormal, "Recovery", "Attempting to recover failed agent")
		return ctrl.Result{Requeue: true}, nil
	}

	// Wait before checking again
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// markAgentFailed marks the agent as failed
func (r *AgentReconciler) markAgentFailed(ctx context.Context, agent *swarmv1alpha1.Agent, reason, message string) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Marking agent as failed", "reason", reason)

	agent.Status.Phase = "Failed"
	
	condHelper := utils.NewConditionHelper(&agent.Status.Conditions)
	condHelper.MarkFailed(reason, message)

	if err := r.Status().Update(ctx, agent); err != nil {
		log.Error(err, "Failed to update agent status")
		return ctrl.Result{}, err
	}

	// Record metrics
	r.MetricsRecorder.RecordAgentPhase(agent.Namespace, agent.Name, string(agent.Spec.Type), agent.Status.Phase)

	r.Recorder.Event(agent, corev1.EventTypeWarning, "Failed", message)
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// finalizeAgent handles cleanup when Agent is deleted
func (r *AgentReconciler) finalizeAgent(ctx context.Context, agent *swarmv1alpha1.Agent) error {
	log := log.FromContext(ctx)
	log.Info("Finalizing agent")

	// Clean up any resources
	// In a real implementation, this would:
	// 1. Close communication channels
	// 2. Release allocated resources
	// 3. Notify peers of disconnection
	// 4. Save state if needed

	// Update metrics
	r.MetricsRecorder.RecordAgentPhase(agent.Namespace, agent.Name, string(agent.Spec.Type), "Terminating")

	r.Recorder.Event(agent, corev1.EventTypeNormal, "Finalized", "Agent finalization complete")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&swarmv1alpha1.Agent{}).
		Complete(r)
}