import http from 'k6/http';
import { check } from 'k6';

export const options = {
    scenarios: {
        login_load: {
            executor: 'ramping-arrival-rate',

            startRate: 5,
            timeUnit: '1s',

            preAllocatedVUs: 30,
            maxVUs: 150,

            stages: [
                { duration: '30s', target: 10 },
                { duration: '30s', target: 25 },
                { duration: '30s', target: 50 },
                { duration: '30s', target: 75 },
                { duration: '30s', target: 100 },
                { duration: '30s', target: 0 },
            ],
        },
    },

    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<500'],
        checks: ['rate>0.99'],
    },
};

export default function () {
    const url = __ENV.AUTH_URL || 'http://hlt.com/api/auth/login';

    const payload = JSON.stringify({
        email: __ENV.TEST_EMAIL,
        password: __ENV.TEST_PASSWORD,
    });

    const response = http.post(url, payload, {
        headers: {
            'Content-Type': 'application/json',
        },
        timeout: '10s',
    });

    check(response, {
        'login returns 200': (r) => r.status === 200,
    });
}

// ЗАПУСК
// TEST_EMAIL='test2@gmail.com' \
// TEST_PASSWORD='123' \
// k6 run login-rps-test.js