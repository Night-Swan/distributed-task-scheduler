package db

import (
	"time"
)

type Job struct {
    ID int64 `db:"id"` 
    Status string `db:"status"`
    Payload []byte `db:"payload"`
    CreatedAt time.Time `db:"created_at"`
    FinishedAt *time.Time `db:"finished_at"`
    ErrorMessage *string `db:"error_message"`
    Result *string `db:"result"`
    SubmittedBy string `db:"submitted_by"`
    JobType string `db:"job_type"`
}

