package jobs

import (
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"context"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"net/http"
	"bytes"
)

const TypeLLMPrompt = "llm:prompt"

type LLMPayload struct {
    JobID  int64  `json:"job_id"`
    Prompt string `json:"prompt"`
}

type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
}

type OllamaResponse struct {
    Response string `json:"response"`
    Done     bool   `json:"done"`
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
func CallOllama(prompt string) (string, error) {
    body, err := json.Marshal(OllamaRequest{
        Model:  "llama3.2",
        Prompt: prompt,
        Stream: false,
    })
    if err != nil {
        return "", err
    }

    resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(body))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var ollamaResp OllamaResponse
    if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
        return "", err
    }

    return ollamaResp.Response, nil
}

func HandleLLMTask(ctx context.Context, t *asynq.Task) error {
	var payload LLMPayload
	err := json.Unmarshal(t.Payload(), &payload)
	// If payload is invalid, skip retrying and mark the job as failed
	if err != nil {
		return fmt.Errorf("invalid payload, skipping retry: %w", asynq.SkipRetry)
	}

	// Update job to running status
	if err := db.UpdateJobRunning(payload.JobID); err != nil {
		return err
	}

	// Call Ollama and get the response
	response, err := CallOllama(payload.Prompt)
	if err != nil {
		if dbErr := db.UpdateJobFailed(payload.JobID, err.Error()); dbErr != nil {
			fmt.Printf("failed to update job status: %v\n", dbErr)
		}
		return err
	}

	// Update job to completed status with the LLM response
	if err := db.UpdateJobFinished(payload.JobID, response); err != nil {
		if dbErr := db.UpdateJobFailed(payload.JobID, err.Error()); dbErr != nil {
			fmt.Printf("failed to update job status: %v\n", dbErr)
		}
		return err
	}

	return nil
}




