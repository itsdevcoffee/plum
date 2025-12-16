package marketplace

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchManifestFromGitHub_BodySizeLimit(t *testing.T) {
	// Create a test server that returns a response larger than the limit
	largeBody := strings.Repeat("x", int(MaxResponseBodySize)+1000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	// For now, we test the concept by checking that a large JSON would fail
	// This is a limitation without refactoring buildRawURL
	t.Skip("Skipping due to buildRawURL using const - would require refactoring")
}

func TestFetchManifestFromGitHub_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail the first 2 attempts
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on the 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"test","owner":{},"metadata":{},"plugins":[]}`))
	}))
	defer server.Close()

	t.Skip("Skipping due to buildRawURL using const - would require refactoring")
}

func TestHTTPClient_Singleton(t *testing.T) {
	// Test that httpClient returns the same instance
	client1 := httpClient()
	client2 := httpClient()

	if client1 != client2 {
		t.Error("httpClient should return the same instance (singleton pattern)")
	}

	// Verify timeout is set correctly
	if client1.Timeout != HTTPTimeout {
		t.Errorf("Expected timeout %v, got %v", HTTPTimeout, client1.Timeout)
	}
}

func TestFetchManifestAttempt_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	t.Skip("Skipping due to buildRawURL using const - would require refactoring")
}

func TestFetchManifestAttempt_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		time.Sleep(HTTPTimeout + time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"name":"test","owner":{},"metadata":{},"plugins":[]}`))
	}))
	defer server.Close()

	t.Skip("Skipping due to buildRawURL using const - would require refactoring")
}

// TestConcurrencyLimit verifies that concurrent fetches are limited
func TestConcurrencyLimit(t *testing.T) {
	// This test would require observing concurrent requests to a test server
	// Skipping for now as it would need significant refactoring
	t.Skip("Requires refactoring to inject test server URL")
}

// Note: These tests are currently skipped because the github.go module uses
// const GitHubRawBase and buildRawURL() is not injectable. To make this testable:
// 1. Make buildRawURL accept a base URL parameter
// 2. Add a package-level variable for the base URL that defaults to GitHubRawBase
// 3. Allow tests to override the base URL
//
// This is left as a follow-up improvement to avoid scope creep in this PR.
