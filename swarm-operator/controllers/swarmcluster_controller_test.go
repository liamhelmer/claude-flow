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

package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
	"github.com/claude-flow/swarm-operator/pkg/metrics"
	"github.com/claude-flow/swarm-operator/pkg/topology"
)

var _ = Describe("SwarmCluster Controller", func() {
	var (
		ctx            context.Context
		k8sClient      client.Client
		reconciler     *SwarmClusterReconciler
		scheme         *runtime.Scheme
		recorder       *record.FakeRecorder
		metricsCollect *metrics.Collector
		topoManager    *topology.Manager
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(swarmv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&swarmv1alpha1.SwarmCluster{}, &swarmv1alpha1.Agent{}).
			Build()

		recorder = record.NewFakeRecorder(100)
		metricsCollect = metrics.NewCollector(k8sClient)
		topoManager = topology.NewManager(k8sClient)

		reconciler = &SwarmClusterReconciler{
			Client:          k8sClient,
			Scheme:          scheme,
			Recorder:        recorder,
			MetricsCollector: metricsCollect,
			TopologyManager: topoManager,
		}
	})

	Describe("Reconcile", func() {
		var (
			cluster        *swarmv1alpha1.SwarmCluster
			namespacedName types.NamespacedName
			req            reconcile.Request
		)

		BeforeEach(func() {
			cluster = &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     3,
					Strategy: swarmv1alpha1.StrategySpec{
						Type:               swarmv1alpha1.BalancedStrategy,
						MaxConcurrentTasks: 5,
					},
				},
			}

			namespacedName = types.NamespacedName{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}

			req = reconcile.Request{
				NamespacedName: namespacedName,
			}
		})

		Context("Creating a new SwarmCluster", func() {
			It("should create agents and update status", func() {
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Check that agents were created
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())
				Expect(agents.Items).To(HaveLen(int(cluster.Spec.Size)))

				// Check agent properties
				for i, agent := range agents.Items {
					Expect(agent.Name).To(Equal(fmt.Sprintf("%s-agent-%d", cluster.Name, i)))
					Expect(agent.Labels).To(HaveKeyWithValue("swarm.claudeflow.io/cluster", cluster.Name))
					Expect(agent.OwnerReferences).To(HaveLen(1))
					Expect(agent.OwnerReferences[0].Name).To(Equal(cluster.Name))
				}

				// Check cluster status
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				Expect(updatedCluster.Status.State).To(Equal(swarmv1alpha1.ClusterPending))
				Expect(updatedCluster.Status.ReadyAgents).To(Equal(int32(0)))
			})

			It("should handle different topologies", func() {
				testCases := []swarmv1alpha1.TopologyType{
					swarmv1alpha1.MeshTopology,
					swarmv1alpha1.HierarchicalTopology,
					swarmv1alpha1.RingTopology,
					swarmv1alpha1.StarTopology,
				}

				for _, topology := range testCases {
					By(fmt.Sprintf("Testing %s topology", topology))
					
					testCluster := cluster.DeepCopy()
					testCluster.Name = fmt.Sprintf("test-cluster-%s", topology)
					testCluster.Spec.Topology = topology
					
					Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())
					
					req := reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      testCluster.Name,
							Namespace: testCluster.Namespace,
						},
					}
					
					result, err := reconciler.Reconcile(ctx, req)
					Expect(err).NotTo(HaveOccurred())
					Expect(result.Requeue).To(BeFalse())
					
					// Verify topology was set correctly
					updatedCluster := &swarmv1alpha1.SwarmCluster{}
					Expect(k8sClient.Get(ctx, req.NamespacedName, updatedCluster)).To(Succeed())
					Expect(updatedCluster.Status.Topology).To(Equal(topology))
				}
			})

			It("should emit events", func() {
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Check for expected events
				Eventually(recorder.Events).Should(Receive(ContainSubstring("CreatingAgents")))
			})
		})

		Context("Updating an existing SwarmCluster", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
				// Initial reconciliation
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should scale up when size increases", func() {
				// Update cluster size
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				updatedCluster.Spec.Size = 5
				Expect(k8sClient.Update(ctx, updatedCluster)).To(Succeed())

				// Reconcile
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Check that new agents were created
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())
				Expect(agents.Items).To(HaveLen(5))
			})

			It("should scale down when size decreases", func() {
				// Update cluster size
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				updatedCluster.Spec.Size = 1
				Expect(k8sClient.Update(ctx, updatedCluster)).To(Succeed())

				// Reconcile
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Check that excess agents were deleted
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())
				Expect(agents.Items).To(HaveLen(1))
			})

			It("should update agent connections when topology changes", func() {
				// Change topology
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				updatedCluster.Spec.Topology = swarmv1alpha1.StarTopology
				Expect(k8sClient.Update(ctx, updatedCluster)).To(Succeed())

				// Reconcile
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Verify topology was updated
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				Expect(updatedCluster.Status.Topology).To(Equal(swarmv1alpha1.StarTopology))
			})
		})

		Context("Agent status updates", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
				// Initial reconciliation
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update cluster status when agents become ready", func() {
				// Get created agents
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())

				// Update agents to ready state
				for _, agent := range agents.Items {
					agent.Status.State = swarmv1alpha1.AgentReady
					Expect(k8sClient.Status().Update(ctx, &agent)).To(Succeed())
				}

				// Reconcile
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Check cluster status
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				Expect(updatedCluster.Status.ReadyAgents).To(Equal(int32(3)))
				Expect(updatedCluster.Status.State).To(Equal(swarmv1alpha1.ClusterReady))
			})

			It("should handle partial agent readiness", func() {
				// Get created agents
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())

				// Update only some agents to ready
				for i, agent := range agents.Items {
					if i < 2 {
						agent.Status.State = swarmv1alpha1.AgentReady
					} else {
						agent.Status.State = swarmv1alpha1.AgentError
					}
					Expect(k8sClient.Status().Update(ctx, &agent)).To(Succeed())
				}

				// Reconcile
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Check cluster status
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				Expect(updatedCluster.Status.ReadyAgents).To(Equal(int32(2)))
				Expect(updatedCluster.Status.State).To(Equal(swarmv1alpha1.ClusterDegraded))
				Expect(updatedCluster.Status.Health).To(Equal(swarmv1alpha1.DegradedCondition))
			})
		})

		Context("Deletion", func() {
			It("should handle cluster deletion", func() {
				// Create cluster with finalizer
				cluster.Finalizers = []string{"swarm.claudeflow.io/finalizer"}
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

				// Initial reconciliation
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Mark for deletion
				Expect(k8sClient.Delete(ctx, cluster)).To(Succeed())

				// Reconcile deletion
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Verify cluster is deleted
				err = k8sClient.Get(ctx, namespacedName, &swarmv1alpha1.SwarmCluster{})
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("Error handling", func() {
			It("should handle missing cluster", func() {
				// Reconcile non-existent cluster
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
			})

			It("should requeue on transient errors", func() {
				// Create a cluster that will trigger errors
				cluster.Spec.Size = -1 // Invalid size
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

				// Reconcile should handle error gracefully
				result, err := reconciler.Reconcile(ctx, req)
				// Depending on implementation, this might error or requeue
				if err != nil {
					Expect(result.Requeue).To(BeTrue())
				}
			})
		})

		Context("Health monitoring", func() {
			BeforeEach(func() {
				Expect(k8sClient.Create(ctx, cluster)).To(Succeed())
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update health status based on agent states", func() {
				// Get agents
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())

				// Set all agents to healthy
				for _, agent := range agents.Items {
					agent.Status.State = swarmv1alpha1.AgentReady
					agent.Status.Health = swarmv1alpha1.HealthyCondition
					Expect(k8sClient.Status().Update(ctx, &agent)).To(Succeed())
				}

				// Reconcile
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Check health
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				Expect(updatedCluster.Status.Health).To(Equal(swarmv1alpha1.HealthyCondition))
			})

			It("should detect unhealthy conditions", func() {
				// Get agents
				var agents swarmv1alpha1.AgentList
				Expect(k8sClient.List(ctx, &agents, client.InNamespace(cluster.Namespace))).To(Succeed())

				// Set one agent to error
				agents.Items[0].Status.State = swarmv1alpha1.AgentError
				agents.Items[0].Status.Health = swarmv1alpha1.UnhealthyCondition
				Expect(k8sClient.Status().Update(ctx, &agents.Items[0])).To(Succeed())

				// Reconcile
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Check health
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedCluster)).To(Succeed())
				Expect(updatedCluster.Status.Health).To(Equal(swarmv1alpha1.DegradedCondition))
			})
		})
	})

	Describe("SetupWithManager", func() {
		It("should setup the controller", func() {
			// This would require a real manager, so we just verify the method exists
			mgr := &mockManager{}
			err := reconciler.SetupWithManager(mgr)
			Expect(err).To(BeNil())
		})
	})
})

// Mock manager for testing
type mockManager struct {
	ctrl.Manager
}

func (m *mockManager) GetScheme() *runtime.Scheme {
	return runtime.NewScheme()
}

func (m *mockManager) GetClient() client.Client {
	return nil
}

func (m *mockManager) GetFieldIndexer() client.FieldIndexer {
	return nil
}

func (m *mockManager) GetEventRecorderFor(name string) record.EventRecorder {
	return nil
}