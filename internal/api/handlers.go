package api

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"github.com/Night-Swan/distributed-task-scheduler/internal/jobs"
	"strconv"	
)

type CreateJobRequest struct {
	JobType string `json:"job_type"`
	Prompt string `json:"prompt"`
	SubmittedBy string `json:"submitted_by"`
}

type CreateJobResponse struct {
	JobID int64 `json:"job_id"`
}

// Async client for enqueuing tasks into redis
type Handler struct {
    AsynqClient *asynq.Client
}

func (h *Handler) CreateJob(c *gin.Context) {
    var req CreateJobRequest
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
	
	jobID, err := db.CreateJob(req.SubmittedBy, req.JobType, []byte(req.Prompt))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create job"})
		return
	}
	task, err := jobs.NewLLMTask(jobID, req.Prompt)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create task"})
		return
	}
	if _, err := h.AsynqClient.Enqueue(task); err != nil {
		c.JSON(500, gin.H{"error": "Failed to enqueue task"})
		return
	}
	c.JSON(200, CreateJobResponse{JobID: jobID})

}

func (h *Handler) GetJob(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid job ID"})
		return
	}
	job, err := db.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Job not found"})
		return
	}
	c.JSON(200, job)
}



