<?php
require 'vendor/autoload.php';

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;
use Slim\Exception\HttpNotFoundException;

// Configuration
$serviceName = $_ENV['SERVICE_NAME'] ?? 'php-service';
$port = $_ENV['PORT'] ?? '8080';
$env = $_ENV['ENV'] ?? 'development';
$startTime = microtime(true);

// Database configuration
$dbHost = $_ENV['POSTGRES_HOST'] ?? 'postgres';
$dbPort = $_ENV['POSTGRES_PORT'] ?? '5432';
$dbName = $_ENV['POSTGRES_DB'] ?? 'nself';
$dbUser = $_ENV['POSTGRES_USER'] ?? 'postgres';
$dbPass = $_ENV['POSTGRES_PASSWORD'] ?? 'postgres';

// Create database connection
function getDb() {
    global $dbHost, $dbPort, $dbName, $dbUser, $dbPass;
    try {
        $dsn = "pgsql:host=$dbHost;port=$dbPort;dbname=$dbName";
        return new PDO($dsn, $dbUser, $dbPass, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC
        ]);
    } catch (PDOException $e) {
        error_log("Database connection error: " . $e->getMessage());
        return null;
    }
}

// Create Redis connection
function getRedis() {
    if (($_ENV['REDIS_ENABLED'] ?? 'false') !== 'true') {
        return null;
    }
    
    try {
        $redis = new Redis();
        $redis->connect($_ENV['REDIS_HOST'] ?? 'redis', 6379);
        return $redis;
    } catch (Exception $e) {
        error_log("Redis connection error: " . $e->getMessage());
        return null;
    }
}

$db = getDb();
$redis = getRedis();

// Create Slim app
$app = AppFactory::create();

// Add error middleware
$app->addErrorMiddleware(
    $env !== 'production', 
    true, 
    true
);

// Add CORS middleware
$app->add(function (Request $request, $handler) {
    $response = $handler->handle($request);
    return $response
        ->withHeader('Access-Control-Allow-Origin', '*')
        ->withHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization')
        ->withHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
        ->withHeader('Content-Type', 'application/json');
});

// Handle OPTIONS requests
$app->options('/{routes:.+}', function (Request $request, Response $response, $args) {
    return $response;
});

// Root endpoint
$app->get('/', function (Request $request, Response $response) use ($serviceName, $env) {
    $data = [
        'message' => "Hello from $serviceName! 🚀",
        'service' => $serviceName,
        'version' => '1.0.0',
        'environment' => $env,
        'timestamp' => gmdate('c')
    ];
    $response->getBody()->write(json_encode($data));
    return $response;
});

// Health check
$app->get('/health', function (Request $request, Response $response) use ($serviceName, $db, $redis) {
    $checks = [];
    $status = 'healthy';
    
    // Check database
    if ($db) {
        try {
            $db->query('SELECT 1');
            $checks['database'] = 'healthy';
        } catch (PDOException $e) {
            $checks['database'] = 'unhealthy: ' . $e->getMessage();
            $status = 'unhealthy';
        }
    } else {
        $checks['database'] = 'not configured';
    }
    
    // Check Redis
    if ($redis) {
        try {
            $redis->ping();
            $checks['redis'] = 'healthy';
        } catch (Exception $e) {
            $checks['redis'] = 'unhealthy';
            if ($status === 'healthy') {
                $status = 'degraded';
            }
        }
    } else {
        $checks['redis'] = 'not configured';
    }
    
    $data = [
        'status' => $status,
        'service' => $serviceName,
        'timestamp' => gmdate('c'),
        'checks' => $checks
    ];
    
    $response->getBody()->write(json_encode($data));
    return $response->withStatus($status === 'healthy' ? 200 : 503);
});

// Status endpoint
$app->get('/status', function (Request $request, Response $response) use ($serviceName, $env, $startTime, $db, $redis) {
    $dbConnected = false;
    $redisConnected = false;
    
    if ($db) {
        try {
            $db->query('SELECT 1');
            $dbConnected = true;
        } catch (PDOException $e) {
            $dbConnected = false;
        }
    }
    
    if ($redis) {
        try {
            $redis->ping();
            $redisConnected = true;
        } catch (Exception $e) {
            $redisConnected = false;
        }
    }
    
    $data = [
        'service' => $serviceName,
        'status' => 'running',
        'uptime' => microtime(true) - $startTime,
        'memory_mb' => memory_get_usage(true) / 1024 / 1024,
        'environment' => [
            'php' => phpversion(),
            'env' => $env,
            'os' => PHP_OS
        ],
        'connections' => [
            'database' => $dbConnected,
            'redis' => $redisConnected
        ],
        'timestamp' => gmdate('c')
    ];
    
    $response->getBody()->write(json_encode($data));
    return $response;
});

// Example API endpoint
$app->get('/api/example', function (Request $request, Response $response) use ($db, $redis) {
    $data = [];
    
    // Database query
    if ($db) {
        try {
            $stmt = $db->query('SELECT NOW() as current_time, current_database() as database');
            $data = $stmt->fetch();
        } catch (PDOException $e) {
            $response->getBody()->write(json_encode(['error' => $e->getMessage()]));
            return $response->withStatus(500);
        }
    } else {
        $data['message'] = 'Database not connected';
    }
    
    // Redis cache example
    $cached = null;
    if ($redis) {
        try {
            $key = 'last_request';
            $value = gmdate('c');
            $redis->setex($key, 60, $value);
            $cached = $redis->get($key);
        } catch (Exception $e) {
            // Ignore Redis errors
        }
    }
    
    $result = [
        'data' => $data,
        'source' => 'live'
    ];
    
    if ($cached) {
        $result['cached'] = $cached;
    }
    
    $response->getBody()->write(json_encode($result));
    return $response;
});

// 404 handler
$app->map(['GET', 'POST', 'PUT', 'DELETE', 'PATCH'], '/{routes:.+}', 
    function (Request $request, Response $response) {
        $data = [
            'error' => 'Not found',
            'path' => $request->getUri()->getPath()
        ];
        $response->getBody()->write(json_encode($data));
        return $response->withStatus(404);
    }
);

// Run app
echo "$serviceName listening on port $port\n";
echo "Environment: $env\n";
echo "Health check: http://localhost:$port/health\n";

$app->run();