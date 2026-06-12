package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/Night-Swan/distributed-task-scheduler/internal/api"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"github.com/Night-Swan/distributed-task-scheduler/internal/jobs"
	"time"
	"os"
	"os/signal"
	"syscall"
	"log/slog"
)

func main() {

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	
	// Connect to the database
	if err := db.Connect(); err != nil {
		panic(err)
	}

	slog.Info("database connected")

	// Create Asynq client and server
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	asynqServer := asynq.NewServer(asynq.RedisClientOpt{Addr: "localhost:6379"}, asynq.Config{
		Concurrency: 10,
		RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
			return time.Duration(5^n) * time.Second
		},
		Queues: map[string]int{
			"critical": 3,
			"default":  2,
			"low":      1,
		},
	})

	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: "localhost:6379"})
	// Create API handler with Asynq client
	handler := &api.Handler{AsynqClient: asynqClient, AsynqInspector: inspector}

	// Start Asynq server in a separate goroutine
	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TypeLLMPrompt, jobs.HandleLLMTask)
	mux.HandleFunc(jobs.TypeTranscription, jobs.HandleTranscriptionTask)
	mux.HandleFunc(jobs.TypeEmbedding, jobs.HandleEmbeddingTask)
	mux.HandleFunc(jobs.TypePDFProcessing, jobs.HandlePDFTask)
	slog.Info("worker started", "concurrency", 10)

	// Run Asynq server in background with Goroutine concurrency
	go func() {
		if err := asynqServer.Run(mux); err != nil {
			panic(err)
		}
	}()
	// Set up Gin router and API routes
	router := gin.Default()
	router.POST("/jobs", handler.CreateJob)
	router.GET("/jobs/:id", handler.GetJob)
	router.POST("/jobs/transcription", handler.CreateTranscriptionJob)
	router.POST("/jobs/pdf", handler.CreatePDFJob)
	router.GET("/metrics", handler.GetMetrics)
	slog.Info("API server starting", "port", "8080")

	// Start the HTTP server
	go func() {
		if err := router.Run(":8080"); err != nil {
			panic(err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Shutdown cleanly
	slog.Info("Shutting down...")
	asynqServer.Shutdown()
	db.Pool.Close()
	slog.Info("Done")
	
}

