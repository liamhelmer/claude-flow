// Minimal MCP Server for Kubernetes
const http = require('http');
const os = require('os');

const PORT = process.env.PORT || 3000;
const GRPC_PORT = process.env.GRPC_PORT || 50051;

// Health check endpoint
const server = http.createServer((req, res) => {
  if (req.url === '/health' || req.url === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      status: 'healthy',
      timestamp: new Date().toISOString(),
      hostname: os.hostname()
    }));
  } else if (req.url === '/ready' || req.url === '/readyz') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      status: 'ready',
      timestamp: new Date().toISOString()
    }));
  } else if (req.url === '/') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({
      name: 'claude-flow-mcp',
      version: '2.0.0',
      mode: process.env.MCP_MODE || 'kubernetes',
      endpoints: {
        health: '/health',
        ready: '/ready',
        grpc: `0.0.0.0:${GRPC_PORT}`
      }
    }));
  } else {
    res.writeHead(404);
    res.end('Not Found');
  }
});

server.listen(PORT, () => {
  console.log(`🐝 MCP Server running on port ${PORT}`);
  console.log(`Mode: ${process.env.MCP_MODE || 'kubernetes'}`);
  console.log(`Namespace: ${process.env.NAMESPACE || 'unknown'}`);
  console.log(`Pod: ${process.env.POD_NAME || 'unknown'}`);
});