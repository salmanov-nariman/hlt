package service

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestEncodeDecodeCursor_Success(t *testing.T) {
	originalTime := time.Date(2026, 11, 5, 15, 30, 45, 123456000, time.UTC)
	originalID := "6bc30c6e-2918-4f35-93c8-5eac50a0e815"

	encoded := encodeCursor(originalTime, originalID)
	if encoded == "" {
		t.Fatal("expected encoded cursor, got empty string")
	}

	decodedTime, decodedID, err := decodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected error decoding valid cursor: %v", err)
	}

	if decodedID == nil || *decodedID != originalID {
		t.Errorf("expected ID %s, got %v", originalID, decodedID)
	}

	if decodedTime == nil || !decodedTime.Equal(originalTime) {
		t.Errorf("expected time %v, got %v", originalTime, decodedTime)
	}
}

func TestDecodeCursor_Errors(t *testing.T) {
	tests := []struct {
		name        string
		inputCursor string
	}{
		{
			name:        "Invalid Base64",
			inputCursor: "not-a-base-64-string!@#",
		},
		{
			name:        "Missing pipe separator",
			inputCursor: base64.RawURLEncoding.EncodeToString([]byte("2026-11-05T15:30:45Z_NO_PIPE_HERE")),
		},
		{
			name:        "Too many pipe separators",
			inputCursor: base64.RawURLEncoding.EncodeToString([]byte("2026-11-05T15:30:45Z|id-123|extra-data")),
		},
		{
			name:        "Invalid timestamp format",
			inputCursor: base64.RawURLEncoding.EncodeToString([]byte("05-11-2026 15:30|6bc30c6e-2918-4f35")),
		},
		{
			name:        "Empty string before pipe",
			inputCursor: base64.RawURLEncoding.EncodeToString([]byte("|6bc30c6e-2918-4f35")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeCursor(tt.inputCursor)

			if err != ErrInvalidCursor {
				t.Errorf("expected ErrInvalidCursor, got %v", err)
			}
		})
	}
}