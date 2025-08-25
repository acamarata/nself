const express = require('express');
const helmet = require('helmet');
const cors = require('cors');
const morgan = require('morgan');
const { Pool } = require('pg');
const redis = require('redis');

// Initialize Express app
const app = express();
const port = process.env.PORT || 3000;
const serviceName = process.env.SERVICE_NAME || 'js-service';

// PostgreSQL connection
const pool = new Pool({
  connectionString: process.env.DATABASE_URL || 
    `postgresql://${process.env.POSTGRES_USER}:${process.env.POSTGRES_PASSWORD}@${process.env.POSTGRES_HOST}:5432/${process.env.POSTGRES_DB}`
});

// Redis connection (optional)
let redisClient = null;
if (process.env.REDIS_URL || process.env.REDIS_ENABLED === 'true') {
  redisClient = redis.createClient({
    url: process.env.REDIS_URL || `redis://${process.env.REDIS_HOST || 'redis'}:6379`
  });
  
  redisClient.on('error', (err) => console.error('Redis Client Error:', err));
  redisClient.on('connect', () => console.log('Redis connected successfully'));
  
  redisClient.connect().catch(err => {
    console.warn('Redis connection failed, continuing without cache:', err.message);
    redisClient = null;
  });
}

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(morgan('combined'));

// Health check endpoint
app.get('/health', async (req, res) => {
  try {
    // Check database connection
    await pool.query('SELECT 1');
    
    // Check Redis if available
    let redisStatus = 'not configured';
    if (redisClient) {
      try {
        await redisClient.ping();
        redisStatus = 'healthy';
      } catch (err) {
        redisStatus = 'unhealthy';
      }
    }
    
    res.json({
      status: 'healthy',
      service: serviceName,
      timestamp: new Date().toISOString(),
      checks: {
        database: 'healthy',
        redis: redisStatus
      }
    });
  } catch (error) {
    res.status(503).json({
      status: 'unhealthy',
      service: serviceName,
      error: error.message,
      timestamp: new Date().toISOString()
    });
  }
});

// Root endpoint
app.get('/', (req, res) => {
  res.json({
    message: `Hello from ${serviceName}! 🚀`,
    service: serviceName,
    version: '1.0.0',
    environment: process.env.ENV || 'development',
    timestamp: new Date().toISOString()
  });
});

// Status endpoint
app.get('/status', async (req, res) => {
  const dbConnected = await pool.query('SELECT current_database(), version()').then(() => true).catch(() => false);
  const redisConnected = redisClient ? await redisClient.ping().then(() => true).catch(() => false) : false;
  
  res.json({
    service: serviceName,
    status: 'running',
    uptime: process.uptime(),
    memory: process.memoryUsage(),
    environment: {
      node: process.version,
      env: process.env.ENV || 'development'
    },
    connections: {
      database: dbConnected,
      redis: redisConnected
    },
    timestamp: new Date().toISOString()
  });
});

// Example API endpoint
app.get('/api/example', async (req, res) => {
  try {
    // Example database query
    const result = await pool.query('SELECT NOW() as current_time, current_database() as database');
    
    // Example Redis cache
    const cacheKey = 'last_request';
    if (redisClient) {
      await redisClient.set(cacheKey, new Date().toISOString(), { EX: 60 });
      const cached = await redisClient.get(cacheKey);
      
      res.json({
        data: result.rows[0],
        cached: cached,
        source: 'live'
      });
    } else {
      res.json({
        data: result.rows[0],
        source: 'live',
        cache: 'not available'
      });
    }
  } catch (error) {
    console.error('API error:', error);
    res.status(500).json({ 
      error: 'Internal server error',
      message: error.message 
    });
  }
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('Unhandled error:', err);
  res.status(500).json({ 
    error: 'Something went wrong!',
    message: err.message 
  });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({ 
    error: 'Not found',
    path: req.path 
  });
});

// Start server
const server = app.listen(port, '0.0.0.0', () => {
  console.log(`${serviceName} listening on port ${port}`);
  console.log(`Environment: ${process.env.ENV || 'development'}`);
  console.log(`Health check: http://localhost:${port}/health`);
});

// Graceful shutdown
const gracefulShutdown = async (signal) => {
  console.log(`${signal} received, starting graceful shutdown...`);
  
  server.close(() => {
    console.log('HTTP server closed');
  });
  
  try {
    await pool.end();
    console.log('Database connections closed');
  } catch (err) {
    console.error('Error closing database:', err);
  }
  
  if (redisClient) {
    try {
      await redisClient.quit();
      console.log('Redis connection closed');
    } catch (err) {
      console.error('Error closing Redis:', err);
    }
  }
  
  process.exit(0);
};

process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
process.on('SIGINT', () => gracefulShutdown('SIGINT'));