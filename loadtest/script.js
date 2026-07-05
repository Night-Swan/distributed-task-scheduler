import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const jobSubmissionRate = new Rate('job_submission_success');
const jobRetrievalRate = new Rate('job_retrieval_success');
const jobProcessingTime = new Trend('job_processing_time_ms');

export const options = {
    stages: [
        { duration: '30s', target: 10 },   // warm up
        { duration: '30s', target: 50 },   // normal load
        { duration: '30s', target: 100 },  // heavy load
        { duration: '30s', target: 200 },  // stress
        { duration: '30s', target: 500 },  // breaking point
        { duration: '30s', target: 0 },    // ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<3000'],
        http_req_failed: ['rate<0.10'],
        job_submission_success: ['rate>0.95'],
        job_retrieval_success: ['rate>0.95'],
    },
};

const JOB_TYPES = ['llm_prompt', 'embedding'];
const PRIORITIES = ['critical', 'default', 'low'];
const PROMPTS = [
    'Explain quantum computing in simple terms',
    'What is the difference between REST and GraphQL?',
    'Summarize the history of the internet',
    'What are the SOLID principles in software engineering?',
    'Explain how TCP/IP works',
];

function randomItem(arr) {
    return arr[Math.floor(Math.random() * arr.length)];
}

export default function () {
    const params = { headers: { 'Content-Type': 'application/json' } };

    group('submit job', () => {
        const payload = JSON.stringify({
            job_type: randomItem(JOB_TYPES),
            prompt: randomItem(PROMPTS),
            submitted_by: `user_${__VU}`,
            priority: randomItem(PRIORITIES),
        });

        const submitRes = http.post('http://localhost:8080/jobs', payload, params);

        const submitted = check(submitRes, {
            'job submitted successfully': (r) => r.status === 200,
            'response has job_id': (r) => {
                try {
                    return JSON.parse(r.body).job_id > 0;
                } catch {
                    return false;
                }
            },
        });

        jobSubmissionRate.add(submitted);

        if (submitted) {
            const jobID = JSON.parse(submitRes.body).job_id;
            const submitTime = Date.now();

            sleep(0.5);

            group('get job status', () => {
                const getRes = http.get(`http://localhost:8080/jobs/${jobID}`);

                const retrieved = check(getRes, {
                    'job found': (r) => r.status === 200,
                    'job has valid status': (r) => {
                        try {
                            const job = JSON.parse(r.body);
                            return ['pending', 'running', 'completed', 'failed'].includes(job.Status);
                        } catch {
                            return false;
                        }
                    },
                });

                jobRetrievalRate.add(retrieved);

                if (retrieved) {
                    const job = JSON.parse(getRes.body);
                    if (job.Status === 'completed' && job.FinishedAt) {
                        const finishedAt = new Date(job.FinishedAt).getTime();
                        jobProcessingTime.add(finishedAt - submitTime);
                    }
                }
            });
        }
    });

    group('get metrics', () => {
        const metricsRes = http.get('http://localhost:8080/metrics');
        check(metricsRes, {
            'metrics available': (r) => r.status === 200,
            'metrics has queue data': (r) => {
                try {
                    const m = JSON.parse(r.body);
                    return m.queue !== undefined && m.jobs !== undefined;
                } catch {
                    return false;
                }
            },
        });
    });

    sleep(0.5);
}