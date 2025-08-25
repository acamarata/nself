require 'sinatra'
require 'sinatra/json'
require 'sinatra/cors'
require 'pg'
require 'redis'
require 'json'
require 'time'

# Configuration
SERVICE_NAME = ENV['SERVICE_NAME'] || 'rb-service'
PORT = ENV['PORT'] || '4567'
ENV_MODE = ENV['ENV'] || 'development'

# Set up Sinatra
set :port, PORT
set :bind, '0.0.0.0'
set :environment, ENV_MODE == 'production' ? :production : :development
set :show_exceptions, false

# Enable CORS
set :allow_origin, '*'
set :allow_methods, 'GET,HEAD,POST,PUT,DELETE,OPTIONS'
set :allow_headers, 'content-type,if-modified-since'
set :expose_headers, 'location,link'

# Database connection
DB_CONFIG = {
  host: ENV['POSTGRES_HOST'] || 'postgres',
  port: 5432,
  dbname: ENV['POSTGRES_DB'] || 'nself',
  user: ENV['POSTGRES_USER'] || 'postgres',
  password: ENV['POSTGRES_PASSWORD'] || 'postgres'
}

def get_db_connection
  PG.connect(DB_CONFIG)
rescue PG::Error => e
  puts "Database connection error: #{e.message}"
  nil
end

# Redis connection
def get_redis_connection
  return nil unless ENV['REDIS_ENABLED'] == 'true'
  
  Redis.new(
    host: ENV['REDIS_HOST'] || 'redis',
    port: 6379,
    db: 0
  )
rescue Redis::BaseError => e
  puts "Redis connection error: #{e.message}"
  nil
end

$db = get_db_connection
$redis = get_redis_connection
$start_time = Time.now

# Middleware
before do
  content_type :json
  headers 'Access-Control-Allow-Origin' => '*',
          'Access-Control-Allow-Methods' => 'GET, POST, PUT, DELETE, OPTIONS',
          'Access-Control-Allow-Headers' => 'Content-Type, Authorization'
end

# OPTIONS requests for CORS
options '*' do
  200
end

# Root endpoint
get '/' do
  json({
    message: "Hello from #{SERVICE_NAME}! 🚀",
    service: SERVICE_NAME,
    version: '1.0.0',
    environment: ENV_MODE,
    timestamp: Time.now.utc.iso8601
  })
end

# Health check
get '/health' do
  checks = {}
  status = 'healthy'
  
  # Check database
  if $db
    begin
      $db.exec('SELECT 1')
      checks[:database] = 'healthy'
    rescue PG::Error => e
      checks[:database] = "unhealthy: #{e.message}"
      status = 'unhealthy'
      # Reconnect
      $db = get_db_connection
    end
  else
    checks[:database] = 'not configured'
  end
  
  # Check Redis
  if $redis
    begin
      $redis.ping
      checks[:redis] = 'healthy'
    rescue Redis::BaseError
      checks[:redis] = 'unhealthy'
      status = 'degraded' if status == 'healthy'
      # Reconnect
      $redis = get_redis_connection
    end
  else
    checks[:redis] = 'not configured'
  end
  
  response_code = status == 'healthy' ? 200 : 503
  
  status response_code
  json({
    status: status,
    service: SERVICE_NAME,
    timestamp: Time.now.utc.iso8601,
    checks: checks
  })
end

# Status endpoint
get '/status' do
  db_connected = false
  redis_connected = false
  
  if $db
    begin
      $db.exec('SELECT 1')
      db_connected = true
    rescue
      db_connected = false
    end
  end
  
  if $redis
    begin
      $redis.ping
      redis_connected = true
    rescue
      redis_connected = false
    end
  end
  
  memory = `ps -o rss= -p #{Process.pid}`.to_i / 1024 # MB
  
  json({
    service: SERVICE_NAME,
    status: 'running',
    uptime: Time.now - $start_time,
    memory_mb: memory,
    environment: {
      ruby: RUBY_VERSION,
      env: ENV_MODE,
      platform: RUBY_PLATFORM
    },
    connections: {
      database: db_connected,
      redis: redis_connected
    },
    timestamp: Time.now.utc.iso8601
  })
end

# Example API endpoint
get '/api/example' do
  data = {}
  
  # Database query
  if $db
    begin
      result = $db.exec('SELECT NOW() as current_time, current_database() as database')
      data = result[0]
    rescue PG::Error => e
      halt 500, json({ error: e.message })
    end
  else
    data[:message] = 'Database not connected'
  end
  
  # Redis cache example
  cached = nil
  if $redis
    begin
      key = 'last_request'
      value = Time.now.utc.iso8601
      $redis.setex(key, 60, value)
      cached = $redis.get(key)
    rescue Redis::BaseError
      # Ignore Redis errors
    end
  end
  
  response = {
    data: data,
    source: 'live'
  }
  response[:cached] = cached if cached
  
  json(response)
end

# Error handling
error do
  status 500
  json({
    error: 'Internal server error',
    message: env['sinatra.error'].message
  })
end

not_found do
  status 404
  json({
    error: 'Not found',
    path: request.path_info
  })
end

# Graceful shutdown
at_exit do
  puts 'Shutting down...'
  $db&.close
  $redis&.close
  puts 'Connections closed'
end

# Start message
puts "#{SERVICE_NAME} listening on port #{PORT}"
puts "Environment: #{ENV_MODE}"
puts "Health check: http://localhost:#{PORT}/health"