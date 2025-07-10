# Swarm Operator Troubleshooting Guide

This guide helps you diagnose and resolve common issues with swarm-operator.

## Table of Contents

1. [Diagnostic Tools](#diagnostic-tools)
2. [Common Issues](#common-issues)
3. [Operator Issues](#operator-issues)
4. [Swarm Issues](#swarm-issues)
5. [Agent Issues](#agent-issues)
6. [Task Issues](#task-issues)
7. [Performance Issues](#performance-issues)
8. [Network Issues](#network-issues)
9. [Getting Help](#getting-help)

## Diagnostic Tools

### Built-in Diagnostics

```bash
# Run operator diagnostics
kubectl exec -n swarm-operator deployment/swarm-operator -- /manager diagnostics

# Check operator health
kubectl get pods -n swarm-operator -o wide
kubectl describe pod -n swarm-operator -l app.kubernetes.io/name=swarm-operator
```

### Useful kubectl Commands

```bash
# Get all swarm resources
kubectl get swarms,agents,tasks -A

# Check events
kubectl get events -A --sort-by='.lastTimestamp' | grep -i swarm

# View operator logs with different verbosity
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator --tail=50
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator --since=10m
```

### Debug Script

Create a debug script for quick diagnostics:

```bash
#!/bin/bash
# save as debug-swarm.sh

echo "=== Swarm Operator Debug Info ==="
echo "Operator Status:"
kubectl get deployment -n swarm-operator swarm-operator

echo -e "\nOperator Pods:"
kubectl get pods -n swarm-operator -l app.kubernetes.io/name=swarm-operator

echo -e "\nCRDs:"
kubectl get crds | grep swarm.cloudflow.io

echo -e "\nSwarms:"
kubectl get swarms -A

echo -e "\nAgents:"
kubectl get agents -A

echo -e "\nTasks:"
kubectl get tasks -A

echo -e "\nRecent Events:"
kubectl get events -A --sort-by='.lastTimestamp' | grep -i swarm | tail -20

echo -e "\nOperator Logs (last 50 lines):"
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator --tail=50
```

## Common Issues

### Issue: "No resources found"

**Symptom**: Commands like `kubectl get swarms` return "No resources found"

**Causes**:
1. CRDs not installed
2. Wrong namespace
3. Resources not created yet

**Solution**:
```bash
# Check if CRDs are installed
kubectl get crds | grep swarm.cloudflow.io

# If not, install them
kubectl apply -f config/crd/bases/

# Check all namespaces
kubectl get swarms -A
```

### Issue: "error validating data"

**Symptom**: Error when applying YAML files

**Causes**:
1. Invalid YAML syntax
2. Wrong API version
3. Missing required fields

**Solution**:
```bash
# Validate YAML syntax
kubectl apply -f your-file.yaml --dry-run=client

# Check API versions
kubectl api-resources | grep swarm

# Use kubectl explain for field information
kubectl explain swarm.spec
```

## Operator Issues

### Operator Not Starting

**Symptoms**:
- Operator pod in CrashLoopBackOff
- Operator pod in Error state

**Diagnosis**:
```bash
# Check pod status
kubectl describe pod -n swarm-operator -l app.kubernetes.io/name=swarm-operator

# Check logs
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator --previous

# Check RBAC
kubectl auth can-i --list --as=system:serviceaccount:swarm-operator:swarm-operator
```

**Common Solutions**:

1. **Missing CRDs**:
   ```bash
   kubectl apply -f config/crd/bases/
   ```

2. **RBAC Issues**:
   ```bash
   kubectl apply -f deploy/helm/swarm-operator/templates/rbac.yaml
   ```

3. **Resource Constraints**:
   ```yaml
   # Increase resources in values.yaml
   resources:
     limits:
       cpu: 1000m
       memory: 1Gi
   ```

### Operator Performance Issues

**Symptoms**:
- Slow reconciliation
- High memory usage
- CPU throttling

**Diagnosis**:
```bash
# Check resource usage
kubectl top pod -n swarm-operator

# Check metrics
kubectl port-forward -n swarm-operator svc/swarm-operator-metrics 8080:8080
curl http://localhost:8080/metrics | grep -E "(cpu|memory|reconcile)"
```

**Solutions**:

1. **Increase concurrent reconciles**:
   ```yaml
   env:
     - name: MAX_CONCURRENT_RECONCILES
       value: "20"
   ```

2. **Adjust resource limits**:
   ```bash
   helm upgrade swarm-operator deploy/helm/swarm-operator \
     --set resources.limits.cpu=2000m \
     --set resources.limits.memory=2Gi
   ```

## Swarm Issues

### Swarm Stuck in Pending

**Diagnosis**:
```bash
# Check swarm status
kubectl describe swarm <swarm-name>

# Check events
kubectl get events --field-selector involvedObject.name=<swarm-name>

# Check operator logs
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator | grep <swarm-name>
```

**Common Causes**:

1. **Resource Quota**:
   ```bash
   kubectl describe resourcequota -n <namespace>
   ```

2. **Node Resources**:
   ```bash
   kubectl describe nodes
   kubectl top nodes
   ```

3. **Pod Security Policies**:
   ```bash
   kubectl get psp
   kubectl auth can-i use podsecuritypolicy/<psp-name> --as=system:serviceaccount:<namespace>:default
   ```

### Swarm Not Creating Agents

**Diagnosis**:
```bash
# Check if agents are being created
kubectl get agents -l swarm=<swarm-name>

# Check swarm controller logs
kubectl logs -n <namespace> -l swarm=<swarm-name>,role=coordinator
```

**Solutions**:

1. **Check agent templates**:
   ```yaml
   spec:
     agentTemplate:
       spec:
         resources:
           requests:
             cpu: "100m"
             memory: "128Mi"
   ```

2. **Verify RBAC for agents**:
   ```bash
   kubectl create rolebinding agent-admin \
     --clusterrole=admin \
     --serviceaccount=<namespace>:default \
     -n <namespace>
   ```

## Agent Issues

### Agents Failing to Start

**Diagnosis**:
```bash
# List all agents
kubectl get agents -A -o wide

# Check failing agent
kubectl describe agent <agent-name>
kubectl logs <agent-pod-name>
```

**Common Issues**:

1. **Image Pull Errors**:
   ```bash
   # Check image pull secrets
   kubectl get secrets -n <namespace>
   
   # Create if missing
   kubectl create secret docker-registry regcred \
     --docker-server=<registry> \
     --docker-username=<username> \
     --docker-password=<password>
   ```

2. **Init Container Failures**:
   ```bash
   kubectl logs <pod-name> -c <init-container-name>
   ```

### Agent Communication Issues

**Symptoms**:
- Agents not receiving tasks
- Coordinator can't reach agents

**Diagnosis**:
```bash
# Check network policies
kubectl get networkpolicies -n <namespace>

# Test connectivity
kubectl exec -n <namespace> <coordinator-pod> -- curl http://<agent-service>:8080/health
```

## Task Issues

### Tasks Not Being Assigned

**Diagnosis**:
```bash
# Check task status
kubectl describe task <task-name>

# Check if agents are available
kubectl get agents -l swarm=<swarm-name> -o jsonpath='{.items[*].status.phase}'

# Check task controller logs
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator | grep -i task
```

**Solutions**:

1. **Verify swarm reference**:
   ```yaml
   spec:
     swarmRef:
       name: <existing-swarm-name>
       namespace: <swarm-namespace>  # if different
   ```

2. **Check agent capabilities**:
   ```bash
   kubectl get agents -o jsonpath='{range .items[*]}{.metadata.name}: {.spec.capabilities}{"\n"}{end}'
   ```

### Tasks Stuck in Running

**Diagnosis**:
```bash
# Check task progress
kubectl get task <task-name> -o jsonpath='{.status}'

# Check assigned agents
kubectl get agents -l task=<task-name>

# Check agent logs
kubectl logs -l task=<task-name>
```

**Common Causes**:

1. **Deadlock in task logic**
2. **Agent crashed without updating status**
3. **Network partition**

**Solutions**:

1. **Set task timeout**:
   ```yaml
   spec:
     timeout: 300s  # 5 minutes
   ```

2. **Force task completion**:
   ```bash
   kubectl patch task <task-name> --type merge -p '{"status":{"phase":"Failed"}}'
   ```

## Performance Issues

### Slow Task Execution

**Diagnosis**:
```bash
# Check agent resource usage
kubectl top pods -l swarm=<swarm-name>

# Check node resources
kubectl describe nodes | grep -A 5 "Allocated resources"

# Profile task execution
kubectl exec <agent-pod> -- pprof http://localhost:6060/debug/pprof/profile
```

**Optimizations**:

1. **Increase agent resources**:
   ```yaml
   spec:
     resources:
       requests:
         cpu: "500m"
         memory: "512Mi"
       limits:
         cpu: "1000m"
         memory: "1Gi"
   ```

2. **Enable agent pooling**:
   ```yaml
   spec:
     agentPooling:
       enabled: true
       minReadyAgents: 3
   ```

### High Memory Usage

**Diagnosis**:
```bash
# Check memory metrics
kubectl top pods -A | sort -k4 -h | tail -20

# Get memory profile
kubectl exec -n swarm-operator <operator-pod> -- curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Solutions**:

1. **Tune garbage collection**:
   ```yaml
   env:
     - name: GOGC
       value: "50"  # More aggressive GC
   ```

2. **Limit concurrent operations**:
   ```yaml
   spec:
     maxConcurrentTasks: 10
     maxAgentsPerTask: 3
   ```

## Network Issues

### Service Discovery Problems

**Symptoms**:
- Agents can't find coordinator
- Tasks can't be distributed

**Diagnosis**:
```bash
# Check services
kubectl get svc -n <namespace>

# Test DNS resolution
kubectl exec <pod> -- nslookup <service-name>.<namespace>.svc.cluster.local

# Check endpoints
kubectl get endpoints -n <namespace>
```

**Solutions**:

1. **Verify service selectors**:
   ```bash
   kubectl get svc <service> -o yaml | grep -A 5 selector
   kubectl get pods -l <selector-labels>
   ```

2. **Check CoreDNS**:
   ```bash
   kubectl logs -n kube-system -l k8s-app=kube-dns
   ```

### Ingress Issues

**For external access to swarm APIs**:

```bash
# Check ingress
kubectl get ingress -A
kubectl describe ingress <ingress-name>

# Verify ingress controller
kubectl get pods -n ingress-nginx
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx
```

## Getting Help

### Collect Diagnostic Bundle

```bash
# Create diagnostic bundle
mkdir swarm-diagnostics
cd swarm-diagnostics

# Collect operator info
kubectl logs -n swarm-operator -l app.kubernetes.io/name=swarm-operator > operator.log
kubectl get all -n swarm-operator -o yaml > operator-resources.yaml

# Collect CRD instances
kubectl get swarms,agents,tasks -A -o yaml > crd-instances.yaml

# Collect events
kubectl get events -A --sort-by='.lastTimestamp' > events.log

# Create tarball
tar -czf swarm-diagnostics.tar.gz .
```

### Debugging Tips

1. **Enable debug logging**:
   ```bash
   kubectl set env deployment/swarm-operator -n swarm-operator LOG_LEVEL=debug
   ```

2. **Use kubectl debug**:
   ```bash
   kubectl debug -n <namespace> <pod-name> -it --image=busybox
   ```

3. **Check webhook certificates**:
   ```bash
   kubectl get certificate -n swarm-operator
   kubectl describe certificate -n swarm-operator
   ```

### Community Support

- **GitHub Issues**: [github.com/cloudflow/swarm-operator/issues](https://github.com/cloudflow/swarm-operator/issues)
- **Slack**: [#swarm-operator](https://cloudflow.slack.com)
- **Documentation**: [docs.swarm-operator.io](https://docs.swarm-operator.io)
- **FAQ**: [docs.swarm-operator.io/faq](https://docs.swarm-operator.io/faq)

### Reporting Issues

When reporting issues, include:

1. Swarm operator version
2. Kubernetes version and distribution
3. Diagnostic bundle (see above)
4. Steps to reproduce
5. Expected vs actual behavior

Template:
```markdown
**Environment:**
- Swarm Operator Version: 
- Kubernetes Version: 
- Cloud Provider/Platform: 

**Description:**
Brief description of the issue

**Steps to Reproduce:**
1. 
2. 
3. 

**Expected Behavior:**
What should happen

**Actual Behavior:**
What actually happens

**Logs/Diagnostics:**
```
Attach diagnostic bundle
```
```