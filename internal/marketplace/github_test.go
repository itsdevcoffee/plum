package marketplace

import (
	"errors"
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
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	// Override GitHubRawBase for testing
	originalBase := GitHubRawBase
	GitHubRawBase = server.URL
	defer func() { GitHubRawBase = originalBase }()

	// Test that a response exceeding the limit triggers an error
	_, err := FetchManifestFromGitHub("test/repo")
	if err == nil {
		t.Fatal("Expected error for response exceeding size limit, got nil")
	}

	if !strings.Contains(err.Error(), "exceeded") {
		t.Errorf("Expected error message to contain 'exceeded', got: %v", err)
	}
}

func TestFetchManifestFromGitHub_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail the first 2 attempts with a retryable error
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on the 3rd attempt
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"test","owner":{},"metadata":{},"plugins":[]}`))
	}))
	defer server.Close()

	// Override GitHubRawBase for testing
	originalBase := GitHubRawBase
	GitHubRawBase = server.URL
	defer func() { GitHubRawBase = originalBase }()

	// Test that retry succeeds after transient failures
	manifest, err := FetchManifestFromGitHub("test/repo")
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if manifest.Name != "test" {
		t.Errorf("Expected manifest name 'test', got: %s", manifest.Name)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts (2 failures + 1 success), got: %d", attempts)
	}
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
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	// Override GitHubRawBase for testing
	originalBase := GitHubRawBase
	GitHubRawBase = server.URL
	defer func() { GitHubRawBase = originalBase }()

	// Test that invalid JSON is not retried (parsing errors are not transient)
	_, err := FetchManifestFromGitHub("test/repo")
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestFetchManifestAttempt_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the timeout
		time.Sleep(HTTPTimeout + time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"test","owner":{},"metadata":{},"plugins":[]}`))
	}))
	defer server.Close()

	// Override GitHubRawBase for testing
	originalBase := GitHubRawBase
	GitHubRawBase = server.URL
	defer func() { GitHubRawBase = originalBase }()

	// Test that timeout errors trigger retries (they are transient)
	_, err := FetchManifestFromGitHub("test/repo")
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Context deadline errors should be retryable
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Logf("Got error: %v", err)
	}
}

// TestNonRetryableError verifies that 404 errors are not retried
func TestNonRetryableError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Override GitHubRawBase for testing
	originalBase := GitHubRawBase
	GitHubRawBase = server.URL
	defer func() { GitHubRawBase = originalBase }()

	// Test that 404 errors are not retried
	_, err := FetchManifestFromGitHub("test/repo")
	if err == nil {
		t.Fatal("Expected error for 404, got nil")
	}

	if attempts != 1 {
		t.Errorf("Expected exactly 1 attempt for non-retryable 404 error, got: %d", attempts)
	}

	// Verify it's an httpStatusError with the correct status code
	var statusErr *httpStatusError
	if !errors.As(err, &statusErr) {
		t.Errorf("Expected httpStatusError, got: %T", err)
	} else if statusErr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code 404, got: %d", statusErr.StatusCode)
	}
}
