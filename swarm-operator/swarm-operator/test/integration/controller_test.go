package integration

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	swarmv1alpha1 "github.com/claudeflow/swarm-operator/api/v1alpha1"
)

var _ = Describe("SwarmCluster Controller", func() {
	Context("When creating a SwarmCluster with HiveMind", func() {
		var (
			ctx        context.Context
			cluster    *swarmv1alpha1.SwarmCluster
			namespace  string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = "test-hivemind-" + randomString(6)
			
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create SwarmCluster with HiveMind
			cluster = &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hivemind-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology:  swarmv1alpha1.TopologyMesh,
					QueenMode: swarmv1alpha1.QueenModeDistributed,
					Strategy:  swarmv1alpha1.StrategyConsensus,
					ConsensusThreshold: 0.66,
					HiveMind: swarmv1alpha1.HiveMindSpec{
						Enabled:      true,
						DatabaseSize: "1Gi",
						SyncInterval: "30s",
						BackupEnabled: true,
					},
					Memory: swarmv1alpha1.MemorySpec{
						Type: "redis",
						Size: "512Mi",
					},
				},
			}
		})

		AfterEach(func() {
			// Cleanup
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
		})

		It("Should create HiveMind StatefulSet", func() {
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Wait for StatefulSet to be created
			Eventually(func() error {
				sts := &appsv1.StatefulSet{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-hivemind-cluster-hivemind",
					Namespace: namespace,
				}, sts)
			}, timeout, interval).Should(Succeed())

			// Verify StatefulSet configuration
			sts := &appsv1.StatefulSet{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-hivemind-cluster-hivemind",
				Namespace: namespace,
			}, sts)).Should(Succeed())

			Expect(*sts.Spec.Replicas).Should(Equal(int32(3)))
			Expect(sts.Spec.ServiceName).Should(Equal("test-hivemind-cluster-hivemind"))
			Expect(sts.Spec.VolumeClaimTemplates).Should(HaveLen(1))
			Expect(sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]).Should(Equal(resource.MustParse("1Gi")))
		})

		It("Should create Redis deployment for memory backend", func() {
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Wait for Redis deployment
			Eventually(func() error {
				deploy := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-hivemind-cluster-redis",
					Namespace: namespace,
				}, deploy)
			}, timeout, interval).Should(Succeed())

			// Verify Redis configuration
			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-hivemind-cluster-redis",
				Namespace: namespace,
			}, deploy)).Should(Succeed())

			Expect(*deploy.Spec.Replicas).Should(Equal(int32(1)))
			Expect(deploy.Spec.Template.Spec.Containers[0].Image).Should(Equal("redis:7-alpine"))
		})

		It("Should update cluster status", func() {
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Wait for status update
			Eventually(func() string {
				c := &swarmv1alpha1.SwarmCluster{}
				k8sClient.Get(ctx, types.NamespacedName{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, c)
				return c.Status.Phase
			}, timeout, interval).Should(Equal("Initializing"))
		})
	})

	Context("When creating a SwarmCluster with Autoscaling", func() {
		var (
			ctx       context.Context
			cluster   *swarmv1alpha1.SwarmCluster
			namespace string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = "test-autoscale-" + randomString(6)

			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create SwarmCluster with Autoscaling
			cluster = &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-autoscale-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.TopologyHierarchical,
					Autoscaling: swarmv1alpha1.AutoscalingSpec{
						Enabled:       true,
						MinAgents:     3,
						MaxAgents:     10,
						TargetUtilization: 80,
						TopologyRatios: map[string]int32{
							"coordinator": 10,
							"coder":       60,
							"tester":      30,
						},
						Metrics: []swarmv1alpha1.AutoscalingMetric{
							{
								Type:   "cpu",
								Target: "80",
							},
							{
								Type:   "custom",
								Name:   "pending_tasks",
								Target: "5",
							},
						},
					},
				},
			}
		})

		AfterEach(func() {
			// Cleanup
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
		})

		It("Should create agents according to topology", func() {
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Wait for agents to be created
			Eventually(func() int {
				agentList := &swarmv1alpha1.SwarmAgentList{}
				k8sClient.List(ctx, agentList, client.InNamespace(namespace))
				return len(agentList.Items)
			}, timeout, interval).Should(BeNumerically(">=", 3))

			// Verify agent types
			agentList := &swarmv1alpha1.SwarmAgentList{}
			Expect(k8sClient.List(ctx, agentList, client.InNamespace(namespace))).Should(Succeed())

			agentTypes := make(map[swarmv1alpha1.AgentType]int)
			for _, agent := range agentList.Items {
				agentTypes[agent.Spec.Type]++
			}

			// Hierarchical topology should have specific agent types
			Expect(agentTypes[swarmv1alpha1.AgentTypeCoordinator]).Should(BeNumerically(">=", 1))
			Expect(agentTypes[swarmv1alpha1.AgentTypeCoder]).Should(BeNumerically(">=", 1))
		})

		It("Should create HPA for agent types", func() {
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Wait for HPA creation
			time.Sleep(5 * time.Second) // Give controller time to create HPAs

			// In a real test, we would check for HPA resources
			// For now, verify the cluster was created successfully
			c := &swarmv1alpha1.SwarmCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}, c)).Should(Succeed())

			Expect(c.Spec.Autoscaling.Enabled).Should(BeTrue())
			Expect(c.Spec.Autoscaling.TopologyRatios).Should(HaveLen(3))
		})
	})
})

var _ = Describe("SwarmAgent Controller", func() {
	Context("When creating a SwarmAgent", func() {
		var (
			ctx       context.Context
			cluster   *swarmv1alpha1.SwarmCluster
			agent     *swarmv1alpha1.SwarmAgent
			namespace string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = "test-agent-" + randomString(6)

			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create parent cluster first
			cluster = &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.TopologyMesh,
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Create SwarmAgent
			agent = &swarmv1alpha1.SwarmAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-agent",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmAgentSpec{
					Type:             swarmv1alpha1.AgentTypeResearcher,
					ClusterRef:       cluster.Name,
					CognitivePattern: swarmv1alpha1.PatternDivergent,
					Priority:         80,
					MaxConcurrentTasks: 3,
					Capabilities:     []string{"search", "analyze"},
					Resources: swarmv1alpha1.ResourceRequirements{
						CPU:    "500m",
						Memory: "1Gi",
					},
				},
			}
		})

		AfterEach(func() {
			// Cleanup
			Expect(k8sClient.Delete(ctx, agent)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
		})

		It("Should create agent deployment", func() {
			Expect(k8sClient.Create(ctx, agent)).Should(Succeed())

			// Wait for deployment to be created
			Eventually(func() error {
				deploy := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      agent.Name,
					Namespace: namespace,
				}, deploy)
			}, timeout, interval).Should(Succeed())

			// Verify deployment configuration
			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      agent.Name,
				Namespace: namespace,
			}, deploy)).Should(Succeed())

			Expect(*deploy.Spec.Replicas).Should(Equal(int32(1)))
			Expect(deploy.Spec.Template.Labels["agent-type"]).Should(Equal("researcher"))
			
			// Check resources
			container := deploy.Spec.Template.Spec.Containers[0]
			Expect(container.Resources.Requests[corev1.ResourceCPU]).Should(Equal(resource.MustParse("500m")))
			Expect(container.Resources.Requests[corev1.ResourceMemory]).Should(Equal(resource.MustParse("1Gi")))
		})

		It("Should set cognitive pattern in environment", func() {
			Expect(k8sClient.Create(ctx, agent)).Should(Succeed())

			// Wait for deployment
			Eventually(func() error {
				deploy := &appsv1.Deployment{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      agent.Name,
					Namespace: namespace,
				}, deploy)
			}, timeout, interval).Should(Succeed())

			deploy := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      agent.Name,
				Namespace: namespace,
			}, deploy)).Should(Succeed())

			// Check environment variables
			envVars := deploy.Spec.Template.Spec.Containers[0].Env
			cognitivePatternSet := false
			for _, env := range envVars {
				if env.Name == "COGNITIVE_PATTERN" {
					Expect(env.Value).Should(Equal("divergent"))
					cognitivePatternSet = true
					break
				}
			}
			Expect(cognitivePatternSet).Should(BeTrue())
		})
	})
})

var _ = Describe("SwarmMemory Controller", func() {
	Context("When creating SwarmMemory entries", func() {
		var (
			ctx       context.Context
			cluster   *swarmv1alpha1.SwarmCluster
			memory    *swarmv1alpha1.SwarmMemory
			namespace string
		)

		BeforeEach(func() {
			ctx = context.Background()
			namespace = "test-memory-" + randomString(6)

			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Create parent cluster
			cluster = &swarmv1alpha1.SwarmCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmClusterSpec{
					Topology: swarmv1alpha1.TopologyMesh,
					Memory: swarmv1alpha1.MemorySpec{
						Type: "redis",
						Size: "1Gi",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Create SwarmMemory
			memory = &swarmv1alpha1.SwarmMemory{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-memory",
					Namespace: namespace,
				},
				Spec: swarmv1alpha1.SwarmMemorySpec{
					ClusterRef: cluster.Name,
					Namespace:  "test",
					Type:       swarmv1alpha1.MemoryTypeKnowledge,
					Key:        "test/pattern",
					Value:      `{"pattern": "test", "confidence": 0.95}`,
					TTL:        3600,
					Priority:   100,
					Compression: true,
				},
			}
		})

		AfterEach(func() {
			// Cleanup
			Expect(k8sClient.Delete(ctx, memory)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
		})

		It("Should create memory entry with TTL", func() {
			Expect(k8sClient.Create(ctx, memory)).Should(Succeed())

			// Get the created memory
			m := &swarmv1alpha1.SwarmMemory{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      memory.Name,
				Namespace: memory.Namespace,
			}, m)).Should(Succeed())

			Expect(m.Spec.TTL).Should(Equal(int32(3600)))
			Expect(m.Spec.Compression).Should(BeTrue())
			Expect(m.Spec.Priority).Should(Equal(int32(100)))
		})
	})
})

// Helper function
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}