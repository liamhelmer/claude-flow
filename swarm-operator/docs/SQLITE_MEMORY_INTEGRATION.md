# SQLite Memory Integration for Swarm Operator

## Overview

The Swarm Operator now fully supports the new SQLite-based memory persistence system introduced in Claude Flow v2.0.0-alpha.43. This provides high-performance, persistent memory storage for swarm agents with advanced features like caching, compression, and automatic garbage collection.

## Features

### Core SQLite Memory Features
- **Persistent Storage**: SQLite database with Write-Ahead Logging (WAL)
- **High-Performance Caching**: LRU cache with configurable memory limits
- **Automatic Compression**: Large entries compressed automatically
- **TTL Support**: Automatic expiration of temporary data
- **Tagging & Search**: Flexible data organization and retrieval
- **Migration Support**: Seamless migration from legacy memory systems

### Kubernetes Integration
- **SwarmMemoryStore CRD**: Dedicated resource for memory management
- **Persistent Volumes**: Automatic PVC creation and management
- **StatefulSet Deployment**: Ensures data persistence across restarts
- **Namespace Isolation**: Memory stores deployed in configured namespaces
- **Backup Management**: Automatic and on-demand backups

## Architecture

### Components

1. **SwarmMemoryStore Controller**
   - Manages lifecycle of SQLite memory stores
   - Handles PVC creation and storage allocation
   - Manages backup and migration jobs
   - Monitors memory usage and performance

2. **SwarmCluster Integration**
   - Automatically creates SwarmMemoryStore when SQLite is configured
   - Configures memory settings based on cluster specification
   - Manages ownership and cleanup

3. **Memory Service Pod**
   - Runs the SQLite memory service
   - Exposes gRPC and HTTP endpoints
   - Provides Prometheus metrics
   - Handles garbage collection and compression

## Configuration

### SwarmCluster Memory Configuration

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: my-cluster
spec:
  memory:
    type: sqlite              # Use SQLite backend
    size: "20Gi"             # Storage size
    persistence: true         # Enable persistence
    enableMemoryStore: true   # Create SwarmMemoryStore resource
    sqliteConfig:
      cacheSize: 2000        # Max cached entries
      cacheMemoryMB: 100     # Max cache memory
      enableWAL: true        # Write-Ahead Logging
      enableVacuum: true     # Auto-vacuum
      gcInterval: "10m"      # Garbage collection interval
      backupInterval: "1h"   # Automatic backup interval
```

### SwarmMemoryStore Resource

When `enableMemoryStore` is true, the operator creates:

```yaml
apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmMemoryStore
metadata:
  name: my-cluster-memory
spec:
  type: sqlite
  swarmId: my-cluster
  storageSize: "20Gi"
  cacheSize: 2000
  cacheMemoryMB: 100
  compressionThreshold: 10240
  gcInterval: "10m"
  backupInterval: "1h"
  enableWAL: true
  enableVacuum: true
  mcpMode: true
```

## Usage Patterns

### 1. Basic Memory Operations

Agents can store and retrieve data using the memory service:

```javascript
// Store data
await memory.store('analysis:project-123', {
  status: 'complete',
  findings: [...],
  timestamp: new Date()
}, {
  namespace: 'analysis',
  ttl: 3600,
  tags: ['project', 'complete']
});

// Retrieve data
const data = await memory.retrieve('analysis:project-123', 'analysis');

// Search by pattern
const results = await memory.search({
  pattern: 'analysis:*',
  tags: ['complete'],
  limit: 10
});
```

### 2. Agent Coordination

Agents use memory for coordination:

```javascript
// Store agent state
await swarm.storeAgent('agent-1', {
  id: 'agent-1',
  type: 'analyzer',
  status: 'active',
  capabilities: ['code-review', 'pattern-detection']
});

// Track tasks
await swarm.storeTask('task-1', {
  id: 'task-1',
  description: 'Analyze codebase',
  status: 'in_progress',
  assignedAgents: ['agent-1']
});

// Store learned patterns
await swarm.storePattern('pattern-1', {
  type: 'optimization',
  confidence: 0.92,
  data: { /* pattern details */ }
});
```

### 3. Hive Mind Integration

For consensus and collective intelligence:

```javascript
// Store consensus decisions
await swarm.storeConsensus('decision-1', {
  topic: 'architecture-choice',
  participants: ['agent-1', 'agent-2', 'agent-3'],
  result: 'microservices',
  confidence: 0.85
});

// Track inter-agent communication
await swarm.storeCommunication('agent-1', 'agent-2', {
  type: 'proposal',
  content: 'Suggest using event-driven architecture'
});
```

## Migration from Legacy Systems

### Automatic Migration

Configure migration in SwarmMemoryStore:

```yaml
spec:
  migrateFromLegacy: true
  legacyDataPVC: "legacy-memory-pvc"
```

The operator will:
1. Create a migration job
2. Import data from legacy JSON or SQLite files
3. Transform to new schema
4. Mark migration as complete

### Manual Migration

For custom migration needs:

```bash
kubectl exec -it my-cluster-memory-0 -- \
  node /app/src/memory/migration.js \
  --source=/legacy/memory-store.json \
  --target=/data/memory/swarm-memory.db
```

## Monitoring and Metrics

### Prometheus Metrics

The memory service exposes metrics on port 9091:

- `swarm_memory_entries_total`: Total number of stored entries
- `swarm_memory_cache_hits_total`: Cache hit count
- `swarm_memory_cache_misses_total`: Cache miss count
- `swarm_memory_db_size_bytes`: Database size in bytes
- `swarm_memory_gc_duration_seconds`: GC duration histogram
- `swarm_memory_compression_ratio`: Average compression ratio

### Status Monitoring

Check SwarmMemoryStore status:

```bash
# List all SwarmMemoryStores in the namespace
kubectl get swarmmemorystores -n claude-flow-swarm

# Get detailed status
kubectl describe swarmemorystore my-cluster-memory -n claude-flow-swarm

# Watch for changes
kubectl get swarmmemorystores -n claude-flow-swarm -w
```

Status includes:
- Phase (Initializing, Ready, Error, Migrating, BackingUp)
- Storage readiness
- Database size
- Entry counts (total, agents, tasks, patterns)
- Cache hit rate
- Last backup time

## Backup and Recovery

### Automatic Backups

Configure automatic backups:

```yaml
spec:
  backupInterval: "6h"
  backupRetention: 7  # Keep 7 backups
```

### Manual Backup

Trigger manual backup:

```bash
kubectl annotate swarmemorystore my-cluster-memory \
  swarm.claudeflow.io/backup-requested=true
```

### Restore from Backup

1. Create new SwarmMemoryStore with same configuration
2. Copy backup to new PVC
3. The init container will detect and restore

## Performance Tuning

### Cache Configuration

Adjust cache for workload:

```yaml
sqliteConfig:
  cacheSize: 5000      # More entries for read-heavy
  cacheMemoryMB: 200   # More memory for large values
```

### Compression

Configure compression threshold:

```yaml
spec:
  compressionThreshold: 5120  # Compress entries > 5KB
```

### Garbage Collection

Tune GC for data patterns:

```yaml
sqliteConfig:
  gcInterval: "30m"  # Less frequent for stable data
```

## Troubleshooting

### Common Issues

1. **Memory Pod Not Starting**
   - Check PVC is bound: `kubectl get pvc -n claude-flow-swarm`
   - Check pod logs: `kubectl logs my-cluster-memory-0 -c init-db -n claude-flow-swarm`

2. **High Memory Usage**
   - Review cache settings
   - Check for expired entries not being cleaned
   - Monitor GC metrics

3. **Slow Queries**
   - Check if indexes are created
   - Review search patterns
   - Consider increasing cache size

4. **Migration Failures**
   - Check legacy PVC is accessible
   - Review migration job logs
   - Try manual migration with verbose mode

### Debug Commands

```bash
# Check database integrity
kubectl exec -it my-cluster-memory-0 -n claude-flow-swarm -- \
  sqlite3 /data/memory/swarm-memory.db "PRAGMA integrity_check"

# View table statistics
kubectl exec -it my-cluster-memory-0 -n claude-flow-swarm -- \
  sqlite3 /data/memory/swarm-memory.db "SELECT COUNT(*) FROM memory_store"

# Force garbage collection
kubectl exec -it my-cluster-memory-0 -n claude-flow-swarm -- \
  curl -X POST localhost:8080/admin/gc

# Check SwarmMemoryStore status
kubectl get swarmmemorystores -n claude-flow-swarm
kubectl describe swarmemorystore my-cluster-memory -n claude-flow-swarm
```

## Best Practices

1. **Namespace Organization**
   - Use meaningful namespaces for data segregation
   - Consider namespace-based access control

2. **TTL Usage**
   - Set appropriate TTLs for temporary data
   - Use permanent storage (TTL=0) sparingly

3. **Tagging Strategy**
   - Develop consistent tagging taxonomy
   - Use tags for efficient searching

4. **Backup Policy**
   - Regular backups for critical data
   - Test restore procedures periodically

5. **Resource Allocation**
   - Size storage based on expected growth
   - Monitor usage and scale accordingly

## Integration with CI/CD

### GitHub Actions Example

```yaml
- name: Deploy Swarm with SQLite Memory
  run: |
    # Create namespaces
    kubectl create namespace claude-flow-swarm || true
    kubectl create namespace claude-flow-hivemind || true
    
    # Deploy SwarmCluster
    kubectl apply -f swarm-cluster-sqlite.yaml
    kubectl wait --for=condition=Ready swarmcluster/my-cluster -n claude-flow-swarm
    kubectl wait --for=condition=Ready swarmemorystore/my-cluster-memory -n claude-flow-swarm
```

### Testing Memory Operations

```yaml
- name: Test Memory Store
  run: |
    kubectl exec my-cluster-memory-0 -n claude-flow-swarm -- \
      node -e "
        const { SwarmMemory } = require('./memory/swarm-memory.js');
        const memory = new SwarmMemory({ directory: '/data/memory' });
        await memory.initialize();
        await memory.store('test-key', { data: 'test' });
        const result = await memory.retrieve('test-key');
        console.log('Test passed:', result.data === 'test');
      "
```

## Conclusion

The SQLite memory integration provides a robust, scalable solution for persistent memory in swarm operations. With automatic management, comprehensive monitoring, and seamless migration paths, it enables sophisticated multi-agent coordination patterns while maintaining high performance and reliability.

For more details on the memory API, see the [Memory Module README](https://github.com/ruvnet/claude-flow/blob/main/src/memory/README.md).