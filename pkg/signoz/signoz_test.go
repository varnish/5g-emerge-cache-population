package signoz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
)

func TestAuthenticateAccessJwt(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"accessJwt": "token"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl:      mockServer.URL,
		SignozUsername: "testuser",
		SignozPassword: "testpass",
	}

	token, err := Authenticate(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token != "token" {
		t.Fatalf("Expected token 'token', got %s", token)
	}
}

func TestAuthenticateDataAccessJwt(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"accessJwt": "token"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl: mockServer.URL,
	}

	token, err := Authenticate(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token != "token" {
		t.Fatalf("Expected token 'token', got %s", token)
	}
}

func TestAuthenticateServerError(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl: mockServer.URL,
	}

	_, err := Authenticate(cfg)
	if err.Error() != "Signoz authentication failed with status 500 Internal Server Error: " {
		t.Fatalf("Expected Signoz server error, got '%s'", err)
	}
}

func TestAuthenticateInvalidJson(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`data`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl: mockServer.URL,
	}

	_, err := Authenticate(cfg)
	if err.Error() != "invalid character 'd' looking for beginning of value" {
		t.Fatalf("Expected JSON unmarshaling error, got %s", err)
	}
}

func TestAuthenticateInvalidJsonRootObject(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`"abc"`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl: mockServer.URL,
	}

	_, err := Authenticate(cfg)
	if err.Error() != "Signoz authentication JSON root is not an object" {
		t.Fatalf("Expected JSON root object error, got %s", err)
	}
}

func TestAuthenticateMissingDataField(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"test": {"otherField": "value"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl: mockServer.URL,
	}

	_, err := Authenticate(cfg)
	if err.Error() != "Signoz authentication 'data' field is not an object in the JSON output" {
		t.Fatalf("Expected data field missing error, got %s", err)
	}
}

func TestAuthenticateMissingDataAccessJwtField(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"otherField": "value"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl: mockServer.URL,
	}

	_, err := Authenticate(cfg)
	if err.Error() != "Signoz authentication 'data.accessJwt' field is not a string in the JSON output" {
		t.Fatalf("Expected data.accessJwt field missing error, got %s", err)
	}
}

func TestQueryV4(t *testing.T) {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"accessJwt": "token"}`))
			return
		}
		if r.URL.Path == "/api/v4/query_range" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
  "data": {
    "result": [
      {
        "list": [
          {
            "data": {
              "title": "example_title",
              "total_watch_duration": 123.45
            }
          }
        ]
      }
    ]
  }
}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl:        mockServer.URL,
		SignozApiVersion: "v4",
	}

	queryResponse, err := Query(cfg, "bla")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(queryResponse) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(queryResponse))
	}

	result, ok := queryResponse[0].([]any)
	if !ok {
		t.Fatalf("Expected result to be an array, got %T", queryResponse[0])
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 elements in result, got %d", len(result))
	}

	title, ok := result[0].(string)
	if !ok || title != "example_title" {
		t.Fatalf("Expected title 'example_title', got %v", result[0])
	}

	duration, ok := result[1].(float64)
	if !ok || duration != 123.45 {
		t.Fatalf("Expected duration 123.45, got %v", result[1])
	}
}
