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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	swarmv1alpha1 "github.com/claude-flow/swarm-operator/api/v1alpha1"
	"github.com/claude-flow/swarm-operator/controllers"
	"github.com/claude-flow/swarm-operator/pkg/metrics"
	"github.com/claude-flow/swarm-operator/pkg/topology"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = swarmv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// Create client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Setup context
	ctx, cancel = context.WithCancel(context.Background())

	// Start controllers
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	// Setup SwarmCluster controller
	err = (&controllers.SwarmClusterReconciler{
		Client:          k8sManager.GetClient(),
		Scheme:          k8sManager.GetScheme(),
		Recorder:        k8sManager.GetEventRecorderFor("swarmcluster-controller"),
		MetricsCollector: metrics.NewCollector(k8sManager.GetClient()),
		TopologyManager: topology.NewManager(k8sManager.GetClient()),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	// Setup Agent controller
	err = (&controllers.AgentReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("agent-controller"),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	// Start manager
	go func() {
		defer GinkgoRecover()
		err := k8sManager.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Helper functions for E2E tests

// CreateNamespace creates a unique namespace for test isolation
func CreateNamespace(ctx context.Context) string {
	namespace := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	return namespace
}

// DeleteNamespace deletes a namespace and all resources within it
func DeleteNamespace(ctx context.Context, namespace string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
}

// WaitForClusterReady waits for a SwarmCluster to reach Ready state
func WaitForClusterReady(ctx context.Context, name, namespace string, timeout time.Duration) {
	Eventually(func() swarmv1alpha1.ClusterState {
		cluster := &swarmv1alpha1.SwarmCluster{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cluster)
		if err != nil {
			return ""
		}
		return cluster.Status.State
	}, timeout, time.Second).Should(Equal(swarmv1alpha1.ClusterReady))
}

// WaitForAgentReady waits for an Agent to reach Ready state
func WaitForAgentReady(ctx context.Context, name, namespace string, timeout time.Duration) {
	Eventually(func() swarmv1alpha1.AgentState {
		agent := &swarmv1alpha1.Agent{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, agent)
		if err != nil {
			return ""
		}
		return agent.Status.State
	}, timeout, time.Second).Should(Equal(swarmv1alpha1.AgentReady))
}

// WaitForTaskComplete waits for a SwarmTask to complete
func WaitForTaskComplete(ctx context.Context, name, namespace string, timeout time.Duration) {
	Eventually(func() swarmv1alpha1.TaskState {
		task := &swarmv1alpha1.SwarmTask{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, task)
		if err != nil {
			return ""
		}
		return task.Status.State
	}, timeout, time.Second).Should(Equal(swarmv1alpha1.TaskCompleted))
}

// GetClusterAgents returns all agents belonging to a cluster
func GetClusterAgents(ctx context.Context, clusterName, namespace string) []swarmv1alpha1.Agent {
	agents := &swarmv1alpha1.AgentList{}
	labels := client.MatchingLabels{
		"swarm.claudeflow.io/cluster": clusterName,
	}
	Expect(k8sClient.List(ctx, agents, client.InNamespace(namespace), labels)).To(Succeed())
	return agents.Items
}