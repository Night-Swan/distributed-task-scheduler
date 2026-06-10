package jobs

import (
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
	"context"
	"github.com/Night-Swan/distributed-task-scheduler/internal/db"
	"net/http"
	"bytes"
	"os"
	"io"
	"mime/multipart"
	"path/filepath"
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

const TypeTranscription = "transcription:audio"

type TranscriptionPayload struct {
    JobID    int64  `json:"job_id"`
    FilePath string `json:"file_path"`
}

type WhisperResponse struct {
    Text string `json:"text"`
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

func CallWhisper(filePath string) (string, error) {
	// Create a buffer and multipart writer
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the model field
	writer.WriteField("model", "Systran/faster-whisper-small")

	// Open the audio file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Add the file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}		
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}

	// Close the writer — important, finalizes the multipart body
	writer.Close()

	// Send the request
	resp, err := http.Post("http://localhost:8000/v1/audio/transcriptions", writer.FormDataContentType(), &buf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Parse the response
	var transcriptionResp WhisperResponse
	if err := json.NewDecoder(resp.Body).Decode(&transcriptionResp); err != nil {
		return "", err
	}	

	return transcriptionResp.Text, nil
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


func NewTranscriptionTask(jobID int64, filePath string) (*asynq.Task, error) {
    payload := TranscriptionPayload{
        JobID:    jobID,
        FilePath: filePath,
    }
    data, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }
    return asynq.NewTask(TypeTranscription, data), nil
}	

func HandleTranscriptionTask(ctx context.Context, t *asynq.Task) error {
	var payload TranscriptionPayload
	err := json.Unmarshal(t.Payload(), &payload)	
	if err != nil {
		return fmt.Errorf("invalid payload, skipping retry: %w", asynq.SkipRetry)
	}

	if err := db.UpdateJobRunning(payload.JobID); err != nil {
		return err
	}

	//placeholder for actual transcription logic, replace with real implementation
	transcriptionResult, err := CallWhisper(payload.FilePath)
	if err != nil {
		if dbErr := db.UpdateJobFailed(payload.JobID, err.Error()); dbErr != nil {
			fmt.Printf("failed to update job status: %v\n", dbErr)
		}
		return err
	}

	// Update job to completed status with the transcription result
	if err := db.UpdateJobFinished(payload.JobID, transcriptionResult); err != nil {
		if dbErr := db.UpdateJobFailed(payload.JobID, err.Error()); dbErr != nil {
			fmt.Printf("failed to update job status: %v\n", dbErr)
		}
		return err
	}

	return nil
}

