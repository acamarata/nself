import os
import asyncio
from datetime import datetime
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import asyncpg
import redis.asyncio as redis
import uvicorn

# Configuration
SERVICE_NAME = os.getenv('SERVICE_NAME', 'py-service')
PORT = int(os.getenv('PORT', 8000))
ENV = os.getenv('ENV', 'development')

# Database configuration
DATABASE_URL = os.getenv('DATABASE_URL') or \
    f"postgresql://{os.getenv('POSTGRES_USER', 'postgres')}:" \
    f"{os.getenv('POSTGRES_PASSWORD', 'postgres')}@" \
    f"{os.getenv('POSTGRES_HOST', 'postgres')}:5432/" \
    f"{os.getenv('POSTGRES_DB', 'nself')}"

# Redis configuration
REDIS_URL = os.getenv('REDIS_URL') or \
    f"redis://{os.getenv('REDIS_HOST', 'redis')}:6379"
REDIS_ENABLED = os.getenv('REDIS_ENABLED', 'false').lower() == 'true'

# Global connections
db_pool: Optional[asyncpg.Pool] = None
redis_client: Optional[redis.Redis] = None

# Pydantic models
class HealthResponse(BaseModel):
    status: str
    service: str
    timestamp: str
    checks: dict

class StatusResponse(BaseModel):
    service: str
    status: str
    uptime: float
    environment: dict
    connections: dict
    timestamp: str

class ExampleResponse(BaseModel):
    data: dict
    source: str
    cached: Optional[str] = None

# Lifespan context manager
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    global db_pool, redis_client
    
    print(f"Starting {SERVICE_NAME}...")
    
    # Initialize database connection
    try:
        db_pool = await asyncpg.create_pool(DATABASE_URL, min_size=1, max_size=10)
        print("Database connected successfully")
    except Exception as e:
        print(f"Database connection failed: {e}")
    
    # Initialize Redis connection
    if REDIS_ENABLED:
        try:
            redis_client = await redis.from_url(REDIS_URL)
            await redis_client.ping()
            print("Redis connected successfully")
        except Exception as e:
            print(f"Redis connection failed: {e}")
            redis_client = None
    
    yield
    
    # Shutdown
    print(f"Shutting down {SERVICE_NAME}...")
    
    if db_pool:
        await db_pool.close()
        print("Database connections closed")
    
    if redis_client:
        await redis_client.close()
        print("Redis connection closed")

# Create FastAPI app
app = FastAPI(
    title=f"{SERVICE_NAME} API",
    description=f"Python FastAPI service for {os.getenv('PROJECT_NAME', 'nself')}",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Health check endpoint
@app.get("/health", response_model=HealthResponse)
async def health_check():
    checks = {}
    overall_status = "healthy"
    
    # Check database
    if db_pool:
        try:
            async with db_pool.acquire() as conn:
                await conn.fetchval("SELECT 1")
            checks["database"] = "healthy"
        except Exception as e:
            checks["database"] = f"unhealthy: {str(e)}"
            overall_status = "unhealthy"
    else:
        checks["database"] = "not configured"
    
    # Check Redis
    if redis_client:
        try:
            await redis_client.ping()
            checks["redis"] = "healthy"
        except Exception:
            checks["redis"] = "unhealthy"
            if overall_status == "healthy":
                overall_status = "degraded"
    else:
        checks["redis"] = "not configured"
    
    if overall_status != "healthy":
        raise HTTPException(status_code=503, detail=checks)
    
    return HealthResponse(
        status=overall_status,
        service=SERVICE_NAME,
        timestamp=datetime.utcnow().isoformat(),
        checks=checks
    )

# Root endpoint
@app.get("/")
async def root():
    return {
        "message": f"Hello from {SERVICE_NAME}! 🚀",
        "service": SERVICE_NAME,
        "version": "1.0.0",
        "environment": ENV,
        "timestamp": datetime.utcnow().isoformat()
    }

# Status endpoint
@app.get("/status", response_model=StatusResponse)
async def status():
    import sys
    import platform
    
    # Get uptime (approximation using asyncio)
    uptime = asyncio.get_event_loop().time()
    
    # Check connections
    connections = {}
    
    if db_pool:
        try:
            async with db_pool.acquire() as conn:
                await conn.fetchval("SELECT 1")
            connections["database"] = True
        except:
            connections["database"] = False
    else:
        connections["database"] = False
    
    if redis_client:
        try:
            await redis_client.ping()
            connections["redis"] = True
        except:
            connections["redis"] = False
    else:
        connections["redis"] = False
    
    return StatusResponse(
        service=SERVICE_NAME,
        status="running",
        uptime=uptime,
        environment={
            "python": sys.version,
            "platform": platform.platform(),
            "env": ENV
        },
        connections=connections,
        timestamp=datetime.utcnow().isoformat()
    )

# Example API endpoint
@app.get("/api/example", response_model=ExampleResponse)
async def example_endpoint():
    try:
        # Database query
        if db_pool:
            async with db_pool.acquire() as conn:
                result = await conn.fetchrow(
                    "SELECT NOW() as current_time, current_database() as database"
                )
                data = dict(result)
                # Convert datetime to string for JSON serialization
                data['current_time'] = str(data['current_time'])
        else:
            data = {"message": "Database not connected"}
        
        # Redis cache example
        cached_value = None
        if redis_client:
            cache_key = "last_request"
            timestamp = datetime.utcnow().isoformat()
            await redis_client.set(cache_key, timestamp, ex=60)
            cached_value = await redis_client.get(cache_key)
            if cached_value:
                cached_value = cached_value.decode('utf-8')
        
        return ExampleResponse(
            data=data,
            source="live",
            cached=cached_value
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

# Error handler
@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    return {
        "error": "Internal server error",
        "message": str(exc),
        "path": request.url.path
    }

# Run with uvicorn when executed directly
if __name__ == "__main__":
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=PORT,
        log_level="info" if ENV == "production" else "debug"
    )