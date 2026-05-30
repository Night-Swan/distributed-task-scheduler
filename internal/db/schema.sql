CREATE TYPE job_status AS ENUM ('pending', 'running', 'completed', 'failed');
CREATE TYPE job_type AS ENUM ('llm_prompt', 'embedding', 'transcription', 'pdf_processing');

CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    status job_status NOT NULL DEFAULT 'pending',
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP DEFAULT NULL,
    error_message TEXT DEFAULT NULL,
    result TEXT DEFAULT NULL,
    submitted_by VARCHAR(255) NOT NULL,
    job_type job_type NOT NULL
);