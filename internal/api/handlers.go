package api

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"github.com/Night-Swan/distributed-task-scheduler/internal/jobs"
	"strconv"	
	"encoding/json"
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
	
	payload, err := json.Marshal(map[string]string{"prompt": req.Prompt})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to marshal payload"})
		return
	}
	jobID, err := db.CreateJob(req.SubmittedBy, req.JobType, payload)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create job"})
		return
	}

	if req.JobType == jobs.TypeLLMPrompt {
		task, err := jobs.NewLLMTask(jobID, req.Prompt)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create task"})
			return
		}
		if _, err := h.AsynqClient.Enqueue(task); err != nil {
			c.JSON(500, gin.H{"error": "Failed to enqueue task"})
			return
		}
	} else if req.JobType == jobs.TypeEmbedding {
		task, err := jobs.NewEmbeddingTask(jobID, req.Prompt)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create task"})
			return
		}
		if _, err := h.AsynqClient.Enqueue(task); err != nil {
			c.JSON(500, gin.H{"error": "Failed to enqueue task"})
			return
		}
	} else {
		c.JSON(400, gin.H{"error": "Unsupported job type"})
		return
	}
	c.JSON(200, CreateJobResponse{JobID: jobID})

}


func (h *Handler) CreateTranscriptionJob(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "File is required"})
			return
		}
		submittedBy := c.PostForm("submitted_by")
		if submittedBy == "" {
			c.JSON(400, gin.H{"error": "submitted_by is required"})
			return
		}
		// Save the uploaded file to a temporary location
		tempFilePath := "uploads/" + file.Filename
		if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save uploaded file"})
			return
		}

		payload, err := json.Marshal(map[string]string{"file_path": tempFilePath})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to marshal payload"})
			return
		}
		jobID, err := db.CreateJob(submittedBy, "transcription", payload)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create job"})
			return
		}	

		task, err := jobs.NewTranscriptionTask(jobID, tempFilePath)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create transcription task"})
			return
		}
		if _, err := h.AsynqClient.Enqueue(task); err != nil {
			c.JSON(500, gin.H{"error": "Failed to enqueue transcription task"})
			return
		}
		c.JSON(200, CreateJobResponse{JobID: jobID})
}	

func (h *Handler) CreatePDFJob(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": "File is required"})
			return
		}
		submittedBy := c.PostForm("submitted_by")
		if submittedBy == "" {
			c.JSON(400, gin.H{"error": "submitted_by is required"})
			return
		}
		// Save the uploaded file to a temporary location
		tempFilePath := "uploads/" + file.Filename
		if err := c.SaveUploadedFile(file, tempFilePath); err != nil {
			c.JSON(500, gin.H{"error": "Failed to save uploaded file"})
			return
		}

		payload, err := json.Marshal(map[string]string{"file_path": tempFilePath})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to marshal payload"})
			return
		}
		jobID, err := db.CreateJob(submittedBy, "pdf_processing", payload)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create job"})
			return
		}

		task, err := jobs.NewPDFTask(jobID, tempFilePath)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create PDF processing task"})
			return
		}
		if _, err := h.AsynqClient.Enqueue(task); err != nil {
			c.JSON(500, gin.H{"error": "Failed to enqueue PDF processing task"})
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



