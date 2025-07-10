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

package topology

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
)

func TestTopologyManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Topology Manager Suite")
}

var _ = Describe("TopologyManager", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		manager    *Manager
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(swarmv1alpha1.AddToScheme(scheme)).To(Succeed())
		
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		
		manager = NewManager(fakeClient)
	})

	Describe("ValidateTopology", func() {
		It("should validate mesh topology", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     3,
				},
			}

			err := manager.ValidateTopology(cluster)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate hierarchical topology", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.HierarchicalTopology,
					Size:     5,
				},
			}

			err := manager.ValidateTopology(cluster)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate ring topology", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.RingTopology,
					Size:     4,
				},
			}

			err := manager.ValidateTopology(cluster)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate star topology", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.StarTopology,
					Size:     6,
				},
			}

			err := manager.ValidateTopology(cluster)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject invalid topology", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: "invalid",
					Size:     3,
				},
			}

			err := manager.ValidateTopology(cluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported topology"))
		})

		It("should reject invalid size for topology", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.StarTopology,
					Size:     0,
				},
			}

			err := manager.ValidateTopology(cluster)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("CalculateAgentConnections", func() {
		Context("Mesh topology", func() {
			It("should calculate full mesh connections", func() {
				cluster := &swarmv1alpha1.SwarmCluster{
					Spec: swarmv1alpha1.SwarmClusterSpec{
						Topology: swarmv1alpha1.MeshTopology,
						Size:     3,
					},
				}

				agents := []swarmv1alpha1.Agent{
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-0"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-2"}},
				}

				connections, err := manager.CalculateAgentConnections(cluster, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// In mesh topology, each agent connects to all others
				Expect(connections).To(HaveLen(3))
				Expect(connections["agent-0"]).To(ConsistOf("agent-1", "agent-2"))
				Expect(connections["agent-1"]).To(ConsistOf("agent-0", "agent-2"))
				Expect(connections["agent-2"]).To(ConsistOf("agent-0", "agent-1"))
			})
		})

		Context("Hierarchical topology", func() {
			It("should calculate hierarchical connections", func() {
				cluster := &swarmv1alpha1.SwarmCluster{
					Spec: swarmv1alpha1.SwarmClusterSpec{
						Topology: swarmv1alpha1.HierarchicalTopology,
						Size:     7,
					},
				}

				agents := []swarmv1alpha1.Agent{
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-0"}}, // root
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-1"}}, // level 1
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-2"}}, // level 1
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-3"}}, // level 2
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-4"}}, // level 2
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-5"}}, // level 2
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-6"}}, // level 2
				}

				connections, err := manager.CalculateAgentConnections(cluster, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Root connects to level 1
				Expect(connections["agent-0"]).To(ConsistOf("agent-1", "agent-2"))
				// Level 1 connects to root and their children
				Expect(connections["agent-1"]).To(ContainElement("agent-0"))
				Expect(connections["agent-2"]).To(ContainElement("agent-0"))
			})
		})

		Context("Ring topology", func() {
			It("should calculate ring connections", func() {
				cluster := &swarmv1alpha1.SwarmCluster{
					Spec: swarmv1alpha1.SwarmClusterSpec{
						Topology: swarmv1alpha1.RingTopology,
						Size:     4,
					},
				}

				agents := []swarmv1alpha1.Agent{
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-0"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-2"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-3"}},
				}

				connections, err := manager.CalculateAgentConnections(cluster, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Ring topology: each agent connects to next and previous
				Expect(connections["agent-0"]).To(ConsistOf("agent-3", "agent-1"))
				Expect(connections["agent-1"]).To(ConsistOf("agent-0", "agent-2"))
				Expect(connections["agent-2"]).To(ConsistOf("agent-1", "agent-3"))
				Expect(connections["agent-3"]).To(ConsistOf("agent-2", "agent-0"))
			})
		})

		Context("Star topology", func() {
			It("should calculate star connections", func() {
				cluster := &swarmv1alpha1.SwarmCluster{
					Spec: swarmv1alpha1.SwarmClusterSpec{
						Topology: swarmv1alpha1.StarTopology,
						Size:     5,
					},
				}

				agents := []swarmv1alpha1.Agent{
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-0"}}, // central
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-2"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-3"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "agent-4"}},
				}

				connections, err := manager.CalculateAgentConnections(cluster, agents)
				Expect(err).NotTo(HaveOccurred())
				
				// Star topology: central connects to all, others only to central
				Expect(connections["agent-0"]).To(ConsistOf("agent-1", "agent-2", "agent-3", "agent-4"))
				Expect(connections["agent-1"]).To(ConsistOf("agent-0"))
				Expect(connections["agent-2"]).To(ConsistOf("agent-0"))
				Expect(connections["agent-3"]).To(ConsistOf("agent-0"))
				Expect(connections["agent-4"]).To(ConsistOf("agent-0"))
			})
		})
	})

	Describe("OptimizeTopology", func() {
		It("should maintain topology during optimization", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     3,
				},
			}

			// Create agents in the fake client
			for i := 0; i < 3; i++ {
				agent := &swarmv1alpha1.Agent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cluster.Name + "-agent-" + string(rune('0'+i)),
						Namespace: cluster.Namespace,
						Labels: map[string]string{
							"swarm.claudeflow.io/cluster": cluster.Name,
						},
					},
				}
				Expect(fakeClient.Create(ctx, agent)).To(Succeed())
			}

			err := manager.OptimizeTopology(ctx, cluster)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify cluster status is updated
			Expect(cluster.Status.Topology).To(Equal(swarmv1alpha1.MeshTopology))
		})

		It("should handle missing agents gracefully", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     3,
				},
			}

			// No agents created
			err := manager.OptimizeTopology(ctx, cluster)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Edge Cases", func() {
		It("should handle nil cluster", func() {
			err := manager.ValidateTopology(nil)
			Expect(err).To(HaveOccurred())
		})

		It("should handle empty agent list", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     3,
				},
			}

			connections, err := manager.CalculateAgentConnections(cluster, []swarmv1alpha1.Agent{})
			Expect(err).NotTo(HaveOccurred())
			Expect(connections).To(BeEmpty())
		})

		It("should handle single agent", func() {
			cluster := &swarmv1alpha1.SwarmCluster{
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.MeshTopology,
					Size:     1,
				},
			}

			agents := []swarmv1alpha1.Agent{
				{ObjectMeta: metav1.ObjectMeta{Name: "agent-0"}},
			}

			connections, err := manager.CalculateAgentConnections(cluster, agents)
			Expect(err).NotTo(HaveOccurred())
			Expect(connections).To(HaveLen(1))
			Expect(connections["agent-0"]).To(BeEmpty())
		})
	})
})

// Mock client for testing
type MockClient struct {
	mock.Mock
	client.Client
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}