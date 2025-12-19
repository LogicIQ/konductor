package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionHealthCheck(t *testing.T) {
	version = "test-version"
	checker := versionHealthCheck()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	err := checker(req)
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}

	if req.Response == nil {
		t.Fatal("expected response to be set")
	}

	if req.Response.Header.Get("X-Konductor-Version") != "test-version" {
		t.Errorf("expected version header 'test-version', got '%s'", req.Response.Header.Get("X-Konductor-Version"))
	}
}
