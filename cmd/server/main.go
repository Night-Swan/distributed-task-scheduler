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
	"fmt"
)

func main() {
	// Connect to the database
	if err := db.Connect(); err != nil {
		panic(err)
	}

	// Create Asynq client and server
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	asynqServer := asynq.NewServer(asynq.RedisClientOpt{Addr: "localhost:6379"}, asynq.Config{
    Concurrency: 10,
    RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
        return time.Duration(5^n) * time.Second
    },
    Queues: map[string]int{
        "default": 1,
    },
	})

	// Create API handler with Asynq client
	handler := &api.Handler{AsynqClient: asynqClient}

	// Start Asynq server in a separate goroutine
	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TypeLLMPrompt, jobs.HandleLLMTask)
	mux.HandleFunc(jobs.TypeTranscription, jobs.HandleTranscriptionTask)
	mux.HandleFunc(jobs.TypeEmbedding, jobs.HandleEmbeddingTask)
	mux.HandleFunc(jobs.TypePDFProcessing, jobs.HandlePDFTask)

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
	fmt.Println("Shutting down...")
	asynqServer.Shutdown()
	db.Pool.Close()
	fmt.Println("Done")
	
}

