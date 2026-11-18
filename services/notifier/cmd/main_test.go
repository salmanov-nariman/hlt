package main

import (
	"context"
	"errors"
	"testing"

	"notifier/internal/clients/user"

	"github.com/stretchr/testify/assert"
)

type mockTokenProvider struct {
	tokens map[string]string
	errs   map[string]error
}

func (m *mockTokenProvider) GetDeviceToken(ctx context.Context, userID string) (string, error) {
	if err, exists := m.errs[userID]; exists {
		return "", err
	}
	if token, exists := m.tokens[userID]; exists {
		return token, nil
	}
	return "", user.ErrTokenNotFound
}

type mockPushSender struct {
	sentPushes map[string]int
	sendErr    error
}

func (m *mockPushSender) SendPush(ctx context.Context, token, title, body string) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	if m.sentPushes == nil {
		m.sentPushes = make(map[string]int)
	}
	m.sentPushes[token]++
	return nil
}

type TestAndBench struct {
	name         string
	verifyResult func(t testing.TB, bench bool, testName string)
}

func Test_ProcessPushNotification(t *testing.T) {
	TBProcessPushNotification(t, nil)
}

func Benchmark_ProcessPushNotification(b *testing.B) {
	TBProcessPushNotification(nil, b)
}

func TBProcessPushNotification(t *testing.T, b *testing.B) {
	ctx := context.Background()

	// Входные данные для тестов
	type testSetup struct {
		tokens        map[string]string
		errs          map[string]error
		sendErr       error
		targetUserIDs []string
	}

	testData := map[string]testSetup{
		"success": {
			tokens:        map[string]string{"user_1": "token_1", "user_2": "token_2"},
			targetUserIDs: []string{"user_1", "user_2"},
		},
		"token_not_found": {
			errs:          map[string]error{"user_offline": user.ErrTokenNotFound},
			targetUserIDs: []string{"user_offline"},
		},
		"db_error": {
			errs:          map[string]error{"user_error": errors.New("db connection failed")},
			targetUserIDs: []string{"user_error"},
		},
		"partial_success": {
			tokens:        map[string]string{"user_good": "token_good"},
			errs:          map[string]error{"user_bad": user.ErrTokenNotFound},
			targetUserIDs: []string{"user_good", "user_bad"},
		},
		"firebase_error": {
			tokens:        map[string]string{"user_1": "token_1"},
			sendErr:       errors.New("firebase internal error"),
			targetUserIDs: []string{"user_1"},
		},
	}

	// Ожидаемые результаты для assert
	type expectedDataSetup struct {
		expectTotal  int
		expectPushes map[string]int
	}

	expectedData := map[string]expectedDataSetup{
		"success": {
			expectTotal:  2,
			expectPushes: map[string]int{"token_1": 1, "token_2": 1},
		},
		"token_not_found": {
			expectTotal:  0,
			expectPushes: map[string]int{},
		},
		"db_error": {
			expectTotal:  0,
			expectPushes: map[string]int{},
		},
		"partial_success": {
			expectTotal:  1,
			expectPushes: map[string]int{"token_good": 1},
		},
		"firebase_error": {
			expectTotal:  0,
			expectPushes: map[string]int{},
		},
	}

	tests := []TestAndBench{
		{
			name: "success",
			verifyResult: func(t testing.TB, bench bool, testName string) {
				data := testData[testName]
				tokenProv := &mockTokenProvider{tokens: data.tokens, errs: data.errs}
				pushSender := &mockPushSender{sendErr: data.sendErr}
				event := ChatEvent{TargetUserIDs: data.targetUserIDs}

				processPushNotificaation(ctx, tokenProv, pushSender, event)

				if !bench {
					exp := expectedData[testName]
					assert.Equal(t, exp.expectTotal, len(pushSender.sentPushes))
					for token, expectedCount := range exp.expectPushes {
						assert.Equal(t, expectedCount, pushSender.sentPushes[token])
					}
				}
			},
		},
		{
			name: "token_not_found",
			verifyResult: func(t testing.TB, bench bool, testName string) {
				data := testData[testName]
				tokenProv := &mockTokenProvider{tokens: data.tokens, errs: data.errs}
				pushSender := &mockPushSender{sendErr: data.sendErr}
				event := ChatEvent{TargetUserIDs: data.targetUserIDs}

				processPushNotificaation(ctx, tokenProv, pushSender, event)

				if !bench {
					assert.Empty(t, pushSender.sentPushes)
				}
			},
		},
		{
			name: "db_error",
			verifyResult: func(t testing.TB, bench bool, testName string) {
				data := testData[testName]
				tokenProv := &mockTokenProvider{tokens: data.tokens, errs: data.errs}
				pushSender := &mockPushSender{sendErr: data.sendErr}
				event := ChatEvent{TargetUserIDs: data.targetUserIDs}

				processPushNotificaation(ctx, tokenProv, pushSender, event)

				if !bench {
					assert.Empty(t, pushSender.sentPushes)
				}
			},
		},
		{
			name: "partial_success",
			verifyResult: func(t testing.TB, bench bool, testName string) {
				data := testData[testName]
				tokenProv := &mockTokenProvider{tokens: data.tokens, errs: data.errs}
				pushSender := &mockPushSender{sendErr: data.sendErr}
				event := ChatEvent{TargetUserIDs: data.targetUserIDs}

				processPushNotificaation(ctx, tokenProv, pushSender, event)

				if !bench {
					exp := expectedData[testName]
					assert.Equal(t, exp.expectTotal, len(pushSender.sentPushes))
					assert.Equal(t, exp.expectPushes["token_good"], pushSender.sentPushes["token_good"])
				}
			},
		},
		{
			name: "firebase_error",
			verifyResult: func(t testing.TB, bench bool, testName string) {
				data := testData[testName]
				tokenProv := &mockTokenProvider{tokens: data.tokens, errs: data.errs}
				pushSender := &mockPushSender{sendErr: data.sendErr}
				event := ChatEvent{TargetUserIDs: data.targetUserIDs}

				// Функция не должна паниковать при ошибке со стороны Firebase
				assert.NotPanics(t, func() {
					processPushNotificaation(ctx, tokenProv, pushSender, event)
				})

				if !bench {
					assert.Empty(t, pushSender.sentPushes)
				}
			},
		},
	}

	for _, test := range tests {
		if t != nil {
			t.Run(test.name, func(t *testing.T) {
				test.verifyResult(t, false, test.name)
			})
		} else if b != nil {
			b.ResetTimer()
			b.Run(test.name, func(b *testing.B) {
				b.StopTimer()
				b.StartTimer()
				for i := 0; i < b.N; i++ {
					test.verifyResult(b, true, test.name)
				}
			})
		}
	}
}