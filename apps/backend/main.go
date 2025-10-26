package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger         *zap.Logger
	tracer         trace.Tracer
	meter          metric.Meter
	requestCounter metric.Int64Counter
	requestDuration metric.Float64Histogram
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	// Logger initialization
	initLogger()
	defer logger.Sync()

	// OpenTelemetry initialization
	ctx := context.Background()
	shutdown := initOpenTelemetry(ctx)
	defer shutdown(ctx)

	// Gin router setup
	router := setupRouter()

	// Graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		logger.Info("Starting backend server", zap.String("port", "8080"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = ""

	var err error
	logger, err = config.Build()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}

func initOpenTelemetry(ctx context.Context) func(context.Context) error {
	// Get OTel Collector endpoint
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "otel-collector.observability.svc.cluster.local:4317"
	}

	// Resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("backend"),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironment("development"),
			attribute.String("application", "backend-api"),
		),
	)
	if err != nil {
		logger.Fatal("Failed to create resource", zap.Error(err))
	}

	// Trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		logger.Fatal("Failed to create trace exporter", zap.Error(err))
	}

	// Trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Metric exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otelEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		logger.Fatal("Failed to create metric exporter", zap.Error(err))
	}

	// Metric provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	// Tracer and Meter
	tracer = otel.Tracer("backend-tracer")
	meter = otel.Meter("backend-meter")

	// Custom metrics
	requestCounter, err = meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Number of HTTP requests"),
		metric.WithUnit("{requests}"),
	)
	if err != nil {
		logger.Fatal("Failed to create request counter", zap.Error(err))
	}

	requestDuration, err = meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("Duration of HTTP requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		logger.Fatal("Failed to create request duration histogram", zap.Error(err))
	}

	return func(ctx context.Context) error {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			return err
		}
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	}
}

func setupRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware("backend"))
	router.Use(loggingMiddleware())
	router.Use(metricsMiddleware())

	// Routes
	router.GET("/health", healthHandler)
	router.GET("/api/users", getUsersHandler)
	router.GET("/api/users/:id", getUserHandler)
	router.POST("/api/process", processHandler)

	return router
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Milliseconds()

		// Record metrics
		attrs := []attribute.KeyValue{
			attribute.String("method", c.Request.Method),
			attribute.String("path", c.FullPath()),
			attribute.Int("status", c.Writer.Status()),
		}

		requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
		requestDuration.Record(c.Request.Context(), float64(duration), metric.WithAttributes(attrs...))
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "backend",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func getUsersHandler(c *gin.Context) {
	ctx := c.Request.Context()
	_, span := tracer.Start(ctx, "getUsersHandler")
	defer span.End()

	// Simulate database query
	users := fetchUsers(ctx)

	span.SetAttributes(attribute.Int("user_count", len(users)))

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

func getUserHandler(c *gin.Context) {
	ctx := c.Request.Context()
	_, span := tracer.Start(ctx, "getUserHandler")
	defer span.End()

	id := c.Param("id")
	span.SetAttributes(attribute.String("user_id", id))

	// Simulate database query with random delay
	user := fetchUserByID(ctx, id)

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func processHandler(c *gin.Context) {
	ctx := c.Request.Context()
	_, span := tracer.Start(ctx, "processHandler")
	defer span.End()

	// Simulate heavy processing
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	// Random error simulation (5% chance)
	if rand.Float32() < 0.05 {
		span.SetAttributes(attribute.Bool("error", true))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Processing failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Processing completed",
		"status":  "success",
	})
}

func fetchUsers(ctx context.Context) []User {
	_, span := tracer.Start(ctx, "fetchUsers")
	defer span.End()

	// Simulate database delay
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now().Add(-24 * time.Hour)},
		{ID: 2, Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)},
		{ID: 3, Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now().Add(-72 * time.Hour)},
	}

	span.SetAttributes(attribute.Int("db.rows_returned", len(users)))
	return users
}

func fetchUserByID(ctx context.Context, id string) *User {
	_, span := tracer.Start(ctx, "fetchUserByID")
	defer span.End()

	span.SetAttributes(attribute.String("db.query_id", id))

	// Simulate database delay
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	// Simple mock data
	users := map[string]*User{
		"1": {ID: 1, Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now()},
		"2": {ID: 2, Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now()},
		"3": {ID: 3, Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now()},
	}

	return users[id]
}
