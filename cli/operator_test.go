package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestCheckHealthWithVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Konductor-Version", "v1.2.3")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	version, status := checkHealthWithVersion(server.URL)
	if version != "v1.2.3" {
		t.Errorf("expected version 'v1.2.3', got '%s'", version)
	}
	if status != "OK" {
		t.Errorf("expected status 'OK', got '%s'", status)
	}
}
