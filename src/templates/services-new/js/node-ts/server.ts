import * as http from 'http';
import * as url from 'url';

const PORT = process.env.PORT || 3000;
const SERVICE_NAME = process.env.SERVICE_NAME || 'node-ts';

interface HealthResponse {
  status: string;
  service: string;
}

interface ApiResponse {
  message?: string;
  timestamp?: string;
  version?: string;
  data?: any;
  echo?: any;
  error?: string;
}

// Create HTTP server
const server = http.createServer((req: http.IncomingMessage, res: http.ServerResponse) => {
  const parsedUrl = url.parse(req.url || '', true);
  const pathname = parsedUrl.pathname;
  const method = req.method;

  // Enable CORS
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  
  // Handle preflight
  if (method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  // Routes
  if (pathname === '/health') {
    const response: HealthResponse = { status: 'healthy', service: SERVICE_NAME };
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(response));
  } else if (pathname === '/') {
    const response: ApiResponse = { 
      message: `Hello from ${SERVICE_NAME}`,
      timestamp: new Date().toISOString(),
      version: '1.0.0'
    };
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(response));
  } else if (pathname === '/api/data' && method === 'GET') {
    const response: ApiResponse = { 
      data: [
        { id: 1, name: 'Item 1' },
        { id: 2, name: 'Item 2' }
      ]
    };
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(response));
  } else if (pathname === '/api/echo' && method === 'POST') {
    let body = '';
    req.on('data', (chunk) => {
      body += chunk.toString();
    });
    req.on('end', () => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      try {
        const data = JSON.parse(body);
        const response: ApiResponse = { echo: data };
        res.end(JSON.stringify(response));
      } catch (e) {
        const response: ApiResponse = { error: 'Invalid JSON' };
        res.end(JSON.stringify(response));
      }
    });
  } else {
    const response: ApiResponse = { error: 'Not found' };
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(response));
  }
});

// Start server
server.listen(PORT, () => {
  console.log(`${SERVICE_NAME} server running on http://localhost:${PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM signal received: closing HTTP server');
  server.close(() => {
    console.log('HTTP server closed');
  });
});