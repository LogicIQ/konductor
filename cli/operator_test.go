package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		response map[string]string
		want     string
	}{
		{
			name:     "valid version",
			response: map[string]string{"version": "v1.0.0"},
			want:     "v1.0.0",
		},
		{
			name:     "empty response",
			response: map[string]string{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			got := callEndpoint(server.URL)
			if got != tt.want {
				t.Errorf("callEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       string
	}{
		{
			name:       "healthy",
			statusCode: http.StatusOK,
			want:       "OK",
		},
		{
			name:       "unhealthy",
			statusCode: http.StatusServiceUnavailable,
			want:       "503 Service Unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			got := checkHealth(server.URL)
			if got != tt.want {
				t.Errorf("checkHealth() = %v, want %v", got, tt.want)
			}
		})
	}
}
