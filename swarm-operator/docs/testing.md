# Claude Flow Swarm Operator Testing Guide

## Overview

The Claude Flow Swarm Operator includes a comprehensive test suite covering unit tests, integration tests, and end-to-end tests. This guide describes how to run tests and contribute new test cases.

## Test Structure

```
swarm-operator/
├── pkg/                     # Package unit tests
│   ├── topology/
│   │   └── manager_test.go
│   ├── metrics/
│   │   └── collector_test.go
│   └── utils/
│       └── task_distributor_test.go
├── controllers/            # Controller tests
│   ├── swarmcluster_controller_test.go
│   └── agent_controller_test.go
├── internal/
│   ├── controller/        # Internal controller tests
│   │   ├── suite_test.go
│   │   └── swarmtask_controller_test.go
│   └── test/             # Test utilities
│       └── fixtures.go
└── e2e/                  # End-to-end tests
    ├── e2e_suite_test.go
    └── swarm_cluster_test.go
```

## Running Tests

### All Tests
```bash
make test
```

### Unit Tests Only
```bash
make test-unit
```

### Integration Tests
```bash
make test-integration
```

### E2E Tests
```bash
make test-e2e
```

### With Race Detection
```bash
make test-race
```

### Generate Coverage Report
```bash
make test-coverage
# Opens coverage.html in your browser
```

### Run Benchmarks
```bash
make test-benchmark
```

## Test Categories

### Unit Tests

Unit tests focus on individual components in isolation:

- **Topology Manager**: Tests topology validation and agent connection calculations
- **Task Distributor**: Tests task assignment logic and workload balancing
- **Metrics Collector**: Tests Prometheus metric collection and registration
- **Utilities**: Tests helper functions and condition management

Example unit test:
```go
var _ = Describe("TopologyManager", func() {
    It("should validate mesh topology", func() {
        cluster := &swarmv1alpha1.SwarmCluster{
            Spec: swarmv1alpha1.SwarmClusterSpec{
                Topology: swarmv1alpha1.MeshTopology,
                Size:     3,
            },
        }
        err := manager.ValidateTopology(cluster)
        Expect(err).NotTo(HaveOccurred())
    })
})
```

### Integration Tests

Integration tests verify controller behavior with a real Kubernetes API:

- **SwarmCluster Controller**: Tests cluster lifecycle, scaling, and agent management
- **Agent Controller**: Tests agent state management and connectivity
- **SwarmTask Controller**: Tests task distribution and completion

Uses envtest for a lightweight Kubernetes API server:
```go
testEnv = &envtest.Environment{
    CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
}
```

### E2E Tests

End-to-end tests verify complete workflows:

- **Cluster Creation**: Tests full cluster deployment with agents
- **Task Distribution**: Tests task assignment and execution
- **Failure Recovery**: Tests resilience and self-healing
- **Performance**: Tests scaling and concurrent operations

Example E2E test:
```go
It("should create a functional mesh topology swarm", func() {
    By("Creating a SwarmCluster")
    // ... create cluster
    
    By("Waiting for cluster to become ready")
    WaitForClusterReady(ctx, cluster.Name, namespace, 2*time.Minute)
    
    By("Verifying all agents are created and ready")
    // ... verify agents
})
```

## Test Utilities

### Fixtures

Use test fixtures for consistent test data:

```go
// Create a ready cluster
cluster := test.ReadySwarmCluster("test-cluster", "default")

// Create agents
agents := test.CreateAgentList("test-cluster", "default", 5,
    swarmv1alpha1.ResearcherAgent,
    swarmv1alpha1.CoderAgent,
    swarmv1alpha1.AnalystAgent)

// Create a running task
task := test.RunningSwarmTask("test-task", "default", "test-cluster",
    []string{"agent-1", "agent-2"})
```

### Helper Functions

Common test helpers are available:

```go
// Wait for cluster ready
WaitForClusterReady(ctx, name, namespace, timeout)

// Get all cluster agents
agents := GetClusterAgents(ctx, clusterName, namespace)

// Create isolated namespace
namespace := CreateNamespace(ctx)
defer DeleteNamespace(ctx, namespace)
```

## Writing Tests

### Test Structure

Follow the Ginkgo BDD style:

```go
var _ = Describe("Component", func() {
    Context("when condition", func() {
        It("should behavior", func() {
            // Arrange
            // Act
            // Assert
        })
    })
})
```

### Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up resources after tests
3. **Timeouts**: Use reasonable timeouts for async operations
4. **Assertions**: Use clear, specific assertions
5. **Error Cases**: Test both success and failure scenarios

### Mocking

Use gomock for mocking dependencies:

```go
ctrl := gomock.NewController(GinkgoT())
defer ctrl.Finish()

mockClient := mocks.NewMockClient(ctrl)
mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
```

## Continuous Integration

Tests run automatically on:
- Push to main/develop branches
- Pull requests
- Scheduled nightly builds

CI includes:
- Unit tests with coverage
- Integration tests
- E2E tests
- Linting
- Security scanning
- Generated code verification

## Troubleshooting

### Common Issues

1. **envtest binary not found**
   ```bash
   make envtest
   ```

2. **CRDs not found**
   ```bash
   make manifests
   ```

3. **Timeout in E2E tests**
   - Increase timeout in test
   - Check for resource constraints

4. **Flaky tests**
   - Add Eventually() with timeout
   - Check for race conditions
   - Ensure proper cleanup

### Debug Mode

Run tests with verbose output:
```bash
go test -v ./...
```

Run specific test:
```bash
go test -v ./controllers -run TestSwarmClusterReconciliation
```

## Coverage Requirements

- Overall: 80% minimum
- Critical paths: 90% minimum
- New code: 85% minimum

Check coverage:
```bash
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out
```

## Performance Testing

Run benchmarks:
```bash
go test -bench=. -benchmem ./pkg/...
```

Profile CPU/memory:
```bash
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof
```

## Contributing Tests

When adding new features:
1. Write unit tests for new functions
2. Add integration tests for controllers
3. Include E2E tests for user workflows
4. Update test documentation
5. Ensure CI passes

## Test Matrix

| Component | Unit | Integration | E2E | Benchmark |
|-----------|------|-------------|-----|-----------|
| Topology Manager | ✓ | ✓ | ✓ | ✓ |
| Task Distributor | ✓ | ✓ | ✓ | ✓ |
| Metrics Collector | ✓ | ✓ | - | ✓ |
| SwarmCluster Controller | ✓ | ✓ | ✓ | - |
| Agent Controller | ✓ | ✓ | ✓ | - |
| SwarmTask Controller | ✓ | ✓ | ✓ | - |

## Resources

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [envtest Guide](https://book.kubebuilder.io/reference/envtest.html)
- [Go Testing](https://golang.org/pkg/testing/)