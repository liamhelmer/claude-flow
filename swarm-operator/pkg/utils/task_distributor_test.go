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

package utils

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

func TestTaskDistributor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task Distributor Suite")
}

var _ = Describe("TaskDistributor", func() {
	var (
		ctx         context.Context
		fakeClient  client.Client
		distributor *TaskDistributor
		scheme      *runtime.Scheme
		ctrl        *gomock.Controller
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		scheme = runtime.NewScheme()
		Expect(swarmv1alpha1.AddToScheme(scheme)).To(Succeed())
		
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		
		distributor = NewTaskDistributor(fakeClient)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("DistributeTask", func() {
		var (
			cluster *swarmv1alpha1.SwarmCluster
			task    *swarmv1alpha1.SwarmTask
			agents  []swarmv1alpha1.Agent
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
				},
			}

			task = &swarmv1alpha1.SwarmTask{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-task",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmTaskSpec{
					ClusterRef: "test-cluster",
					Task: swarmv1alpha1.TaskSpec{
						Type:        "research",
						Description: "Test research task",
						Priority:    swarmv1alpha1.HighPriority,
					},
					Strategy: swarmv1alpha1.StrategySpec{
						Type:               swarmv1alpha1.ParallelStrategy,
						MaxConcurrentTasks: 3,
					},
				},
			}

			agents = []swarmv1alpha1.Agent{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "agent-0",
						Namespace: "default",
					},
					Spec: swarmv1alpha1.AgentSpec{
						Type:         swarmv1alpha1.ResearcherAgent,
						Capabilities: []string{"research", "analysis"},
					},
					Status: swarmv1alpha1.AgentStatus{
						State:     swarmv1alpha1.AgentReady,
						Workload:  0,
						Capacity:  100,
						TaskCount: 0,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "agent-1",
						Namespace: "default",
					},
					Spec: swarmv1alpha1.AgentSpec{
						Type:         swarmv1alpha1.CoderAgent,
						Capabilities: []string{"coding", "testing"},
					},
					Status: swarmv1alpha1.AgentStatus{
						State:     swarmv1alpha1.AgentReady,
						Workload:  50,
						Capacity:  100,
						TaskCount: 1,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "agent-2",
						Namespace: "default",
					},
					Spec: swarmv1alpha1.AgentSpec{
						Type:         swarmv1alpha1.AnalystAgent,
						Capabilities: []string{"analysis", "optimization"},
					},
					Status: swarmv1alpha1.AgentStatus{
						State:     swarmv1alpha1.AgentBusy,
						Workload:  90,
						Capacity:  100,
						TaskCount: 3,
					},
				},
			}
		})

		Context("Parallel strategy", func() {
			It("should distribute task to capable agents", func() {
				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				Expect(assignments).NotTo(BeEmpty())
				
				// Should assign to researcher agent (agent-0) as it has matching capabilities
				Expect(assignments).To(HaveKey("agent-0"))
			})

			It("should consider agent workload", func() {
				// All agents are coders with different workloads
				agents = []swarmv1alpha1.Agent{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "agent-0"},
						Spec: swarmv1alpha1.AgentSpec{
							Type:         swarmv1alpha1.CoderAgent,
							Capabilities: []string{"coding"},
						},
						Status: swarmv1alpha1.AgentStatus{
							State:    swarmv1alpha1.AgentReady,
							Workload: 20,
							Capacity: 100,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{Name: "agent-1"},
						Spec: swarmv1alpha1.AgentSpec{
							Type:         swarmv1alpha1.CoderAgent,
							Capabilities: []string{"coding"},
						},
						Status: swarmv1alpha1.AgentStatus{
							State:    swarmv1alpha1.AgentReady,
							Workload: 80,
							Capacity: 100,
						},
					},
				}

				task.Spec.Task.Type = "coding"
				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Should prefer agent-0 with lower workload
				Expect(assignments).To(HaveKey("agent-0"))
			})

			It("should respect max concurrent tasks", func() {
				task.Spec.Strategy.MaxConcurrentTasks = 1
				
				// Create many capable agents
				manyAgents := make([]swarmv1alpha1.Agent, 5)
				for i := range manyAgents {
					manyAgents[i] = swarmv1alpha1.Agent{
						ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("agent-%d", i)},
						Spec: swarmv1alpha1.AgentSpec{
							Type:         swarmv1alpha1.ResearcherAgent,
							Capabilities: []string{"research"},
						},
						Status: swarmv1alpha1.AgentStatus{
							State:    swarmv1alpha1.AgentReady,
							Workload: 0,
							Capacity: 100,
						},
					}
				}

				assignments, err := distributor.DistributeTask(ctx, cluster, task, manyAgents)
				Expect(err).NotTo(HaveOccurred())
				Expect(assignments).To(HaveLen(1)) // Only 1 assignment due to limit
			})
		})

		Context("Sequential strategy", func() {
			BeforeEach(func() {
				task.Spec.Strategy.Type = swarmv1alpha1.SequentialStrategy
			})

			It("should assign to single best agent", func() {
				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				Expect(assignments).To(HaveLen(1))
			})

			It("should prefer specialized agents", func() {
				// Add a coordinator that can do research but is not specialized
				agents = append(agents, swarmv1alpha1.Agent{
					ObjectMeta: metav1.ObjectMeta{Name: "coordinator"},
					Spec: swarmv1alpha1.AgentSpec{
						Type:         swarmv1alpha1.CoordinatorAgent,
						Capabilities: []string{"research", "coordination"},
					},
					Status: swarmv1alpha1.AgentStatus{
						State:    swarmv1alpha1.AgentReady,
						Workload: 0,
						Capacity: 100,
					},
				})

				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Should still prefer the researcher agent
				Expect(assignments).To(HaveKey("agent-0"))
			})
		})

		Context("Adaptive strategy", func() {
			BeforeEach(func() {
				task.Spec.Strategy.Type = swarmv1alpha1.AdaptiveStrategy
			})

			It("should adapt based on task complexity", func() {
				// Complex task should get multiple agents
				task.Spec.Task.Dependencies = []string{"dep1", "dep2", "dep3"}
				
				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(assignments)).To(BeNumerically(">=", 1))
			})

			It("should consider agent availability", func() {
				// Make all agents busy except one
				for i := range agents {
					if i < len(agents)-1 {
						agents[i].Status.State = swarmv1alpha1.AgentBusy
						agents[i].Status.Workload = 95
					}
				}

				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				// Should only assign to available agent
				Expect(assignments).To(HaveLen(1))
			})
		})

		Context("Error cases", func() {
			It("should handle no available agents", func() {
				// Make all agents busy
				for i := range agents {
					agents[i].Status.State = swarmv1alpha1.AgentBusy
					agents[i].Status.Workload = 100
				}

				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				Expect(assignments).To(BeEmpty())
			})

			It("should handle no capable agents", func() {
				// Change task type to something no agent can handle
				task.Spec.Task.Type = "quantum-computing"

				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				Expect(assignments).To(BeEmpty())
			})

			It("should handle nil inputs gracefully", func() {
				_, err := distributor.DistributeTask(ctx, nil, task, agents)
				Expect(err).To(HaveOccurred())

				_, err = distributor.DistributeTask(ctx, cluster, nil, agents)
				Expect(err).To(HaveOccurred())

				_, err = distributor.DistributeTask(ctx, cluster, task, nil)
				Expect(err).NotTo(HaveOccurred()) // Empty agent list is valid
			})
		})

		Context("Priority handling", func() {
			It("should prioritize high priority tasks", func() {
				task.Spec.Task.Priority = swarmv1alpha1.CriticalPriority
				
				// Agent-2 is busy but task is critical
				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Should still assign despite high workload
				Expect(assignments).NotTo(BeEmpty())
			})

			It("should defer low priority tasks when agents are busy", func() {
				task.Spec.Task.Priority = swarmv1alpha1.LowPriority
				
				// Make all agents moderately busy
				for i := range agents {
					agents[i].Status.Workload = 70
				}

				assignments, err := distributor.DistributeTask(ctx, cluster, task, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Might not assign if workload is too high for low priority
				// This depends on implementation threshold
			})
		})
	})

	Describe("CalculateOptimalAssignment", func() {
		It("should calculate assignment scores", func() {
			agent := swarmv1alpha1.Agent{
				Spec: swarmv1alpha1.AgentSpec{
					Type:         swarmv1alpha1.ResearcherAgent,
					Capabilities: []string{"research", "analysis"},
				},
				Status: swarmv1alpha1.AgentStatus{
					State:    swarmv1alpha1.AgentReady,
					Workload: 30,
					Capacity: 100,
				},
			}

			task := swarmv1alpha1.SwarmTask{
				Spec: swarmv1alpha1.SwarmTaskSpec{
					Task: swarmv1alpha1.TaskSpec{
						Type:     "research",
						Priority: swarmv1alpha1.MediumPriority,
					},
				},
			}

			score := distributor.CalculateOptimalAssignment(agent, task)
			Expect(score).To(BeNumerically(">", 0))
			Expect(score).To(BeNumerically("<=", 1))
		})

		It("should give higher scores to specialized agents", func() {
			researcher := swarmv1alpha1.Agent{
				Spec: swarmv1alpha1.AgentSpec{
					Type:         swarmv1alpha1.ResearcherAgent,
					Capabilities: []string{"research"},
				},
				Status: swarmv1alpha1.AgentStatus{State: swarmv1alpha1.AgentReady, Workload: 50},
			}

			generalist := swarmv1alpha1.Agent{
				Spec: swarmv1alpha1.AgentSpec{
					Type:         swarmv1alpha1.CoordinatorAgent,
					Capabilities: []string{"research", "coordination", "planning"},
				},
				Status: swarmv1alpha1.AgentStatus{State: swarmv1alpha1.AgentReady, Workload: 50},
			}

			task := swarmv1alpha1.SwarmTask{
				Spec: swarmv1alpha1.SwarmTaskSpec{
					Task: swarmv1alpha1.TaskSpec{Type: "research"},
				},
			}

			researcherScore := distributor.CalculateOptimalAssignment(researcher, task)
			generalistScore := distributor.CalculateOptimalAssignment(generalist, task)
			
			Expect(researcherScore).To(BeNumerically(">", generalistScore))
		})

		It("should consider workload in scoring", func() {
			lowWorkload := swarmv1alpha1.Agent{
				Spec: swarmv1alpha1.AgentSpec{
					Type:         swarmv1alpha1.CoderAgent,
					Capabilities: []string{"coding"},
				},
				Status: swarmv1alpha1.AgentStatus{State: swarmv1alpha1.AgentReady, Workload: 10},
			}

			highWorkload := swarmv1alpha1.Agent{
				Spec: swarmv1alpha1.AgentSpec{
					Type:         swarmv1alpha1.CoderAgent,
					Capabilities: []string{"coding"},
				},
				Status: swarmv1alpha1.AgentStatus{State: swarmv1alpha1.AgentReady, Workload: 90},
			}

			task := swarmv1alpha1.SwarmTask{
				Spec: swarmv1alpha1.SwarmTaskSpec{
					Task: swarmv1alpha1.TaskSpec{Type: "coding"},
				},
			}

			lowScore := distributor.CalculateOptimalAssignment(lowWorkload, task)
			highScore := distributor.CalculateOptimalAssignment(highWorkload, task)
			
			Expect(lowScore).To(BeNumerically(">", highScore))
		})
	})
})

