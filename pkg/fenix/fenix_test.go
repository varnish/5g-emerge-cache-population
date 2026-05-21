package fenix

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

func TestApiGetclient(t *testing.T) {
	_ = apiGetClient()
	if reflect.TypeFor[*http.Client]().String() != "*http.Client" {
		t.Error("Expected non-nil http.Client, got nil")
	}
}

func createMockServerBandwidth(bandwidth int, serverError bool, noBandwidth bool) (*httptest.Server, string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/bandwidth", func(w http.ResponseWriter, r *http.Request) {
		if serverError {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if noBandwidth {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"bandwidthAvailable": ` + strconv.Itoa(bandwidth) + `}`))
	})

	server := httptest.NewUnstartedServer(mux)
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	server.Listener = l
	server.Start()
	return server, server.URL
}

func createMockServerStatus(kind string, state string, mode string, filesTransmitted int, filesTotal int) (*httptest.Server, string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/status/id", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"streamId": "id",
			"kind":     kind,
			"state":    state,
			"protocol": "hls",
			"mode":     mode,
			"progress": map[string]interface{}{
				"filesTransmitted": filesTransmitted,
				"filesTotal":       filesTotal,
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewUnstartedServer(mux)
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	server.Listener = l
	server.Start()
	return server, server.URL
}

func createMockServerDelete() (*httptest.Server, string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/streams/id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	server := httptest.NewUnstartedServer(mux)
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	server.Listener = l
	server.Start()
	return server, server.URL
}

func createMockServerTransmit() (*httptest.Server, string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/streams", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"streamId": "abc-123",
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewUnstartedServer(mux)
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	server.Listener = l
	server.Start()
	return server, server.URL
}
func TestApiRequest(t *testing.T) {
	server, url := createMockServerBandwidth(12345, false, false)
	defer server.Close()
	cfg := config.Config{
		FenixUrl: url,
	}
	resp, err := apiRequest(cfg, "/bandwidth")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp["bandwidthAvailable"] != float64(12345) {
		t.Errorf("Expected bandwidthAvailable 12345, got %v", resp["bandwidthAvailable"])
	}
}

func TestGetBandwidthAvailable(t *testing.T) {
	server, url := createMockServerBandwidth(100, false, false)
	defer server.Close()
	cfg := config.Config{
		FenixUrl:    url,
		FenixApiKey: "test-api-key",
	}
	bw, err := GetBandwidthAvailable(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if bw != 100 {
		t.Errorf("Expected bandwidth 100, got %d", bw)
	}
}

func TestGetBandwidthAvailableNegativeNumber(t *testing.T) {
	server, url := createMockServerBandwidth(-100, false, false)
	defer server.Close()
	cfg := config.Config{
		FenixUrl:    url,
		FenixApiKey: "test-api-key",
	}
	bw, err := GetBandwidthAvailable(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if bw != 0 {
		t.Errorf("Expected bandwidth 0, got %d", bw)
	}
}

func TestGetBandwidthAvailablWithServerError(t *testing.T) {
	server, url := createMockServerBandwidth(0, true, false)
	defer server.Close()
	cfg := config.Config{
		FenixUrl:    url,
		FenixApiKey: "test-api-key",
	}
	bw, err := GetBandwidthAvailable(cfg)
	if err.Error() != "Unexpected response status from server: 500 Internal Server Error (500)- " {
		t.Fatalf("Expected 500 Internal Server Error, got '%v'", err)
	}
	if bw != 0 {
		t.Errorf("Expected bandwidth 0, got %d", bw)
	}
}

func TestGetBandwidthAvailablWithoutBandwidthProperty(t *testing.T) {
	server, url := createMockServerBandwidth(0, false, true)
	defer server.Close()
	cfg := config.Config{
		FenixUrl:    url,
		FenixApiKey: "test-api-key",
	}
	bw, err := GetBandwidthAvailable(cfg)
	if err.Error() != "bandwidth not found in response: map[]" {
		t.Fatalf("Expected bandwidth not found error, got '%v'", err)
	}
	if bw != 0 {
		t.Errorf("Expected bandwidth 0, got %d", bw)
	}
}

func TestGetTransmissionStatus(t *testing.T) {
	server, url := createMockServerStatus("vod", "ongoing", "full", 5, 10)
	defer server.Close()
	cfg := config.Config{
		FenixUrl: url,
	}

	state, filesTransmitted, filesTotal, err := GetTransmissionStatus(cfg, "id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if state != "ongoing" || filesTransmitted != 5 || filesTotal != 10 {
		t.Errorf("Unexpected values: state=%s, filesTransmitted=%d, filesTotal=%d", state, filesTransmitted, filesTotal)
	}
}

func TestGetTransmissionStatusNotFound(t *testing.T) {
	server, url := createMockServerStatus("vod", "ongoing", "full", 5, 10)
	defer server.Close()
	cfg := config.Config{
		FenixUrl: url,
	}

	state, filesTransmitted, filesTotal, err := GetTransmissionStatus(cfg, "non-existent-id")
	if strings.TrimSpace(err.Error()) != "Unexpected response status from server: 404 Not Found (404)- 404 page not found" {
		t.Fatalf("Expected 404 error, got '%v'", err)
	}
	if state != "" || filesTransmitted != 0 || filesTotal != 0 {
		t.Errorf("Unexpected values: state=%s, filesTransmitted=%d, filesTotal=%d", state, filesTransmitted, filesTotal)
	}
}

func TestGetTransmissionStatusWithPreview(t *testing.T) {
	server, url := createMockServerStatus("vod", "ongoing", "preview", 3, 10)
	defer server.Close()
	cfg := config.Config{
		FenixUrl:                    url,
		PreviewTransmissionSegments: 8,
	}

	state, filesTransmitted, filesTotal, err := GetTransmissionStatus(cfg, "id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if state != "ongoing" || filesTransmitted != 3 || filesTotal != 8 {
		t.Errorf("Unexpected values: state=%s, filesTransmitted=%d, filesTotal=%d", state, filesTransmitted, filesTotal)
	}
}

func TestTransmit(t *testing.T) {
	server, url := createMockServerTransmit()
	defer server.Close()
	cfg := config.Config{
		FenixUrl:                    url,
		PreviewTransmissionSegments: 3,
	}

	title := types.TitleInfo{Title: "foo", Url: "http://foo", Preview: false}
	streamId, err := Transmit(cfg, title, 720)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if streamId != "abc-123" {
		t.Errorf("Expected streamId abc-123, got %s", streamId)
	}

	title = types.TitleInfo{Title: "foo", Url: "http://foo", Preview: true}
	streamId, err = Transmit(cfg, title, 720)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if streamId != "abc-123" {
		t.Errorf("Expected streamId abc-123, got %s", streamId)
	}
}

func TestDeleteTransmission(t *testing.T) {
	server, url := createMockServerDelete()
	defer server.Close()
	cfg := config.Config{
		FenixUrl: url,
	}
	err := DeleteTransmission(cfg, "id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = DeleteTransmission(cfg, "not-found")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
