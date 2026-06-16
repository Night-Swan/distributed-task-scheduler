package api

import (
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"encoding/json"
	"testing"
	"net/http/httptest"
	"bytes"
	"strconv"
)

func setupRouter() *gin.Engine {
    asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})
    inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: "localhost:6379"})
    handler := &Handler{
        AsynqClient:    asynqClient,
        AsynqInspector: inspector,
    }
    router := gin.Default()
    router.POST("/jobs", handler.CreateJob)
    router.GET("/jobs/:id", handler.GetJob)
    return router
}

func TestCreateJob(t *testing.T) {

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	router := setupRouter()

	body := `{"job_type": "llm_prompt", "prompt": "test prompt", "submitted_by": "test", "priority": "default"}`

	req := httptest.NewRequest("POST", "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder() // Create a response recorder
	router.ServeHTTP(w, req) // Send it through the router

	if w.Code != 200 {
		t.Fatalf("Expected status code 200, got %d", w.Code)
	}

	var resp CreateJobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.JobID == 0 {
		t.Fatal("Expected non-zero job ID")
	}
}

func TestGetJob(t *testing.T) {

	if err := db.Connect(); err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	router := setupRouter()

	body := `{"job_type": "llm_prompt", "prompt": "test prompt", "submitted_by": "test", "priority": "default"}`
	req := httptest.NewRequest("POST", "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected status code 200, got %d", w.Code)
	}

	var createResp CreateJobResponse
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("Failed to unmarshal create response: %v", err)
	}

	getReq := httptest.NewRequest("GET", "/jobs/"+strconv.FormatInt(createResp.JobID, 10), nil) // Get job by ID
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	if getW.Code != 200 {
		t.Fatalf("Expected status code 200, got %d", getW.Code)
	}

	var job db.Job
	if err := json.Unmarshal(getW.Body.Bytes(), &job); err != nil {
		t.Fatalf("Failed to unmarshal get response: %v", err)
	}

	if job.ID != createResp.JobID {
		t.Fatalf("Expected job ID %d, got %d", createResp.JobID, job.ID)
	}
}