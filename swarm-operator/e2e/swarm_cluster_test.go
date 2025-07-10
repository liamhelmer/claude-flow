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

package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

var _ = Describe("SwarmCluster E2E Tests", func() {
	var (
		namespace string
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = CreateNamespace(ctx)
	})

	AfterEach(func() {
		DeleteNamespace(ctx, namespace)
	})

	Context("Basic SwarmCluster Operations", func() {
		It("should create a functional mesh topology swarm", func() {
			By("Creating a SwarmCluster with mesh topology")
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mesh-cluster",
					Namespace: namespace,
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
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Waiting for cluster to become ready")
			WaitForClusterReady(ctx, cluster.Name, namespace, 2*time.Minute)

			By("Verifying all agents are created and ready")
			agents := GetClusterAgents(ctx, cluster.Name, namespace)
			Expect(agents).To(HaveLen(3))
			
			for _, agent := range agents {
				WaitForAgentReady(ctx, agent.Name, namespace, time.Minute)
			}

			By("Verifying agent connections in mesh topology")
			// In mesh topology, each agent should connect to all others
			for _, agent := range agents {
				updatedAgent := &swarmv1alpha1.Agent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      agent.Name,
					Namespace: namespace,
				}, updatedAgent)).To(Succeed())
				
				// Each agent should have connections to other agents
				Expect(updatedAgent.Status.Connections).To(HaveLen(2))
			}

			By("Verifying cluster metrics")
			updatedCluster := &swarmv1alpha1.SwarmCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: namespace,
			}, updatedCluster)).To(Succeed())
			
			Expect(updatedCluster.Status.ReadyAgents).To(Equal(int32(3)))
			Expect(updatedCluster.Status.ActiveAgents).To(BeNumerically(">=", int32(0)))
			Expect(updatedCluster.Status.Health).To(Equal(swarmv1alpha1.HealthyCondition))
		})

		It("should create and scale a hierarchical topology swarm", func() {
			By("Creating a SwarmCluster with hierarchical topology")
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hierarchical-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.HierarchicalTopology,
					Size:     7,
					Strategy: swarmv1alpha1.StrategySpec{
						Type:               swarmv1alpha1.AdaptiveStrategy,
						MaxConcurrentTasks: 10,
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Waiting for initial cluster to become ready")
			WaitForClusterReady(ctx, cluster.Name, namespace, 2*time.Minute)

			By("Verifying hierarchical structure")
			agents := GetClusterAgents(ctx, cluster.Name, namespace)
			Expect(agents).To(HaveLen(7))

			// Verify root agent exists
			rootAgent := &swarmv1alpha1.Agent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-agent-0", cluster.Name),
				Namespace: namespace,
			}, rootAgent)).To(Succeed())

			By("Scaling down the cluster")
			updatedCluster := &swarmv1alpha1.SwarmCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: namespace,
			}, updatedCluster)).To(Succeed())
			
			updatedCluster.Spec.Size = 3
			Expect(k8sClient.Update(ctx, updatedCluster)).To(Succeed())

			By("Waiting for cluster to scale down")
			Eventually(func() int {
				agents := GetClusterAgents(ctx, cluster.Name, namespace)
				return len(agents)
			}, 2*time.Minute, 5*time.Second).Should(Equal(3))

			By("Verifying cluster remains healthy after scaling")
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: namespace,
			}, updatedCluster)).To(Succeed())
			Expect(updatedCluster.Status.ReadyAgents).To(Equal(int32(3)))
		})
	})

	Context("Failure Recovery", func() {
		It("should recover from agent failures", func() {
			By("Creating a SwarmCluster")
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "recovery-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.StarTopology,
					Size:     5,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Waiting for cluster to become ready")
			WaitForClusterReady(ctx, cluster.Name, namespace, 2*time.Minute)

			By("Simulating agent failure")
			agents := GetClusterAgents(ctx, cluster.Name, namespace)
			Expect(agents).To(HaveLen(5))

			// Delete one non-central agent
			agentToDelete := &agents[2]
			Expect(k8sClient.Delete(ctx, agentToDelete)).To(Succeed())

			By("Waiting for cluster to detect and recover")
			Eventually(func() int {
				agents := GetClusterAgents(ctx, cluster.Name, namespace)
				return len(agents)
			}, 2*time.Minute, 5*time.Second).Should(Equal(5))

			By("Verifying cluster health after recovery")
			updatedCluster := &swarmv1alpha1.SwarmCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: namespace,
			}, updatedCluster)).To(Succeed())
			
			Eventually(func() swarmv1alpha1.HealthCondition {
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      cluster.Name,
					Namespace: namespace,
				}, updatedCluster)).To(Succeed())
				return updatedCluster.Status.Health
			}, time.Minute, 5*time.Second).Should(Equal(swarmv1alpha1.HealthyCondition))
		})

		It("should handle central node failure in star topology", func() {
			By("Creating a star topology cluster")
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "star-recovery-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.StarTopology,
					Size:     5,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Waiting for cluster to become ready")
			WaitForClusterReady(ctx, cluster.Name, namespace, 2*time.Minute)

			By("Simulating central agent failure")
			centralAgent := &swarmv1alpha1.Agent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      fmt.Sprintf("%s-agent-0", cluster.Name),
				Namespace: namespace,
			}, centralAgent)).To(Succeed())

			centralAgent.Status.State = swarmv1alpha1.AgentError
			centralAgent.Status.Health = swarmv1alpha1.UnhealthyCondition
			Expect(k8sClient.Status().Update(ctx, centralAgent)).To(Succeed())

			By("Verifying cluster detects degraded state")
			Eventually(func() swarmv1alpha1.ClusterState {
				updatedCluster := &swarmv1alpha1.SwarmCluster{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      cluster.Name,
					Namespace: namespace,
				}, updatedCluster)).To(Succeed())
				return updatedCluster.Status.State
			}, time.Minute, 5*time.Second).Should(Equal(swarmv1alpha1.ClusterDegraded))
		})
	})

	Context("Task Distribution", func() {
		It("should distribute tasks across agents", func() {
			By("Creating a SwarmCluster")
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "task-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     3,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Waiting for cluster to become ready")
			WaitForClusterReady(ctx, cluster.Name, namespace, 2*time.Minute)

			By("Creating a task for the cluster")
			task := &swarmv1alpha1.SwarmTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-task",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmTaskSpec{
					ClusterRef: cluster.Name,
					Task: swarmv1alpha1.TaskSpec{
						Type:        "research",
						Description: "Test research task",
						Priority:    swarmv1alpha1.HighPriority,
					},
					Strategy: swarmv1alpha1.StrategySpec{
						Type:               swarmv1alpha1.ParallelStrategy,
						MaxConcurrentTasks: 2,
					},
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("Verifying task is assigned to agents")
			Eventually(func() int {
				updatedTask := &swarmv1alpha1.SwarmTask{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      task.Name,
					Namespace: namespace,
				}, updatedTask)
				if err != nil {
					return 0
				}
				return len(updatedTask.Status.AssignedAgents)
			}, time.Minute, 5*time.Second).Should(BeNumerically(">=", 1))

			By("Verifying agent workload is updated")
			agents := GetClusterAgents(ctx, cluster.Name, namespace)
			assignedCount := 0
			for _, agent := range agents {
				updatedAgent := &swarmv1alpha1.Agent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      agent.Name,
					Namespace: namespace,
				}, updatedAgent)).To(Succeed())
				
				if updatedAgent.Status.TaskCount > 0 {
					assignedCount++
					Expect(updatedAgent.Status.Workload).To(BeNumerically(">", 0))
				}
			}
			Expect(assignedCount).To(BeNumerically(">=", 1))
		})
	})

	Context("Performance and Scaling", func() {
		It("should handle large cluster sizes efficiently", func() {
			By("Creating a large SwarmCluster")
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "large-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.HierarchicalTopology,
					Size:     20,
					Strategy: swarmv1alpha1.StrategySpec{
						Type:               swarmv1alpha1.BalancedStrategy,
						MaxConcurrentTasks: 50,
					},
				},
			}
			
			startTime := time.Now()
			Expect(k8sClient.Create(ctx, cluster)).To(Succeed())

			By("Measuring time to reach ready state")
			WaitForClusterReady(ctx, cluster.Name, namespace, 5*time.Minute)
			readyTime := time.Since(startTime)

			By("Verifying all agents are created")
			agents := GetClusterAgents(ctx, cluster.Name, namespace)
			Expect(agents).To(HaveLen(20))

			By("Checking performance metrics")
			Expect(readyTime).To(BeNumerically("<", 5*time.Minute))
			
			// Verify cluster can handle concurrent tasks
			By("Creating multiple concurrent tasks")
			for i := 0; i < 5; i++ {
				task := &swarmv1alpha1.SwarmTask{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("perf-task-%d", i),
						Namespace: namespace,
					},
					Spec: swarmv1alpha1.SwarmTaskSpec{
						ClusterRef: cluster.Name,
						Task: swarmv1alpha1.TaskSpec{
							Type:        "analysis",
							Description: fmt.Sprintf("Performance test task %d", i),
							Priority:    swarmv1alpha1.MediumPriority,
						},
						Strategy: swarmv1alpha1.StrategySpec{
							Type: swarmv1alpha1.ParallelStrategy,
						},
					},
				}
				Expect(k8sClient.Create(ctx, task)).To(Succeed())
			}

			By("Verifying tasks are distributed efficiently")
			time.Sleep(30 * time.Second) // Allow time for distribution
			
			// Check workload distribution
			var totalWorkload int32
			var activeAgents int
			for _, agent := range agents {
				updatedAgent := &swarmv1alpha1.Agent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      agent.Name,
					Namespace: namespace,
				}, updatedAgent)).To(Succeed())
				
				if updatedAgent.Status.Workload > 0 {
					activeAgents++
					totalWorkload += updatedAgent.Status.Workload
				}
			}
			
			// Verify reasonable distribution
			Expect(activeAgents).To(BeNumerically(">=", 3))
			averageWorkload := totalWorkload / int32(activeAgents)
			Expect(averageWorkload).To(BeNumerically("<", 80)) // No agent should be overloaded
		})
	})
})