# Distributed Tasks Scheduler

## Description
The scheduler is a job queue system built in Go. It accepts jobs via REST API, queues via Redis, and distributes the different jobs to concurrent workers for processing. The jobs are stored in PostgreSQL, which tracks the entire process from the beginning (submitting a job) to processing to completion or failure. The system supports 4 job types focused on AI:
1. LLM Text Generation
2. Audio Transcription 
3. Text Embeddings
4. Pdf Summarization
These jobs are processed through local models which are Ollama and Whisper without any external API dependencies. 


## Tech stack
1. Go: Run many tasks concurrently using lightweight goroutines without the overhead of managing OS threads.
2. Gin: Build fast and lightweight HTTP APIs with built-in routing, middleware, and request handling.
3. Asynq: Process background jobs asynchronously through a task queue, keeping API responses fast and responsive.
4. Redis: Provide high-speed caching and data access using an in-memory data store. 
5. PostgreSQL: Reliably store and query structured application data with transactions and powerful SQL capabilities.

### Design Decisions
1. Go over Java/Python because it provides lightweight goroutines for efficient concurrency with low runtime overhead.
2. Redis over Kafka because this case only needs a simple task queue or messaging system, not Kafka's more advanced event-streaming architecture.
3. Asynq over Celery/BullMQ as it is a Go-native job queue built on Redis, so no need to bridge to Python workers. Handles retries, scheduling, and priorities out of the box.
4. Gin over Echo/Chi because of its minimal and fast HTTP framework with excellent middleware support and the largest Go web framework community.

## Project Structure
├── cmd/server/main.go          # Entry point — wires everything together and starts the server
├── internal/
│   ├── api/
│   │   ├── handlers.go         # HTTP handlers for job submission and status
│   │   ├── middleware.go       # Rate limiting middleware
│   │   └── integration_test.go # Integration tests
│   ├── db/
│   │   ├── db.go               # Database connection pool and query functions
│   │   └── schema.sql          # Reference schema (migrations are source of truth)
│   └── jobs/
│       └── tasks.go            # Job type definitions, worker handlers, external service calls
├── migrations/                 # Database migrations managed by golang-migrate
├── uploads/                    # Temporary storage for uploaded audio and PDF files
├── docker-compose.yml          # Infrastructure — Redis, PostgreSQL, Asynqmon, Whisper
├── .env.example                # Template for required environment variables
└── README.md

## Prerequisites
1. Go (version 1.22 or above): https://go.dev/doc/install
2. Ollama models: https://ollama.com/download/linux, then pull the required model: `ollama pull llama3.2` and `ollama pull nomic-embed-text`
3. Docker and docker compose: https://docs.docker.com/engine/install/, https://docs.docker.com/compose/install/
4. golang-migrate — for running database migrations:```curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/migrate```
5. poppler-utils — for PDF text extraction: `bashsudo apt install poppler-utils`
6. NVIDIA GPU drivers (optional, for GPU acceleration): `bashsudo apt install nvidia-driver-595`

## Setup
Clone the repository
`git clone https://github.com/Night-Swan/distributed-task-scheduler.git`
`cd distributed-task-scheduler`

Copy contents of .env.example file to your own .env file
`cp .env.example .env`

### First time setup
```bash
export $(cat .env | xargs)
sudo docker compose up -d
sleep 3 && migrate -path migrations -database "$DATABASE_URL" up
go get ./...
go build -o server cmd/server/main.go
OLLAMA_GPU=1 ollama serve &
./server
```

### Daily startup
```bash
export $(cat .env | xargs)
sudo docker compose up -d
OLLAMA_GPU=1 ollama serve &
./server
```

### After making changes
```bash
go build -o server cmd/server/main.go
./server
```

## API Endpoints

### Submit a text job
`POST /jobs`

Body:
```json
{
    "job_type": "llm_prompt",
    "prompt": "Summarize the theory of relativity",
    "submitted_by": "username",
    "priority": "default"
}
```
Job types: `llm_prompt`, `embedding`
Priority levels: `critical`, `default`, `low`

Response:
```json
{"job_id": 1}
```

### Submit a transcription job
`POST /jobs/transcription`

Multipart form fields:
- `file` — audio file (wav, mp3)
- `submitted_by` — username
- `priority` — optional, defaults to `default`

### Submit a PDF job
`POST /jobs/pdf`

Multipart form fields:
- `file` — PDF file
- `submitted_by` — username
- `priority` — optional, defaults to `default`

### Get job status
`GET /jobs/:id`

Response:
```json
{
    "ID": 1,
    "Status": "completed",
    "JobType": "llm_prompt",
    "Result": "...",
    "SubmittedBy": "username",
    "Priority": "default",
    "CreatedAt": "...",
    "FinishedAt": "..."
}
```

### Get system metrics
`GET /metrics`


## Job Types

1. llm_prompt — sends a text prompt to a local Llama 3.2 model via Ollama and returns the generated response
2. embedding — converts text into a vector representation using the nomic-embed-text model, useful for semantic search and RAG pipelines
3. transcription — transcribes an uploaded audio file to text using a local Whisper model
4. pdf_processing — extracts text from an uploaded PDF and summarizes it using Llama 3.2