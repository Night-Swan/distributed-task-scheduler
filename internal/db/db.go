package db

import (
	"time"
    "context"
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

func CreateJob(submittedBy string, jobType string, payload []byte) (int64, error) {
    var id int64
    err := Pool.QueryRow(context.Background(), `
    INSERT INTO jobs (job_type, payload, submitted_by)
    VALUES ($1, $2, $3)
    RETURNING id
    `, jobType, payload, submittedBy).Scan(&id)
    return id, err
}
