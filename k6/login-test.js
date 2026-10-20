import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '30s', target: 5 },
        { duration: '30s', target: 10 },
        { duration: '30s', target: 20 },
        { duration: '30s', target: 0 },
    ],

    thresholds: {
        http_req_failed: ['rate<0.01'],
        http_req_duration: ['p(95)<1000'],
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

    sleep(1);
}

// ЗАПУСК
// TEST_EMAIL='test@gmail.com' \
// TEST_PASSWORD='123' \
// k6 run login-test.js