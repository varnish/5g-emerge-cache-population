package titles

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

func TestLoadTitles(t *testing.T) {
	// Create a temporary JSON file with test data
	testData := `{
    "5GBroadcast": {
        "Title": "5GBroadcast",
        "Url": "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5GBroadcast/5GBroadcast.m3u8",
        "Resolutions": [1183600, 2261600]
    },
    "5G-Emerge": {
        "Title": "5G-Emerge",
        "Url": "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5G-Emerge/5G-Emerge.m3u8",
        "Resolutions": [1183600, 2261600]
    }
}`
	tmpFile, err := os.CreateTemp("", "titles_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(testData)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load titles from the temporary file
	titles, err := LoadTitles(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadTitles returned an error: %v", err)
	}

	// Validate the loaded titles
	expected := map[string]types.TitleInfo{
		"5GBroadcast": {
			Title:       "5GBroadcast",
			Resolutions: []int{2261600, 1183600},
			Url:         "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5GBroadcast/5GBroadcast.m3u8",
		},
		"5G-Emerge": {
			Title:       "5G-Emerge",
			Resolutions: []int{2261600, 1183600},
			Url:         "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5G-Emerge/5G-Emerge.m3u8",
		},
	}

	if !reflect.DeepEqual(expected, titles) {
		t.Fatalf("Loaded titles do not match expected values. Expected: %v, Got: %v", expected, titles)
	}
}

func TestLoadTitlesWithNonExistentFile(t *testing.T) {
	_, err := LoadTitles("non_existent_file.json")
	if err == nil {
		t.Fatalf("Expected an error when loading a non-existent file, but got nil")
	}
}

func TestLoadTitlesWithEmptyFile(t *testing.T) {
	// Create an empty temporary file
	tmpFile, err := os.CreateTemp("", "titles_test_empty_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Load titles from the empty file
	titles, err := LoadTitles(tmpFile.Name())
	if err.Error() != "unexpected end of JSON input" {
		t.Fatalf("LoadTitles returned an error: %v", err)
	}

	if titles != nil {
		t.Fatalf("Expected nil, but got %v", titles)
	}

	// Load titles from the empty file string
	titles, err = LoadTitles("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if titles != nil {
		t.Fatalf("Expected nil, but got %v", titles)
	}
}

func runSignozQuery(signozResponse string) *httptest.Server {
	// Mock Signoz server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"accessJwt": "token"}`))
			return
		}
		if r.URL.Path == "/api/v4/query_range" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(signozResponse))
			return
		}
		http.NotFound(w, r)
	}))
	return mockServer
}

func TestGetTitlesWithWatchDuration(t *testing.T) {
	mockServer := runSignozQuery(`{
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
}`)
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl:        mockServer.URL,
		SignozApiVersion: "v4",
		AvailableTitles: map[string]types.TitleInfo{
			"example_title": {Title: "example_title", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600, 2261600}},
		},
	}

	titles, err := GetTitlesWithWatchDuration(cfg)
	if err != nil {
		t.Fatalf("GetTitlesWithWatchDuration returned an error: %v", err)
	}

	expected := map[string]types.TitleInfo{
		"example_title": {
			Title:         "example_title",
			WatchDuration: 123 * time.Millisecond,
			Preview:       false,
			Resolutions:   []int{2261600, 1183600},
			Url:           "https://example.com/test.m3u8",
		},
	}

	if !reflect.DeepEqual(expected, titles) {
		t.Fatalf("GetTitlesWithWatchDuration did not return expected values. Expected: %v, Got: %v", expected, titles)
	}
}

func TestGetTitlesWithWatchDurationForNonAvailableTitle(t *testing.T) {
	mockServer := runSignozQuery(`{
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
}`)
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl:        mockServer.URL,
		SignozApiVersion: "v4",
	}

	titles, err := GetTitlesWithWatchDuration(cfg)
	if err != nil {
		t.Fatalf("GetTitlesWithWatchDuration returned an error: %v", err)
	}

	if len(titles) != 0 {
		t.Fatalf("Expected no titles, but got: %v", titles)
	}
}
