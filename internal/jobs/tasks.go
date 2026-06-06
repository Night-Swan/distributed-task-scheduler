package jobs

import (
	"encoding/json"
	"github.com/hibiken/asynq"
	"context"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"fmt"
)

const TypeLLMPrompt = "llm:prompt"

type LLMPayload struct {
    JobID  int64  `json:"job_id"`
    Prompt string `json:"prompt"`
}


func NewLLMTask(jobID int64, prompt string) (*asynq.Task, error) {
	// Create a new LLM task payload
	payload := LLMPayload{
		JobID: jobID,
		Prompt: prompt,
	}

	// Convert the payload to JSON
	data, err := json.Marshal(payload)

	if err != nil {
		return nil, err
	}

	// Return and create a new Asynq task with the payload
	return asynq.NewTask(TypeLLMPrompt, data), nil
}


func HandleLLMTask(ctx context.Context, t *asynq.Task) error {
	var payload LLMPayload
	err := json.Unmarshal(t.Payload(), &payload)
	if err != nil {
		return err
	}

	// Update job to running status
	if err := db.UpdateJobRunning(payload.JobID); err != nil {
		return err
	}

	// Placeholder to simulate LLM processing
	fmt.Printf("Processing LLM task for Job ID: %d with prompt: %s\n", payload.JobID, payload.Prompt)

	// Update job to completed status with a dummy result
	if err := db.UpdateJobFinished(payload.JobID, "LLM response here"); err != nil {
		return err
	}

	return nil
}




