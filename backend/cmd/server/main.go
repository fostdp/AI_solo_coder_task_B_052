package main

import (
	"ancient-wood-monitor/config"
	"ancient-wood-monitor/internal/handlers"
	"ancient-wood-monitor/internal/middleware"
	"ancient-wood-monitor/internal/pipeline"
	"ancient-wood-monitor/internal/services"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := getConfigPath()
	if err := config.LoadConfig(configPath); err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		log.Println("Using default configuration")
	}

	gin.SetMode(config.AppConfig.Server.Mode)

	influxDBService, err := services.NewInfluxDBService()
	if err != nil {
		log.Printf("Warning: Failed to connect to InfluxDB: %v", err)
		log.Println("Running in mock mode - some features may not work properly")
	} else {
		defer influxDBService.Close()
		log.Println("Successfully connected to InfluxDB")
	}

	alertService := services.NewAlertService(influxDBService)
	sensorService := services.NewSensorService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	servicePipeline, err := pipeline.NewServicePipeline(config.AppConfig)
	if err != nil {
		log.Fatalf("Failed to create service pipeline: %v", err)
	}
	if err := servicePipeline.Start(ctx); err != nil {
		log.Fatalf("Failed to start pipeline: %v", err)
	}
	defer servicePipeline.Stop()
	log.Println("Service pipeline started (4 stages)")

	handler := handlers.NewHandler(influxDBService, alertService, sensorService, servicePipeline)

	go startPprofServer(ctx)

	r := gin.Default()

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(middleware.PrometheusMiddleware())

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/health", handler.HealthCheck)

	api := r.Group("/api/v1")
	{
		api.POST("/lora/data", handler.ReceiveLoRaData)

		api.GET("/sensors", handler.GetSensors)
		api.GET("/sensors/:id", handler.GetSensor)
		api.GET("/buildings", handler.GetBuildings)

		api.GET("/data/acoustic", handler.GetAcousticData)
		api.GET("/data/moisture", handler.GetMoistureData)

		api.GET("/alerts", handler.GetAlerts)

		api.GET("/risk-zones", handler.GetRiskZones)

		api.GET("/predict/termite", handler.PredictTermiteActivity)

		api.POST("/simulate/fumigation", handler.SimulateFumigation)

		api.GET("/analysis/wavelet", handler.GetWaveletAnalysis)
	}

	frontendPath := getFrontendPath()
	r.StaticFile("/", filepath.Join(frontendPath, "index.html"))
	r.StaticFile("/app.js", filepath.Join(frontendPath, "app.js"))
	r.StaticFile("/TimberModel.js", filepath.Join(frontendPath, "TimberModel.js"))
	r.StaticFile("/VoxelRisk.js", filepath.Join(frontendPath, "VoxelRisk.js"))
	r.Static("/static", frontendPath)

	r.NoRoute(func(c *gin.Context) {
		c.File(filepath.Join(frontendPath, "index.html"))
	})

	addr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		log.Printf("========================================")
		log.Printf("古代木结构建筑虫蛀监测系统")
		log.Printf("服务器启动在 http://localhost%s", addr)
		log.Printf("前端页面: http://localhost%s/", addr)
		log.Printf("API文档:  http://localhost%s/api/v1/", addr)
		log.Printf("Metrics:  http://localhost%s/metrics", addr)
		log.Printf("Pprof:    http://localhost:6060/debug/pprof/")
		log.Printf("========================================")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}

func startPprofServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:    ":6060",
		Handler: mux,
	}

	go func() {
		log.Printf("Pprof server starting on :6060")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Pprof server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func getConfigPath() string {
	if cp := os.Getenv("CONFIG_PATH"); cp != "" {
		return cp
	}
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	
	configPath := filepath.Join(basepath, "..", "..", "config", "config.yaml")
	return configPath
}

func getFrontendPath() string {
	if fp := os.Getenv("FRONTEND_PATH"); fp != "" {
		return fp
	}
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	
	frontendPath := filepath.Join(basepath, "..", "..", "..", "frontend", "public")
	absPath, err := filepath.Abs(frontendPath)
	if err == nil {
		return absPath
	}
	return frontendPath
}
