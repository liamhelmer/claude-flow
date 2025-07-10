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

package utils

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Common condition types
const (
	// ConditionReady indicates the resource is ready for use
	ConditionReady = "Ready"
	
	// ConditionProgressing indicates the resource is progressing towards ready state
	ConditionProgressing = "Progressing"
	
	// ConditionDegraded indicates the resource is in a degraded state
	ConditionDegraded = "Degraded"
	
	// ConditionAvailable indicates the resource is available
	ConditionAvailable = "Available"
	
	// ConditionReconciling indicates the resource is being reconciled
	ConditionReconciling = "Reconciling"
)

// Common condition reasons
const (
	// ReasonInitializing indicates the resource is initializing
	ReasonInitializing = "Initializing"
	
	// ReasonReady indicates the resource is ready
	ReasonReady = "Ready"
	
	// ReasonFailed indicates an operation failed
	ReasonFailed = "Failed"
	
	// ReasonInProgress indicates an operation is in progress
	ReasonInProgress = "InProgress"
	
	// ReasonCompleted indicates an operation completed successfully
	ReasonCompleted = "Completed"
	
	// ReasonTimeout indicates an operation timed out
	ReasonTimeout = "Timeout"
	
	// ReasonResourcesNotAvailable indicates required resources are not available
	ReasonResourcesNotAvailable = "ResourcesNotAvailable"
	
	// ReasonConfigurationError indicates a configuration error
	ReasonConfigurationError = "ConfigurationError"
)

// ConditionHelper provides utility functions for managing conditions
type ConditionHelper struct {
	conditions *[]metav1.Condition
}

// NewConditionHelper creates a new condition helper
func NewConditionHelper(conditions *[]metav1.Condition) *ConditionHelper {
	return &ConditionHelper{
		conditions: conditions,
	}
}

// SetCondition sets a condition with the given parameters
func (h *ConditionHelper) SetCondition(conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(h.conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	})
}

// SetReadyCondition sets the Ready condition
func (h *ConditionHelper) SetReadyCondition(status metav1.ConditionStatus, reason, message string) {
	h.SetCondition(ConditionReady, status, reason, message)
}

// SetProgressingCondition sets the Progressing condition
func (h *ConditionHelper) SetProgressingCondition(status metav1.ConditionStatus, reason, message string) {
	h.SetCondition(ConditionProgressing, status, reason, message)
}

// SetDegradedCondition sets the Degraded condition
func (h *ConditionHelper) SetDegradedCondition(status metav1.ConditionStatus, reason, message string) {
	h.SetCondition(ConditionDegraded, status, reason, message)
}

// RemoveCondition removes a condition by type
func (h *ConditionHelper) RemoveCondition(conditionType string) {
	meta.RemoveStatusCondition(h.conditions, conditionType)
}

// GetCondition returns the condition with the given type
func (h *ConditionHelper) GetCondition(conditionType string) *metav1.Condition {
	return meta.FindStatusCondition(*h.conditions, conditionType)
}

// IsConditionTrue returns true if the condition is present and true
func (h *ConditionHelper) IsConditionTrue(conditionType string) bool {
	condition := h.GetCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionTrue
}

// IsConditionFalse returns true if the condition is present and false
func (h *ConditionHelper) IsConditionFalse(conditionType string) bool {
	condition := h.GetCondition(conditionType)
	return condition != nil && condition.Status == metav1.ConditionFalse
}

// IsReady returns true if the Ready condition is true
func (h *ConditionHelper) IsReady() bool {
	return h.IsConditionTrue(ConditionReady)
}

// IsProgressing returns true if the Progressing condition is true
func (h *ConditionHelper) IsProgressing() bool {
	return h.IsConditionTrue(ConditionProgressing)
}

// IsDegraded returns true if the Degraded condition is true
func (h *ConditionHelper) IsDegraded() bool {
	return h.IsConditionTrue(ConditionDegraded)
}

// MarkReady marks the resource as ready
func (h *ConditionHelper) MarkReady(message string) {
	h.SetReadyCondition(metav1.ConditionTrue, ReasonReady, message)
	h.SetProgressingCondition(metav1.ConditionFalse, ReasonCompleted, "Operation completed")
	h.RemoveCondition(ConditionDegraded)
}

// MarkNotReady marks the resource as not ready
func (h *ConditionHelper) MarkNotReady(reason, message string) {
	h.SetReadyCondition(metav1.ConditionFalse, reason, message)
}

// MarkProgressing marks the resource as progressing
func (h *ConditionHelper) MarkProgressing(reason, message string) {
	h.SetProgressingCondition(metav1.ConditionTrue, reason, message)
	h.SetReadyCondition(metav1.ConditionFalse, ReasonInProgress, "Resource is being configured")
}

// MarkDegraded marks the resource as degraded
func (h *ConditionHelper) MarkDegraded(reason, message string) {
	h.SetDegradedCondition(metav1.ConditionTrue, reason, message)
}

// MarkFailed marks the resource as failed
func (h *ConditionHelper) MarkFailed(reason, message string) {
	h.SetReadyCondition(metav1.ConditionFalse, reason, message)
	h.SetProgressingCondition(metav1.ConditionFalse, ReasonFailed, "Operation failed")
	h.SetDegradedCondition(metav1.ConditionTrue, reason, message)
}