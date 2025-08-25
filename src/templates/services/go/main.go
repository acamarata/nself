package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	_ "github.com/lib/pq"
)

var (
	db          *sql.DB
	redisClient *redis.Client
	ctx         = context.Background()
	serviceName string
	startTime   time.Time
)

func init() {
	startTime = time.Now()
	serviceName = getEnv("SERVICE_NAME", "go-service")
	
	// Initialize database
	dbURL := getEnv("DATABASE_URL", fmt.Sprintf(
		"postgresql://%s:%s@%s:5432/%s?sslmode=disable",
		getEnv("POSTGRES_USER", "postgres"),
		getEnv("POSTGRES_PASSWORD", "postgres"),
		getEnv("POSTGRES_HOST", "postgres"),
		getEnv("POSTGRES_DB", "nself"),
	))
	
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Database connection error: %v", err)
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		
		if err := db.Ping(); err != nil {
			log.Printf("Database ping failed: %v", err)
		} else {
			log.Println("Database connected successfully")
		}
	}
	
	// Initialize Redis
	if getEnv("REDIS_ENABLED", "false") == "true" {
		redisURL := getEnv("REDIS_URL", fmt.Sprintf("redis://%s:6379", getEnv("REDIS_HOST", "redis")))
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			redisClient = redis.NewClient(opt)
			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("Redis connection error: %v", err)
				redisClient = nil
			} else {
				log.Println("Redis connected successfully")
			}
		}
	}
}

func main() {
	// Set Gin mode
	if getEnv("ENV", "development") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	router.Use(corsMiddleware())
	
	// Routes
	router.GET("/", rootHandler)
	router.GET("/health", healthHandler)
	router.GET("/status", statusHandler)
	router.GET("/api/example", exampleHandler)
	
	// Server configuration
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	
	// Start server in goroutine
	go func() {
		log.Printf("%s listening on port %s", serviceName, port)
		log.Printf("Environment: %s", getEnv("ENV", "development"))
		log.Printf("Health check: http://localhost:%s/health", port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
	
	// Close connections
	if db != nil {
		db.Close()
		log.Println("Database connections closed")
	}
	
	if redisClient != nil {
		redisClient.Close()
		log.Println("Redis connection closed")
	}
	
	log.Println("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

func rootHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":     fmt.Sprintf("Hello from %s! 🚀", serviceName),
		"service":     serviceName,
		"version":     "1.0.0",
		"environment": getEnv("ENV", "development"),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	})
}

func healthHandler(c *gin.Context) {
	checks := make(map[string]string)
	status := "healthy"
	
	// Check database
	if db != nil {
		if err := db.Ping(); err != nil {
			checks["database"] = fmt.Sprintf("unhealthy: %v", err)
			status = "unhealthy"
		} else {
			checks["database"] = "healthy"
		}
	} else {
		checks["database"] = "not configured"
	}
	
	// Check Redis
	if redisClient != nil {
		if err := redisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unhealthy"
			if status == "healthy" {
				status = "degraded"
			}
		} else {
			checks["redis"] = "healthy"
		}
	} else {
		checks["redis"] = "not configured"
	}
	
	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, gin.H{
		"status":    status,
		"service":   serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    checks,
	})
}

func statusHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	dbConnected := false
	if db != nil {
		dbConnected = db.Ping() == nil
	}
	
	redisConnected := false
	if redisClient != nil {
		redisConnected = redisClient.Ping(ctx).Err() == nil
	}
	
	c.JSON(http.StatusOK, gin.H{
		"service": serviceName,
		"status":  "running",
		"uptime":  time.Since(startTime).Seconds(),
		"memory": gin.H{
			"alloc":      m.Alloc / 1024 / 1024,      // MB
			"totalAlloc": m.TotalAlloc / 1024 / 1024, // MB
			"sys":        m.Sys / 1024 / 1024,        // MB
			"numGC":      m.NumGC,
		},
		"environment": gin.H{
			"go":      runtime.Version(),
			"env":     getEnv("ENV", "development"),
			"numCPU":  runtime.NumCPU(),
			"os":      runtime.GOOS,
			"arch":    runtime.GOARCH,
		},
		"connections": gin.H{
			"database": dbConnected,
			"redis":    redisConnected,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func exampleHandler(c *gin.Context) {
	data := make(map[string]interface{})
	
	// Database query
	if db != nil {
		var currentTime time.Time
		var database string
		err := db.QueryRow("SELECT NOW(), current_database()").Scan(&currentTime, &database)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		data["current_time"] = currentTime.Format(time.RFC3339)
		data["database"] = database
	} else {
		data["message"] = "Database not connected"
	}
	
	// Redis cache example
	var cached string
	if redisClient != nil {
		key := "last_request"
		value := time.Now().UTC().Format(time.RFC3339)
		redisClient.Set(ctx, key, value, 60*time.Second)
		cached, _ = redisClient.Get(ctx, key).Result()
	}
	
	response := gin.H{
		"data":   data,
		"source": "live",
	}
	
	if cached != "" {
		response["cached"] = cached
	}
	
	c.JSON(http.StatusOK, response)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}