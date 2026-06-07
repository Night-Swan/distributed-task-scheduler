package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/Night-Swan/distributed-task-scheduler/internal/api"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"github.com/Night-Swan/distributed-task-scheduler/internal/jobs"
)

func main() {
	// Connect to the database
	if err := db.Connect(); err != nil {
		panic(err)
	}

	// Create Asynq client and server
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
	asynqServer := asynq.NewServer(asynq.RedisClientOpt{Addr: "localhost:6379"}, asynq.Config{})

	// Create API handler with Asynq client
	handler := &api.Handler{AsynqClient: asynqClient}

	// Start Asynq server in a separate goroutine
	mux := asynq.NewServeMux()
	mux.HandleFunc(jobs.TypeLLMPrompt, jobs.HandleLLMTask)

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

	// Start the HTTP server
	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}

