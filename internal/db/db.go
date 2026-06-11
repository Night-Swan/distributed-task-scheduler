package db

import (
	"time"
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "os"
    "fmt"
    "encoding/json"
)

var Pool *pgxpool.Pool

func Connect() error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	config, err := pgxpool.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("could not parse config: %w", err)
	}

	config.ConnConfig.TLSConfig = nil

	var poolErr error
	Pool, poolErr = pgxpool.NewWithConfig(context.Background(), config)
	if poolErr != nil {
		return fmt.Errorf("could not create pool: %w", poolErr)
	}

	if err := Pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	return nil
}

type Job struct {
    ID int64 `db:"id"` 
    Status string `db:"status"`
    Payload json.RawMessage `db:"payload"`
    CreatedAt time.Time `db:"created_at"`
    FinishedAt *time.Time `db:"finished_at"`
    ErrorMessage *string `db:"error_message"`
    Result *string `db:"result"`
    SubmittedBy string `db:"submitted_by"`
    JobType string `db:"job_type"`
}

func CreateJob(submittedBy string, jobType string, payload json.RawMessage) (int64, error) {
	var id int64
	err := Pool.QueryRow(context.Background(), `
		INSERT INTO jobs (job_type, payload, submitted_by)
		VALUES ($1, $2, $3)
		RETURNING id
	`, jobType, payload, submittedBy).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetJob(id int64) (*Job, error) {
    job := &Job{}
    err := Pool.QueryRow(context.Background(), `
    SELECT id, status, payload, created_at, finished_at, error_message, result, submitted_by, job_type
    FROM jobs
    WHERE id = $1
    `, id).Scan(&job.ID, &job.Status, &job.Payload, &job.CreatedAt, &job.FinishedAt, &job.ErrorMessage, &job.Result, &job.SubmittedBy, &job.JobType)
    if err != nil {
        return nil, err
    }
    return job, nil
}

func UpdateJobRunning(id int64) error {
    _, err := Pool.Exec(context.Background(), `
    UPDATE jobs
    SET status = 'running'
    WHERE id = $1
    `, id)
    return err
}

func UpdateJobFinished(id int64, result string) error {
    now := time.Now()
    _, err := Pool.Exec(context.Background(), `
    UPDATE jobs
    SET status = 'completed', finished_at = $2, result = $3
    WHERE id = $1
    `, id, now, result)
    return err
}

func UpdateJobFailed(id int64, errorMessage string) error {
    now := time.Now()
    _, err := Pool.Exec(context.Background(), `
    UPDATE jobs
    SET status = 'failed', finished_at = $2, error_message = $3
    WHERE id = $1
    `, id, now, errorMessage)
    return err
}

func GetJobCounts() (map[string]int, error) {
    rows, err := Pool.Query(context.Background(), `
        SELECT status, COUNT(*) 
        FROM jobs 
        GROUP BY status
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    counts := map[string]int{
        "pending":   0,
        "running":   0,
        "completed": 0,
        "failed":    0,
    }

    for rows.Next() {
        var status string
        var count int
        if err := rows.Scan(&status, &count); err != nil {
            return nil, err
        }
        counts[status] = count
    }

    return counts, nil
}