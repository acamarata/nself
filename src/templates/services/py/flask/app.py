import os
from datetime import datetime
from flask import Flask, jsonify, request
from flask_cors import CORS
import psycopg2
from psycopg2.extras import RealDictCursor
import redis
import json

# Configuration
SERVICE_NAME = os.getenv('SERVICE_NAME', 'flask-service')
PORT = int(os.getenv('PORT', 5000))
ENV = os.getenv('ENV', 'development')

# Create Flask app
app = Flask(__name__)
CORS(app)

# Database connection
def get_db():
    try:
        conn = psycopg2.connect(
            host=os.getenv('POSTGRES_HOST', 'postgres'),
            port=5432,
            database=os.getenv('POSTGRES_DB', 'nself'),
            user=os.getenv('POSTGRES_USER', 'postgres'),
            password=os.getenv('POSTGRES_PASSWORD', 'postgres'),
            cursor_factory=RealDictCursor
        )
        return conn
    except Exception as e:
        print(f"Database connection error: {e}")
        return None

# Redis connection
def get_redis():
    if os.getenv('REDIS_ENABLED', 'false').lower() != 'true':
        return None
    try:
        r = redis.Redis(
            host=os.getenv('REDIS_HOST', 'redis'),
            port=6379,
            decode_responses=True
        )
        r.ping()
        return r
    except Exception as e:
        print(f"Redis connection error: {e}")
        return None

db = get_db()
redis_client = get_redis()
start_time = datetime.utcnow()

@app.route('/')
def root():
    return jsonify({
        'message': f'Hello from {SERVICE_NAME}! 🚀',
        'service': SERVICE_NAME,
        'version': '1.0.0',
        'environment': ENV,
        'timestamp': datetime.utcnow().isoformat()
    })

@app.route('/health')
def health():
    checks = {}
    status = 'healthy'
    
    # Check database
    if db:
        try:
            with db.cursor() as cur:
                cur.execute('SELECT 1')
                checks['database'] = 'healthy'
        except Exception as e:
            checks['database'] = f'unhealthy: {str(e)}'
            status = 'unhealthy'
    else:
        checks['database'] = 'not configured'
    
    # Check Redis  
    if redis_client:
        try:
            redis_client.ping()
            checks['redis'] = 'healthy'
        except Exception:
            checks['redis'] = 'unhealthy'
            if status == 'healthy':
                status = 'degraded'
    else:
        checks['redis'] = 'not configured'
    
    response = jsonify({
        'status': status,
        'service': SERVICE_NAME,
        'timestamp': datetime.utcnow().isoformat(),
        'checks': checks
    })
    
    if status != 'healthy':
        response.status_code = 503
    
    return response

@app.route('/status')
def status():
    uptime = (datetime.utcnow() - start_time).total_seconds()
    
    db_connected = False
    if db:
        try:
            with db.cursor() as cur:
                cur.execute('SELECT 1')
            db_connected = True
        except:
            pass
    
    redis_connected = False
    if redis_client:
        try:
            redis_client.ping()
            redis_connected = True
        except:
            pass
    
    return jsonify({
        'service': SERVICE_NAME,
        'status': 'running',
        'uptime': uptime,
        'environment': {
            'python': os.sys.version,
            'env': ENV
        },
        'connections': {
            'database': db_connected,
            'redis': redis_connected
        },
        'timestamp': datetime.utcnow().isoformat()
    })

@app.route('/api/example')
def example():
    data = {}
    
    # Database query
    if db:
        try:
            with db.cursor() as cur:
                cur.execute('SELECT NOW() as current_time, current_database() as database')
                result = cur.fetchone()
                data = dict(result)
                data['current_time'] = str(data['current_time'])
        except Exception as e:
            return jsonify({'error': str(e)}), 500
    else:
        data = {'message': 'Database not connected'}
    
    # Redis cache
    cached = None
    if redis_client:
        try:
            key = 'last_request'
            value = datetime.utcnow().isoformat()
            redis_client.setex(key, 60, value)
            cached = redis_client.get(key)
        except:
            pass
    
    response = {
        'data': data,
        'source': 'live'
    }
    
    if cached:
        response['cached'] = cached
    
    return jsonify(response)

@app.errorhandler(404)
def not_found(e):
    return jsonify({
        'error': 'Not found',
        'path': request.path
    }), 404

@app.errorhandler(500)
def server_error(e):
    return jsonify({
        'error': 'Internal server error',
        'message': str(e)
    }), 500

if __name__ == '__main__':
    print(f'{SERVICE_NAME} listening on port {PORT}')
    print(f'Environment: {ENV}')
    print(f'Health check: http://localhost:{PORT}/health')
    app.run(host='0.0.0.0', port=PORT, debug=(ENV != 'production'))