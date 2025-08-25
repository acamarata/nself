use actix_web::{middleware, web, App, HttpResponse, HttpServer, Result};
use serde::{Deserialize, Serialize};
use sqlx::{postgres::PgPoolOptions, Pool, Postgres};
use redis::AsyncCommands;
use std::env;
use std::time::{SystemTime, UNIX_EPOCH};
use chrono::{DateTime, Utc};

#[derive(Serialize)]
struct HealthResponse {
    status: String,
    service: String,
    timestamp: String,
    checks: std::collections::HashMap<String, String>,
}

#[derive(Serialize)]
struct StatusResponse {
    service: String,
    status: String,
    uptime: u64,
    environment: Environment,
    connections: Connections,
    timestamp: String,
}

#[derive(Serialize)]
struct Environment {
    rust: String,
    env: String,
    os: String,
}

#[derive(Serialize)]
struct Connections {
    database: bool,
    redis: bool,
}

#[derive(Serialize)]
struct ExampleResponse {
    data: serde_json::Value,
    source: String,
    cached: Option<String>,
}

struct AppState {
    db: Pool<Postgres>,
    redis: Option<redis::aio::ConnectionManager>,
    service_name: String,
    start_time: SystemTime,
}

async fn root(data: web::Data<AppState>) -> Result<HttpResponse> {
    Ok(HttpResponse::Ok().json(serde_json::json!({
        "message": format!("Hello from {}! 🚀", data.service_name),
        "service": data.service_name,
        "version": "1.0.0",
        "environment": env::var("ENV").unwrap_or_else(|_| "development".to_string()),
        "timestamp": Utc::now().to_rfc3339(),
    })))
}

async fn health(data: web::Data<AppState>) -> Result<HttpResponse> {
    let mut checks = std::collections::HashMap::new();
    let mut status = "healthy".to_string();
    
    // Check database
    match sqlx::query("SELECT 1").fetch_one(&data.db).await {
        Ok(_) => checks.insert("database".to_string(), "healthy".to_string()),
        Err(e) => {
            checks.insert("database".to_string(), format!("unhealthy: {}", e));
            status = "unhealthy".to_string();
        }
    };
    
    // Check Redis
    if let Some(ref redis_conn) = data.redis {
        let mut conn = redis_conn.clone();
        match redis::cmd("PING").query_async::<_, String>(&mut conn).await {
            Ok(_) => checks.insert("redis".to_string(), "healthy".to_string()),
            Err(_) => {
                checks.insert("redis".to_string(), "unhealthy".to_string());
                if status == "healthy" {
                    status = "degraded".to_string();
                }
            }
        };
    } else {
        checks.insert("redis".to_string(), "not configured".to_string());
    }
    
    let response = HealthResponse {
        status: status.clone(),
        service: data.service_name.clone(),
        timestamp: Utc::now().to_rfc3339(),
        checks,
    };
    
    if status == "healthy" {
        Ok(HttpResponse::Ok().json(response))
    } else {
        Ok(HttpResponse::ServiceUnavailable().json(response))
    }
}

async fn status(data: web::Data<AppState>) -> Result<HttpResponse> {
    let uptime = SystemTime::now()
        .duration_since(data.start_time)
        .unwrap_or_default()
        .as_secs();
    
    let db_connected = sqlx::query("SELECT 1")
        .fetch_one(&data.db)
        .await
        .is_ok();
    
    let mut redis_connected = false;
    if let Some(ref redis_conn) = data.redis {
        let mut conn = redis_conn.clone();
        redis_connected = redis::cmd("PING")
            .query_async::<_, String>(&mut conn)
            .await
            .is_ok();
    }
    
    let response = StatusResponse {
        service: data.service_name.clone(),
        status: "running".to_string(),
        uptime,
        environment: Environment {
            rust: env!("CARGO_PKG_VERSION").to_string(),
            env: env::var("ENV").unwrap_or_else(|_| "development".to_string()),
            os: std::env::consts::OS.to_string(),
        },
        connections: Connections {
            database: db_connected,
            redis: redis_connected,
        },
        timestamp: Utc::now().to_rfc3339(),
    };
    
    Ok(HttpResponse::Ok().json(response))
}

async fn example(data: web::Data<AppState>) -> Result<HttpResponse> {
    // Database query
    let db_result = sqlx::query!("SELECT NOW() as current_time, current_database() as database")
        .fetch_one(&data.db)
        .await;
    
    let data_json = match db_result {
        Ok(row) => serde_json::json!({
            "current_time": row.current_time.map(|t| t.to_rfc3339()),
            "database": row.database,
        }),
        Err(e) => serde_json::json!({
            "error": format!("Database error: {}", e)
        }),
    };
    
    // Redis cache example
    let mut cached = None;
    if let Some(ref redis_conn) = data.redis {
        let mut conn = redis_conn.clone();
        let key = "last_request";
        let value = Utc::now().to_rfc3339();
        
        let _: Result<(), _> = conn.set_ex(key, &value, 60).await;
        cached = conn.get(key).await.ok();
    }
    
    let response = ExampleResponse {
        data: data_json,
        source: "live".to_string(),
        cached,
    };
    
    Ok(HttpResponse::Ok().json(response))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));
    
    let service_name = env::var("SERVICE_NAME").unwrap_or_else(|_| "rs-service".to_string());
    let port = env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let bind_address = format!("0.0.0.0:{}", port);
    
    // Database connection
    let database_url = env::var("DATABASE_URL").unwrap_or_else(|_| {
        format!(
            "postgresql://{}:{}@{}/{}",
            env::var("POSTGRES_USER").unwrap_or_else(|_| "postgres".to_string()),
            env::var("POSTGRES_PASSWORD").unwrap_or_else(|_| "postgres".to_string()),
            env::var("POSTGRES_HOST").unwrap_or_else(|_| "postgres".to_string()),
            env::var("POSTGRES_DB").unwrap_or_else(|_| "nself".to_string()),
        )
    });
    
    let db = PgPoolOptions::new()
        .max_connections(5)
        .connect(&database_url)
        .await
        .expect("Failed to connect to database");
    
    // Redis connection
    let redis_conn = if env::var("REDIS_ENABLED").unwrap_or_else(|_| "false".to_string()) == "true" {
        let redis_url = env::var("REDIS_URL")
            .unwrap_or_else(|_| format!("redis://{}", env::var("REDIS_HOST").unwrap_or_else(|_| "redis".to_string())));
        
        match redis::Client::open(redis_url) {
            Ok(client) => match redis::aio::ConnectionManager::new(client).await {
                Ok(conn) => Some(conn),
                Err(e) => {
                    eprintln!("Redis connection error: {}", e);
                    None
                }
            },
            Err(e) => {
                eprintln!("Redis client error: {}", e);
                None
            }
        }
    } else {
        None
    };
    
    let app_state = web::Data::new(AppState {
        db,
        redis: redis_conn,
        service_name: service_name.clone(),
        start_time: SystemTime::now(),
    });
    
    println!("{} listening on {}", service_name, bind_address);
    println!("Environment: {}", env::var("ENV").unwrap_or_else(|_| "development".to_string()));
    println!("Health check: http://localhost:{}/health", port);
    
    HttpServer::new(move || {
        App::new()
            .app_data(app_state.clone())
            .wrap(middleware::Logger::default())
            .wrap(
                middleware::DefaultHeaders::new()
                    .add(("Access-Control-Allow-Origin", "*"))
                    .add(("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS"))
                    .add(("Access-Control-Allow-Headers", "Content-Type, Authorization"))
            )
            .route("/", web::get().to(root))
            .route("/health", web::get().to(health))
            .route("/status", web::get().to(status))
            .route("/api/example", web::get().to(example))
    })
    .bind(&bind_address)?
    .run()
    .await
}