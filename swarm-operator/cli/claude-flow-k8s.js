#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs').promises;

// Kubernetes commands for claude-flow
const k8sCommands = {
  'swarm-deploy': {
    description: 'Deploy a swarm to Kubernetes cluster',
    handler: deploySwarm
  },
  'swarm-status': {
    description: 'Check status of swarms in Kubernetes',
    handler: swarmStatus
  },
  'task-create': {
    description: 'Create a swarm task in Kubernetes',
    handler: createTask
  },
  'task-monitor': {
    description: 'Monitor task execution in Kubernetes',
    handler: monitorTask
  },
  'github-app-setup': {
    description: 'Setup GitHub App credentials in Kubernetes',
    handler: setupGitHubApp
  },
  'operator-deploy': {
    description: 'Deploy the swarm operator to Kubernetes',
    handler: deployOperator
  },
  'operator-logs': {
    description: 'View swarm operator logs',
    handler: operatorLogs
  }
};

// Helper to run kubectl commands
async function kubectl(args) {
  return new Promise((resolve, reject) => {
    const proc = spawn('kubectl', args, { stdio: 'inherit' });
    proc.on('close', (code) => {
      if (code === 0) resolve();
      else reject(new Error(`kubectl exited with code ${code}`));
    });
  });
}

// Helper to run kubectl and capture output
async function kubectlOutput(args) {
  return new Promise((resolve, reject) => {
    let stdout = '';
    let stderr = '';
    const proc = spawn('kubectl', args);
    proc.stdout.on('data', (data) => stdout += data);
    proc.stderr.on('data', (data) => stderr += data);
    proc.on('close', (code) => {
      if (code === 0) resolve(stdout);
      else reject(new Error(stderr || `kubectl exited with code ${code}`));
    });
  });
}

// Deploy a swarm to Kubernetes
async function deploySwarm(args) {
  const swarmName = args[0] || 'claude-flow-swarm';
  const topology = args[1] || 'mesh';
  const maxAgents = args[2] || '5';
  
  console.log(`🐝 Deploying swarm '${swarmName}' with ${topology} topology...`);
  
  const swarmYaml = `apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmCluster
metadata:
  name: ${swarmName}
  namespace: default
spec:
  topology: ${topology}
  size: ${maxAgents}
  agents:
    - type: coordinator
      replicas: 1
    - type: researcher
      replicas: 2
    - type: coder
      replicas: 2
    - type: analyst
      replicas: 1
    - type: tester
      replicas: 1
`;

  // Write temp file
  const tempFile = `/tmp/swarm-${Date.now()}.yaml`;
  await fs.writeFile(tempFile, swarmYaml);
  
  try {
    await kubectl(['apply', '-f', tempFile]);
    console.log(`✅ Swarm '${swarmName}' deployed successfully!`);
  } finally {
    await fs.unlink(tempFile).catch(() => {});
  }
}

// Check swarm status
async function swarmStatus(args) {
  const swarmName = args[0];
  
  console.log('🔍 Checking swarm status...\n');
  
  if (swarmName) {
    await kubectl(['get', 'swarmcluster', swarmName, '-o', 'yaml']);
  } else {
    console.log('📊 All Swarms:');
    await kubectl(['get', 'swarmclusters', '-o', 'wide']);
    
    console.log('\n👥 All Agents:');
    await kubectl(['get', 'agents', '-o', 'wide']);
    
    console.log('\n📋 All Tasks:');
    await kubectl(['get', 'swarmtasks', '-o', 'wide']);
  }
}

// Create a swarm task
async function createTask(args) {
  const taskName = args[0] || `task-${Date.now()}`;
  const taskDescription = args[1] || 'Analyze and optimize code';
  const swarmRef = args[2] || 'claude-flow-swarm';
  const priority = args[3] || 'medium';
  
  console.log(`📝 Creating task '${taskName}'...`);
  
  const taskYaml = `apiVersion: swarm.claudeflow.io/v1alpha1
kind: SwarmTask
metadata:
  name: ${taskName}
  namespace: default
spec:
  swarmRef: ${swarmRef}
  task: "${taskDescription}"
  priority: ${priority}
  strategy: adaptive
  timeout: "30m"
`;

  const tempFile = `/tmp/task-${Date.now()}.yaml`;
  await fs.writeFile(tempFile, taskYaml);
  
  try {
    await kubectl(['apply', '-f', tempFile]);
    console.log(`✅ Task '${taskName}' created successfully!`);
    
    // Start monitoring
    console.log('\n📊 Monitoring task execution...');
    await monitorTask([taskName]);
  } finally {
    await fs.unlink(tempFile).catch(() => {});
  }
}

// Monitor task execution
async function monitorTask(args) {
  const taskName = args[0];
  
  if (!taskName) {
    console.error('❌ Task name required');
    return;
  }
  
  let lastStatus = '';
  let attempts = 0;
  const maxAttempts = 120; // 10 minutes with 5-second intervals
  
  while (attempts < maxAttempts) {
    try {
      const output = await kubectlOutput(['get', 'swarmtask', taskName, '-o', 'jsonpath={.status.phase}']);
      const status = output.trim();
      
      if (status !== lastStatus) {
        console.log(`\n⏳ Task status: ${status}`);
        lastStatus = status;
        
        // Get more details
        const message = await kubectlOutput(['get', 'swarmtask', taskName, '-o', 'jsonpath={.status.message}']);
        if (message) console.log(`   ${message}`);
      }
      
      if (status === 'Completed' || status === 'Failed') {
        console.log(`\n${status === 'Completed' ? '✅' : '❌'} Task ${status.toLowerCase()}`);
        
        // Show job logs if available
        const jobName = `swarm-job-${taskName}`;
        console.log('\n📜 Job logs:');
        await kubectl(['logs', `job/${jobName}`, '--tail=50']).catch(() => {
          console.log('   (No logs available)');
        });
        
        break;
      }
      
      await new Promise(resolve => setTimeout(resolve, 5000));
      attempts++;
    } catch (error) {
      console.error('❌ Error monitoring task:', error.message);
      break;
    }
  }
}

// Setup GitHub App credentials
async function setupGitHubApp(args) {
  const privateKeyPath = args[0];
  const appId = args[1];
  const installationId = args[2];
  const clientId = args[3];
  
  if (!privateKeyPath || !appId || !installationId) {
    console.error('❌ Usage: claude-flow-k8s github-app-setup <private-key-path> <app-id> <installation-id> [client-id]');
    return;
  }
  
  console.log('🔐 Setting up GitHub App credentials...');
  
  // Check if private key exists
  try {
    await fs.access(privateKeyPath);
  } catch {
    console.error(`❌ Private key file not found: ${privateKeyPath}`);
    return;
  }
  
  // Create secret
  const secretArgs = [
    'create', 'secret', 'generic', 'github-app-credentials',
    `--from-file=private-key=${privateKeyPath}`,
    `--from-literal=app-id=${appId}`,
    `--from-literal=installation-id=${installationId}`,
    '--namespace=default',
    '--dry-run=client', '-o', 'yaml'
  ];
  
  if (clientId) {
    secretArgs.splice(-3, 0, `--from-literal=client-id=${clientId}`);
  }
  
  try {
    // First generate the YAML
    const yamlOutput = await kubectlOutput(secretArgs);
    
    // Then apply it using a different approach
    const applyProc = spawn('kubectl', ['apply', '-f', '-']);
    applyProc.stdin.write(yamlOutput);
    applyProc.stdin.end();
    
    await new Promise((resolve, reject) => {
      applyProc.on('close', (code) => {
        if (code === 0) {
          console.log('✅ GitHub App credentials configured successfully!');
          resolve();
        } else {
          reject(new Error(`kubectl apply failed with code ${code}`));
        }
      });
    });
    
    // Verify
    console.log('\n📋 Credential details:');
    await kubectl(['describe', 'secret', 'github-app-credentials']);
  } catch (error) {
    console.error('❌ Failed to setup GitHub App credentials:', error.message);
  }
}

// Deploy the swarm operator
async function deployOperator(args) {
  const version = args[0] || 'latest';
  const namespace = args[1] || 'swarm-system';
  
  console.log(`🚀 Deploying Swarm Operator v${version} to namespace '${namespace}'...`);
  
  // Create namespace if needed
  await kubectl(['create', 'namespace', namespace]).catch(() => {});
  
  // Apply CRDs
  console.log('📋 Installing CRDs...');
  const crds = ['swarmcluster-crd.yaml', 'agent-crd.yaml', 'swarmtask-crd.yaml'];
  for (const crd of crds) {
    await kubectl(['apply', '-f', `https://raw.githubusercontent.com/claude-flow/swarm-operator/main/deploy/crds/${crd}`]);
  }
  
  // Deploy operator
  console.log('🤖 Deploying operator...');
  await kubectl(['apply', '-f', `https://raw.githubusercontent.com/claude-flow/swarm-operator/main/deploy/operator.yaml`, '-n', namespace]);
  
  console.log('✅ Swarm Operator deployed successfully!');
  
  // Check status
  console.log('\n📊 Operator status:');
  await kubectl(['get', 'pods', '-n', namespace, '-l', 'app=swarm-operator']);
}

// View operator logs
async function operatorLogs(args) {
  const namespace = args[0] || 'swarm-system';
  const follow = args.includes('-f') || args.includes('--follow');
  
  console.log(`📜 Swarm Operator logs (namespace: ${namespace})...`);
  
  const logArgs = ['logs', '-n', namespace, '-l', 'app=swarm-operator', '--tail=100'];
  if (follow) logArgs.push('-f');
  
  await kubectl(logArgs);
}

// Main CLI handler
async function main() {
  const args = process.argv.slice(2);
  const command = args[0];
  const commandArgs = args.slice(1);
  
  if (!command || command === 'help' || command === '--help') {
    console.log('🐝 Claude Flow Kubernetes CLI\n');
    console.log('Commands:');
    Object.entries(k8sCommands).forEach(([cmd, info]) => {
      console.log(`  ${cmd.padEnd(20)} ${info.description}`);
    });
    console.log('\nExamples:');
    console.log('  claude-flow-k8s swarm-deploy my-swarm hierarchical 8');
    console.log('  claude-flow-k8s task-create analyze-code "Analyze repository for optimization" my-swarm high');
    console.log('  claude-flow-k8s github-app-setup /path/to/key.pem 123456 789012 Iv23liABCD');
    console.log('  claude-flow-k8s swarm-status');
    console.log('  claude-flow-k8s operator-logs -f');
    return;
  }
  
  const handler = k8sCommands[command];
  if (!handler) {
    console.error(`❌ Unknown command: ${command}`);
    console.log('Run "claude-flow-k8s help" for available commands');
    return;
  }
  
  try {
    await handler.handler(commandArgs);
  } catch (error) {
    console.error(`❌ Error: ${error.message}`);
    process.exit(1);
  }
}

// Run CLI
if (require.main === module) {
  main().catch(console.error);
}

module.exports = { k8sCommands };