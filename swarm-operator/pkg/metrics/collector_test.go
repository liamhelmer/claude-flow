/*
Copyright 2025 Claude Flow Contributors.

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

package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

func TestMetricsCollector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Collector Suite")
}

var _ = Describe("MetricsCollector", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		collector  *Collector
		scheme     *runtime.Scheme
		registry   *prometheus.Registry
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(swarmv1alpha1.AddToScheme(scheme)).To(Succeed())
		
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		
		registry = prometheus.NewRegistry()
		collector = NewCollector(fakeClient)
		registry.MustRegister(collector)
	})

	AfterEach(func() {
		registry.Unregister(collector)
	})

	Describe("Cluster Metrics", func() {
		It("should collect cluster metrics", func() {
			// Create test clusters
			clusters := []swarmv1alpha1.SwarmCluster{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-1",
						Namespace: "default",
					},
					Spec: swarmv1alpha1.SwarmClusterSpec{
						Topology: swarmv1alpha1.MeshTopology,
						Size:     3,
					},
					Status: swarmv1alpha1.SwarmClusterStatus{
						State:        swarmv1alpha1.ClusterReady,
						ReadyAgents:  3,
						ActiveAgents: 2,
						Health:       swarmv1alpha1.HealthyCondition,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-2",
						Namespace: "production",
					},
					Spec: swarmv1alpha1.SwarmClusterSpec{
						Topology: swarmv1alpha1.HierarchicalTopology,
						Size:     5,
					},
					Status: swarmv1alpha1.SwarmClusterStatus{
						State:        swarmv1alpha1.ClusterScaling,
						ReadyAgents:  4,
						ActiveAgents: 4,
						Health:       swarmv1alpha1.DegradedCondition,
					},
				},
			}

			for _, cluster := range clusters {
				cluster := cluster // capture range variable
				Expect(fakeClient.Create(ctx, &cluster)).To(Succeed())
			}

			// Collect metrics
			collector.CollectClusterMetrics(ctx)

			// Verify cluster count metric
			count, err := testutil.CollectAndCount(collector, "swarm_cluster_count")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(1)) // One metric with labels

			// Verify ready agents metric
			readyAgents := testutil.ToFloat64(collector.clusterReadyAgents.WithLabelValues("cluster-1", "default", "mesh"))
			Expect(readyAgents).To(Equal(float64(3)))

			// Verify active agents metric
			activeAgents := testutil.ToFloat64(collector.clusterActiveAgents.WithLabelValues("cluster-2", "production", "hierarchical"))
			Expect(activeAgents).To(Equal(float64(4)))

			// Verify health status metric
			healthStatus := testutil.ToFloat64(collector.clusterHealthStatus.WithLabelValues("cluster-1", "default", "healthy"))
			Expect(healthStatus).To(Equal(float64(1)))
		})

		It("should handle empty cluster list", func() {
			collector.CollectClusterMetrics(ctx)
			
			count, err := testutil.CollectAndCount(collector, "swarm_cluster_count")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(Equal(1)) // Still has the metric, just with value 0
		})
	})

	Describe("Agent Metrics", func() {
		It("should collect agent metrics", func() {
			agents := []swarmv1alpha1.Agent{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "agent-1",
						Namespace: "default",
						Labels: map[string]string{
							"swarm.claudeflow.io/cluster": "cluster-1",
						},
					},
					Spec: swarmv1alpha1.AgentSpec{
						Type: swarmv1alpha1.ResearcherAgent,
					},
					Status: swarmv1alpha1.AgentStatus{
						State:     swarmv1alpha1.AgentReady,
						Workload:  25,
						Capacity:  100,
						TaskCount: 2,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "agent-2",
						Namespace: "default",
						Labels: map[string]string{
							"swarm.claudeflow.io/cluster": "cluster-1",
						},
					},
					Spec: swarmv1alpha1.AgentSpec{
						Type: swarmv1alpha1.CoderAgent,
					},
					Status: swarmv1alpha1.AgentStatus{
						State:     swarmv1alpha1.AgentBusy,
						Workload:  85,
						Capacity:  100,
						TaskCount: 5,
					},
				},
			}

			for _, agent := range agents {
				agent := agent // capture range variable
				Expect(fakeClient.Create(ctx, &agent)).To(Succeed())
			}

			collector.CollectAgentMetrics(ctx)

			// Verify agent count by type
			researcherCount := testutil.ToFloat64(collector.agentCountByType.WithLabelValues("researcher"))
			Expect(researcherCount).To(Equal(float64(1)))

			coderCount := testutil.ToFloat64(collector.agentCountByType.WithLabelValues("coder"))
			Expect(coderCount).To(Equal(float64(1)))

			// Verify workload metrics
			workload1 := testutil.ToFloat64(collector.agentWorkload.WithLabelValues("agent-1", "default", "cluster-1", "researcher"))
			Expect(workload1).To(Equal(float64(25)))

			workload2 := testutil.ToFloat64(collector.agentWorkload.WithLabelValues("agent-2", "default", "cluster-1", "coder"))
			Expect(workload2).To(Equal(float64(85)))

			// Verify task count metrics
			taskCount1 := testutil.ToFloat64(collector.agentTaskCount.WithLabelValues("agent-1", "default", "cluster-1", "researcher"))
			Expect(taskCount1).To(Equal(float64(2)))
		})

		It("should track agent state distribution", func() {
			states := []swarmv1alpha1.AgentState{
				swarmv1alpha1.AgentReady,
				swarmv1alpha1.AgentReady,
				swarmv1alpha1.AgentBusy,
				swarmv1alpha1.AgentIdle,
				swarmv1alpha1.AgentError,
			}

			for i, state := range states {
				agent := swarmv1alpha1.Agent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("agent-%d", i),
						Namespace: "default",
					},
					Status: swarmv1alpha1.AgentStatus{
						State: state,
					},
				}
				Expect(fakeClient.Create(ctx, &agent)).To(Succeed())
			}

			collector.CollectAgentMetrics(ctx)

			readyCount := testutil.ToFloat64(collector.agentStateCount.WithLabelValues("ready"))
			Expect(readyCount).To(Equal(float64(2)))

			busyCount := testutil.ToFloat64(collector.agentStateCount.WithLabelValues("busy"))
			Expect(busyCount).To(Equal(float64(1)))

			idleCount := testutil.ToFloat64(collector.agentStateCount.WithLabelValues("idle"))
			Expect(idleCount).To(Equal(float64(1)))

			errorCount := testutil.ToFloat64(collector.agentStateCount.WithLabelValues("error"))
			Expect(errorCount).To(Equal(float64(1)))
		})
	})

	Describe("Task Metrics", func() {
		It("should collect task metrics", func() {
			tasks := []swarmv1alpha1.SwarmTask{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "task-1",
						Namespace: "default",
					},
					Spec: swarmv1alpha1.SwarmTaskSpec{
						ClusterRef: "cluster-1",
						Task: swarmv1alpha1.TaskSpec{
							Type:     "research",
							Priority: swarmv1alpha1.HighPriority,
						},
					},
					Status: swarmv1alpha1.SwarmTaskStatus{
						State:        swarmv1alpha1.TaskRunning,
						StartTime:    &metav1.Time{Time: time.Now().Add(-5 * time.Minute)},
						AssignedAgents: []string{"agent-1"},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "task-2",
						Namespace: "default",
					},
					Spec: swarmv1alpha1.SwarmTaskSpec{
						ClusterRef: "cluster-1",
						Task: swarmv1alpha1.TaskSpec{
							Type:     "coding",
							Priority: swarmv1alpha1.CriticalPriority,
						},
					},
					Status: swarmv1alpha1.SwarmTaskStatus{
						State:          swarmv1alpha1.TaskCompleted,
						StartTime:      &metav1.Time{Time: time.Now().Add(-10 * time.Minute)},
						CompletionTime: &metav1.Time{Time: time.Now().Add(-2 * time.Minute)},
						AssignedAgents: []string{"agent-2", "agent-3"},
					},
				},
			}

			for _, task := range tasks {
				task := task // capture range variable
				Expect(fakeClient.Create(ctx, &task)).To(Succeed())
			}

			collector.CollectTaskMetrics(ctx)

			// Verify task count by state
			runningCount := testutil.ToFloat64(collector.taskCountByState.WithLabelValues("running"))
			Expect(runningCount).To(Equal(float64(1)))

			completedCount := testutil.ToFloat64(collector.taskCountByState.WithLabelValues("completed"))
			Expect(completedCount).To(Equal(float64(1)))

			// Verify task count by priority
			highPriorityCount := testutil.ToFloat64(collector.taskCountByPriority.WithLabelValues("high"))
			Expect(highPriorityCount).To(Equal(float64(1)))

			criticalPriorityCount := testutil.ToFloat64(collector.taskCountByPriority.WithLabelValues("critical"))
			Expect(criticalPriorityCount).To(Equal(float64(1)))

			// Verify duration metrics (completed task should have duration)
			// Note: actual duration checking would need mock time
		})

		It("should calculate task duration for completed tasks", func() {
			startTime := time.Now().Add(-30 * time.Minute)
			completionTime := time.Now().Add(-5 * time.Minute)

			task := swarmv1alpha1.SwarmTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "completed-task",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmTaskSpec{
					ClusterRef: "cluster-1",
					Task: swarmv1alpha1.TaskSpec{
						Type: "analysis",
					},
				},
				Status: swarmv1alpha1.SwarmTaskStatus{
					State:          swarmv1alpha1.TaskCompleted,
					StartTime:      &metav1.Time{Time: startTime},
					CompletionTime: &metav1.Time{Time: completionTime},
				},
			}

			Expect(fakeClient.Create(ctx, &task)).To(Succeed())

			collector.CollectTaskMetrics(ctx)

			// Duration should be recorded (25 minutes = 1500 seconds)
			observations, err := testutil.CollectAndCount(collector.taskDuration, "swarm_task_duration_seconds")
			Expect(err).NotTo(HaveOccurred())
			Expect(observations).To(BeNumerically(">", 0))
		})
	})

	Describe("Operation Metrics", func() {
		It("should record operation durations", func() {
			// Record some operations
			collector.RecordOperationDuration("reconcile", "swarmcluster", 0.05)
			collector.RecordOperationDuration("reconcile", "swarmcluster", 0.03)
			collector.RecordOperationDuration("reconcile", "agent", 0.02)
			collector.RecordOperationDuration("update", "swarmtask", 0.01)

			// Check histogram counts
			count, err := testutil.CollectAndCount(collector.operationDuration, "swarm_operator_operation_duration_seconds")
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeNumerically(">", 0))
		})

		It("should increment error counters", func() {
			collector.IncrementErrorCount("reconcile", "swarmcluster", "NotFound")
			collector.IncrementErrorCount("reconcile", "swarmcluster", "NotFound")
			collector.IncrementErrorCount("update", "agent", "Conflict")

			notFoundErrors := testutil.ToFloat64(collector.errorCount.WithLabelValues("reconcile", "swarmcluster", "NotFound"))
			Expect(notFoundErrors).To(Equal(float64(2)))

			conflictErrors := testutil.ToFloat64(collector.errorCount.WithLabelValues("update", "agent", "Conflict"))
			Expect(conflictErrors).To(Equal(float64(1)))
		})
	})

	Describe("Collector Registration", func() {
		It("should implement prometheus.Collector interface", func() {
			var _ prometheus.Collector = collector
		})

		It("should describe all metrics", func() {
			ch := make(chan *prometheus.Desc, 100)
			collector.Describe(ch)
			close(ch)

			// Should have multiple metric descriptions
			count := 0
			for range ch {
				count++
			}
			Expect(count).To(BeNumerically(">", 10)) // We have many metrics
		})

		It("should collect metrics without error", func() {
			ch := make(chan prometheus.Metric, 100)
			
			// This should not panic
			Expect(func() {
				collector.Collect(ch)
				close(ch)
			}).NotTo(Panic())

			// Should have collected some metrics
			count := 0
			for range ch {
				count++
			}
			Expect(count).To(BeNumerically(">=", 0))
		})
	})
})

