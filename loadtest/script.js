import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '30s', target: 10 },  // ramp up to 10 users over 30s
        { duration: '1m', target: 50 },   // ramp up to 50 users over 1 minute
        { duration: '30s', target: 0 },   // ramp down to 0
    ],
    thresholds: {
        http_req_duration: ['p(95)<2000'], // 95% of requests under 2 seconds
        http_req_failed: ['rate<0.05'],    // less than 5% failure rate
    },
};

export default function () {
    // Test 1: Submit a job
    const payload = JSON.stringify({
        job_type: 'llm_prompt',
        prompt: 'What is Go programming language?',
        submitted_by: 'loadtest',
        priority: 'default',
    });

    const params = { headers: { 'Content-Type': 'application/json' } };
    const submitRes = http.post('http://localhost:8080/jobs', payload, params);
    
    check(submitRes, {
        'job submitted successfully': (r) => r.status === 200,
        'response has job_id': (r) => JSON.parse(r.body).job_id > 0,
    });

    const jobID = JSON.parse(submitRes.body).job_id;

    sleep(1);

    // Test 2: Get job status
    const getRes = http.get(`http://localhost:8080/jobs/${jobID}`);
    check(getRes, {
        'job found': (r) => r.status === 200,
    });

    // Test 3: Get metrics
    const metricsRes = http.get('http://localhost:8080/metrics');
    check(metricsRes, {
        'metrics available': (r) => r.status === 200,
    });

    sleep(1);
}