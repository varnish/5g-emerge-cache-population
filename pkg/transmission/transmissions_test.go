package transmission

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	types "github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

type FenixStatusProgressResponse struct {
	FilesTransmitted int `json:"filesTransmitted"`
	FilesTotal       int `json:"filesTotal"`
}

type FenixStatusResponse struct {
	StreamId string                      `json:"streamId"`
	Kind     string                      `json:"kind"`
	State    string                      `json:"state"`
	Protocol string                      `json:"protocol"`
	Mode     string                      `json:"mode"`
	Progress FenixStatusProgressResponse `json:"progress"`
}

func runFenixQuery(bandwidth []int, responses []FenixStatusResponse, streamIds []string) *httptest.Server {
	statusCountCall := 0
	transmitCountCall := 0
	bandwidthCountCall := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bandwidth" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if bandwidthCountCall > len(bandwidth)-1 {
				bandwidthCountCall = 0
			}
			w.Write([]byte(`{"bandwidthAvailable": ` + strconv.Itoa(bandwidth[bandwidthCountCall]) + `}`))
			bandwidthCountCall++
			return
		}

		if r.URL.Path == "/streams" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			if transmitCountCall > len(streamIds)-1 {
				transmitCountCall = 0
			}
			resp := map[string]interface{}{
				"streamId": streamIds[transmitCountCall],
			}
			transmitCountCall++
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/streams" && r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/status/") && len(r.URL.Path) > len("/status/") {
			w.Header().Set("Content-Type", "application/json")
			streamID := r.URL.Path[len("/status/"):]

			w.WriteHeader(http.StatusOK)
			if statusCountCall > len(responses)-1 {
				statusCountCall = 0
			}
			respData := responses[statusCountCall]
			if respData.StreamId != streamID {
				http.NotFound(w, r)
				return
			}

			statusCountCall++
			json.NewEncoder(w).Encode(respData)
			return
		}
		http.NotFound(w, r)
	}))
}

type SignozDataResponse struct {
	Title              string  `json:"title"`
	TotalWatchDuration float64 `json:"total_watch_duration"`
}

type SignozQueryV4Response struct {
	Data struct {
		Result []struct {
			List []struct {
				Data SignozDataResponse `json:"data"`
			} `json:"list"`
		} `json:"result"`
	} `json:"data"`
}

func newSignozQueryV4Response(dataArr []SignozDataResponse) SignozQueryV4Response {
	result := make([]struct {
		List []struct {
			Data SignozDataResponse `json:"data"`
		} `json:"list"`
	}, 1)

	list := make([]struct {
		Data SignozDataResponse `json:"data"`
	}, len(dataArr))

	for i, d := range dataArr {
		list[i] = struct {
			Data SignozDataResponse `json:"data"`
		}{Data: d}
	}

	result[0] = struct {
		List []struct {
			Data SignozDataResponse `json:"data"`
		} `json:"list"`
	}{List: list}

	return SignozQueryV4Response{
		Data: struct {
			Result []struct {
				List []struct {
					Data SignozDataResponse `json:"data"`
				} `json:"list"`
			} `json:"result"`
		}{
			Result: result,
		},
	}
}

func parseJsonLogData(logBuffer *bytes.Buffer) ([]map[string]any, error) {
	var logData []map[string]any

	for _, line := range bytes.Split(logBuffer.Bytes(), []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		dec := json.NewDecoder(bytes.NewReader(line))
		dec.UseNumber()

		var entry map[string]any
		if err := dec.Decode(&entry); err != nil {
			return nil, err
		}

		entry = convertJSONNumbers(entry).(map[string]any)
		logData = append(logData, entry)
	}

	return logData, nil
}

func convertJSONNumbers(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			x[k] = convertJSONNumbers(vv)
		}
		return x
	case []any:
		for i, vv := range x {
			x[i] = convertJSONNumbers(vv)
		}
		return x
	case json.Number:
		// Decide int vs float by inspecting the literal.
		s := x.String()
		if !bytes.ContainsAny([]byte(s), ".eE") {
			if i, errInt := x.Int64(); errInt == nil {
				return int(i)
			}
		}
		if f, errFloat := x.Float64(); errFloat == nil {
			return f
		}
		return s // fallback (should be rare)
	default:
		return v
	}
}

func runSignozQuery(responses []SignozQueryV4Response) *httptest.Server {
	countCall := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"accessJwt": "token"}`))
			return
		}
		if r.URL.Path == "/api/v4/query_range" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			if countCall > len(responses)-1 {
				countCall = 0
			}
			respData := responses[countCall]
			countCall++
			json.NewEncoder(w).Encode(respData)
			return
		}

		http.NotFound(w, r)
	}))
}

func TestInitTransmission(t *testing.T) {

	transmission := InitTransmission("Title", "000-111-222", 123, false)

	expected := types.TitleTransmission{
		Title:                 "Title",
		StreamId:              "000-111-222",
		State:                 "queued",
		Resolution:            123,
		FilesTransmitted:      0,
		FilesTotal:            0,
		PercentageTransmitted: 0.0,
		Preview:               false,
	}

	if transmission != expected {
		t.Errorf("Expected %+v, but got %+v", expected, transmission)
	}
}

func TestUpdateTransmissionState(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	transmissions := []types.TitleTransmission{
		{
			Title:                 "Title",
			StreamId:              "1",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mockServer := runFenixQuery([]int{1000}, []FenixStatusResponse{
		{
			StreamId: "1",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 5,
				FilesTotal:       10,
			},
		},
	}, []string{})
	cfg := config.Config{
		FenixUrl:                    mockServer.URL,
		PreviewTransmissionSegments: 8,
		Breakpoint:                  true,
	}
	err := UpdateTransmissionState(ctx, cfg, &transmissions)
	defer mockServer.Close()
	cancel()
	ctx.Done()
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	expected := types.TitleTransmission{
		Title:                 "Title",
		StreamId:              "1",
		State:                 "transmitting",
		Resolution:            123,
		FilesTransmitted:      5,
		FilesTotal:            10,
		PercentageTransmitted: 50.0,
		Preview:               false,
	}

	if transmissions[0] != expected {
		t.Errorf("Expected %+v, but got %+v", expected, transmissions[0])
	}

}

func TestUpdateTransmissionStateWithTransmissionStateError(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	transmissions := []types.TitleTransmission{
		{
			Title:                 "Title",
			StreamId:              "2",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
		{
			Title:                 "Title2",
			StreamId:              "1",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mockServer := runFenixQuery([]int{1000}, []FenixStatusResponse{
		{
			StreamId: "1",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 5,
				FilesTotal:       10,
			},
		},
	}, []string{})
	cfg := config.Config{
		FenixUrl:                    mockServer.URL,
		PreviewTransmissionSegments: 8,
		Breakpoint:                  true,
	}
	err := UpdateTransmissionState(ctx, cfg, &transmissions)
	defer mockServer.Close()
	cancel()
	ctx.Done()
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[2]["msg"] != "Error getting transmission status" {
		t.Errorf("Expected transmission status error, but got: %v", logData[2]["msg"])
	}

	expected := types.TitleTransmission{
		Title:                 "Title2",
		StreamId:              "1",
		State:                 "transmitting",
		Resolution:            123,
		FilesTransmitted:      5,
		FilesTotal:            10,
		PercentageTransmitted: 50.0,
		Preview:               false,
	}

	if transmissions[1] != expected {
		t.Errorf("Expected %+v, but got %+v", expected, transmissions[1])
	}
}

func TestUpdateTransmissionStateWithoutChanges(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	transmissions := []types.TitleTransmission{
		{
			Title:                 "Title",
			StreamId:              "1",
			State:                 "transmitting",
			Resolution:            123,
			FilesTransmitted:      5,
			FilesTotal:            10,
			PercentageTransmitted: 50.0,
			Preview:               false,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mockServer := runFenixQuery([]int{1000}, []FenixStatusResponse{
		{
			StreamId: "1",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 5,
				FilesTotal:       10,
			},
		},
	}, []string{})
	cfg := config.Config{
		FenixUrl:   mockServer.URL,
		Breakpoint: true,
		LogLevel:   "DEBUG",
	}
	err := UpdateTransmissionState(ctx, cfg, &transmissions)
	defer mockServer.Close()
	defer cancel()
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)
	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[1]["msg"] != "No change in transmission status" {
		t.Errorf("Expected log message 'No change in transmission status', but got '%v'", logData[1]["msg"])
	}
}

func TestUpdateTransmissionStateWithPercentageRounding(t *testing.T) {
	transmissions := []types.TitleTransmission{
		{
			Title:                 "Title",
			StreamId:              "1",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mockServer := runFenixQuery([]int{1000}, []FenixStatusResponse{
		{
			StreamId: "1",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 3,
				FilesTotal:       10,
			},
		},
	}, []string{})
	cfg := config.Config{
		FenixUrl:                    mockServer.URL,
		PreviewTransmissionSegments: 8,
		Breakpoint:                  true,
	}
	err := UpdateTransmissionState(ctx, cfg, &transmissions)
	defer mockServer.Close()
	defer cancel()
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	if transmissions[0].PercentageTransmitted != 30 {
		t.Errorf("Expected 30 percent transmitted, but got %v", transmissions[0].PercentageTransmitted)
	}

}

func TestUpdateTransmissionStateWithNegativeTotalFiles(t *testing.T) {
	transmissions := []types.TitleTransmission{
		{
			Title:                 "Title",
			StreamId:              "1",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mockServer := runFenixQuery([]int{1000}, []FenixStatusResponse{
		{
			StreamId: "1",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 5,
				FilesTotal:       -10,
			},
		},
	}, []string{})
	cfg := config.Config{
		FenixUrl:                    mockServer.URL,
		PreviewTransmissionSegments: 8,
		Breakpoint:                  true,
	}
	err := UpdateTransmissionState(ctx, cfg, &transmissions)
	defer mockServer.Close()
	cancel()
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	if transmissions[0].PercentageTransmitted != 0 {
		t.Errorf("Expected 0 percent transmitted, but got %v", transmissions[0].PercentageTransmitted)
	}
}

func TestUpdateTransmissionStateWithContextCancel(t *testing.T) {
	transmissions := []types.TitleTransmission{
		{
			Title:                 "Title",
			StreamId:              "1",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
		{
			Title:                 "Title2",
			StreamId:              "2",
			State:                 "queued",
			Resolution:            123,
			FilesTransmitted:      0,
			FilesTotal:            10,
			PercentageTransmitted: 0.0,
			Preview:               false,
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	mockServer := runFenixQuery([]int{1000}, []FenixStatusResponse{
		{
			StreamId: "1",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 6,
				FilesTotal:       10,
			},
		},
		{
			StreamId: "2",
			Kind:     "vod",
			State:    "transmitting",
			Mode:     "full",
			Protocol: "hls",
			Progress: FenixStatusProgressResponse{
				FilesTransmitted: 3,
				FilesTotal:       10,
			},
		},
	}, []string{})
	cfg := config.Config{
		FenixUrl:                    mockServer.URL,
		PreviewTransmissionSegments: 8,
		Sleep:                       1 * time.Millisecond,
	}

	errs := make(chan error)

	go func() {
		errs <- UpdateTransmissionState(ctx, cfg, &transmissions)
	}()

	defer mockServer.Close()
	time.Sleep(1 * time.Second)
	cancel()
	err := <-errs
	if err.Error() != "context canceled" {
		t.Errorf("Expected context canceled error, but got %v", err)
	}

	if transmissions[0].FilesTransmitted != 6 {
		t.Errorf("Expected 6 files transmitted on first title, but got %v", transmissions[0].FilesTransmitted)
	}

	if transmissions[1].FilesTransmitted != 3 {
		t.Errorf("Expected 3 files transmitted on second title, but got %v", transmissions[1].FilesTransmitted)
	}
}

func TestTransmitPopularTitlesWithErrorGettingPopularTitles(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	cfg := config.Config{
		SignozUrl:        mockServer.URL,
		Breakpoint:       true,
		SignozApiVersion: "v4",
		LogLevel:         "Debug",
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if strings.TrimSpace(err.Error()) != "Signoz authentication failed with status 404 Not Found: 404 page not found" {
		t.Errorf("Expected Signoz error, but got '%v'", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[1]["msg"] != "Error getting popular titles" {
		t.Errorf("Expected log message 'Error getting popular titles', but got '%v'", logData[1]["msg"])
	}

	if len(transmissions) != 0 {
		t.Errorf("Expected no transmissions to be queued, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithWatchDurationBelowMinimumPreviewDuration(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 10,
		},
	})

	mockServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600, 2261600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[3]["msg"] != "Title watch duration below minimum preview watch duration, skipping" {
		t.Errorf("Expected log message 'Title watch duration below minimum preview watch duration, skipping', but got '%v'", logData[3]["msg"])
	}

	if len(transmissions) != 0 {
		t.Errorf("Expected no transmissions to be queued, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithPreview(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 40000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{3000000}, []FenixStatusResponse{}, []string{"abc-123", "xyz-456"})
	defer mockFenixServer.Close()

	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		PreviewTransmissionSegments: 4,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600, 2261600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[4]["msg"] != "Transmitted title" {
		t.Errorf("Expected log message 'Transmitted title', but got '%v'", logData[4]["msg"])
	}

	if logData[4]["title"] != "example" {
		t.Errorf("Expected log message title field to be 'example', but got '%v'", logData[4]["title"])
	}

	if logData[4]["resolution"] != 2261600 {
		t.Errorf("Expected log message resolution field to be '2261600', but got '%v'", logData[4]["resolution"])
	}

	if logData[4]["preview"] != true {
		t.Errorf("Expected log message preview field to be 'true', but got '%v'", logData[4]["preview"])
	}

	if logData[4]["streamId"] != "abc-123" {
		t.Errorf("Expected log message streamId field to be 'abc-123', but got '%v'", logData[4]["streamId"])
	}

	if transmissions[0].StreamId != "abc-123" {
		t.Errorf("Expected transmission streamId to be 'abc-123', but got '%v'", transmissions[0].StreamId)
	}

	if transmissions[0].State != "queued" {
		t.Errorf("Expected transmission state to be 'queued', but got '%v'", transmissions[0].State)
	}

	if transmissions[0].Resolution != 2261600 {
		t.Errorf("Expected transmission resolution to be '2261600', but got '%v'", transmissions[0].Resolution)
	}

	if transmissions[0].Preview != true {
		t.Errorf("Expected transmission preview to be 'true', but got '%v'", transmissions[0].Preview)
	}

	if logData[6]["msg"] != "Transmitted title" {
		t.Errorf("Expected log message 'Transmitted title', but got '%v'", logData[6]["msg"])
	}

	if logData[6]["title"] != "example" {
		t.Errorf("Expected log message title field to be 'example', but got '%v'", logData[6]["title"])
	}

	if logData[6]["resolution"] != 1183600 {
		t.Errorf("Expected log message resolution field to be '1183600', but got '%v'", logData[6]["resolution"])
	}

	if logData[6]["preview"] != true {
		t.Errorf("Expected log message preview field to be 'true', but got '%v'", logData[6]["preview"])
	}

	if logData[6]["streamId"] != "xyz-456" {
		t.Errorf("Expected log message streamId field to be 'xyz-456', but got '%v'", logData[6]["streamId"])
	}

	if transmissions[1].StreamId != "xyz-456" {
		t.Errorf("Expected transmission streamId to be 'xyz-456', but got '%v'", transmissions[1].StreamId)
	}

	if transmissions[1].State != "queued" {
		t.Errorf("Expected transmission state to be 'queued', but got '%v'", transmissions[1].State)
	}

	if transmissions[1].Resolution != 1183600 {
		t.Errorf("Expected transmission resolution to be '1183600', but got '%v'", transmissions[1].Resolution)
	}

	if transmissions[1].Preview != true {
		t.Errorf("Expected transmission preview to be 'true', but got '%v'", transmissions[1].Preview)
	}
}

func TestTransmitPopularTitlesWithFullTransmission(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 70000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{3000000}, []FenixStatusResponse{}, []string{"abc-123", "xyz-456"})
	defer mockFenixServer.Close()

	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		PreviewTransmissionSegments: 4,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600, 2261600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[4]["msg"] != "Transmitted title" {
		t.Errorf("Expected log message 'Transmitted title', but got '%v'", logData[4]["msg"])
	}

	if logData[4]["title"] != "example" {
		t.Errorf("Expected log message title field to be 'example', but got '%v'", logData[4]["title"])
	}

	if logData[4]["resolution"] != 2261600 {
		t.Errorf("Expected log message resolution field to be '2261600', but got '%v'", logData[4]["resolution"])
	}

	if logData[4]["preview"] != false {
		t.Errorf("Expected log message preview field to be 'false', but got '%v'", logData[4]["preview"])
	}

	if logData[4]["streamId"] != "abc-123" {
		t.Errorf("Expected log message streamId field to be 'abc-123', but got '%v'", logData[4]["streamId"])
	}

	if transmissions[0].StreamId != "abc-123" {
		t.Errorf("Expected transmission streamId to be 'abc-123', but got '%v'", transmissions[0].StreamId)
	}

	if transmissions[0].State != "queued" {
		t.Errorf("Expected transmission state to be 'queued', but got '%v'", transmissions[0].State)
	}

	if transmissions[0].Resolution != 2261600 {
		t.Errorf("Expected transmission resolution to be '2261600', but got '%v'", transmissions[0].Resolution)
	}

	if transmissions[0].Preview != false {
		t.Errorf("Expected transmission preview to be 'false', but got '%v'", transmissions[0].Preview)
	}

	if logData[6]["msg"] != "Transmitted title" {
		t.Errorf("Expected log message 'Transmitted title', but got '%v'", logData[6]["msg"])
	}

	if logData[6]["title"] != "example" {
		t.Errorf("Expected log message title field to be 'example', but got '%v'", logData[6]["title"])
	}

	if logData[6]["resolution"] != 1183600 {
		t.Errorf("Expected log message resolution field to be '1183600', but got '%v'", logData[6]["resolution"])
	}

	if logData[6]["preview"] != false {
		t.Errorf("Expected log message preview field to be 'false', but got '%v'", logData[6]["preview"])
	}

	if logData[6]["streamId"] != "xyz-456" {
		t.Errorf("Expected log message streamId field to be 'xyz-456', but got '%v'", logData[6]["streamId"])
	}

	if transmissions[1].StreamId != "xyz-456" {
		t.Errorf("Expected transmission streamId to be 'xyz-456', but got '%v'", transmissions[1].StreamId)
	}

	if transmissions[1].State != "queued" {
		t.Errorf("Expected transmission state to be 'queued', but got '%v'", transmissions[1].State)
	}

	if transmissions[1].Resolution != 1183600 {
		t.Errorf("Expected transmission resolution to be '1183600', but got '%v'", transmissions[1].Resolution)
	}

	if transmissions[1].Preview != false {
		t.Errorf("Expected transmission preview to be 'false', but got '%v'", transmissions[1].Preview)
	}
}

func TestTransmitPopularTitlesWithNoTransmissionMade(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 70000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{3000000}, []FenixStatusResponse{}, []string{""})
	defer mockFenixServer.Close()

	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[4]["msg"] != "No transmission made" {
		t.Errorf("Expected log message 'No transmission made', but got '%v'", logData[4]["msg"])
	}

	if len(transmissions) != 0 {
		t.Errorf("Expected no transmissions to be queued, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithTopItemsLimit(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example1",
			TotalWatchDuration: 70000,
		},
		{
			Title:              "example2",
			TotalWatchDuration: 70000,
		},
		{
			Title:              "example3",
			TotalWatchDuration: 70000,
		},
		{
			Title:              "example4",
			TotalWatchDuration: 70000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{3000000}, []FenixStatusResponse{}, []string{"a", "b", "c", "d"})
	defer mockFenixServer.Close()

	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		TopItems:                    2,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example1": {Title: "example1", Url: "https://example.com/test1.m3u8", Resolutions: []int{1183600}},
			"example2": {Title: "example2", Url: "https://example.com/test2.m3u8", Resolutions: []int{1183600}},
			"example3": {Title: "example3", Url: "https://example.com/test3.m3u8", Resolutions: []int{1183600}},
			"example4": {Title: "example4", Url: "https://example.com/test4.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[10]["msg"] != "Reached iteration limit, pausing further transmissions" {
		t.Errorf("Expected log message 'Reached iteration limit, pausing further transmissions', but got '%v'", logData[10]["msg"])
	}

	if logData[10]["top_items"] != cfg.TopItems {
		t.Errorf("Expected top_items to be %v, but got '%v'", cfg.TopItems, logData[10]["top_items"])
	}

	if len(transmissions) != 2 {
		t.Errorf("Expected 2 transmissions to be queued, but got %v", len(transmissions))
	}

	if transmissions[0].StreamId != "a" {
		t.Errorf("Expected first transmission streamId to be 'a', but got '%v'", transmissions[0].StreamId)
	}

	if transmissions[0].Resolution != 1183600 {
		t.Errorf("Expected first transmission resolution to be 1183600, but got '%v'", transmissions[0].Resolution)
	}

	if transmissions[1].StreamId != "b" {
		t.Errorf("Expected second transmission streamId to be 'b', but got '%v'", transmissions[1].StreamId)
	}

	if transmissions[1].Resolution != 1183600 {
		t.Errorf("Expected second transmission resolution to be 1183600, but got '%v'", transmissions[1].Resolution)
	}
}

func TestTransmitPopularTitlesWithContextCancel(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 70000,
			},
		}),
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 70000,
			},
		}),
	})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{3000000}, []FenixStatusResponse{}, []string{"abc-123", "xyz-456"})
	defer mockFenixServer.Close()

	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		PreviewTransmissionSegments: 4,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  false,
		Sleep:                       1 * time.Millisecond,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600, 2261600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	errs := make(chan error)

	go func() {
		errs <- TransmitPopularTitles(ctx, cfg, &transmissions)
	}()

	time.Sleep(1 * time.Second)
	cancel()
	err := <-errs
	if err.Error() != "context canceled" {
		t.Errorf("Expected context canceled error, but got %v", err)
	}
}

func TestTransmitPopularTitlesWithTransmissionError(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 50000,
		},
	})

	mockServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[3]["msg"] != "Error transmitting popular title" {
		t.Errorf("Expected log message 'Error transmitting popular title', but got '%v'", logData[3]["msg"])
	}

	if len(transmissions) != 0 {
		t.Errorf("Expected no transmissions to be queued, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithNoAvailableBandwidth(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 50000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{20}, []FenixStatusResponse{}, []string{})
	defer mockFenixServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[3]["msg"] != "Not enough satellite bandwidth" {
		t.Errorf("Expected log message 'Not enough satellite bandwidth', but got '%v'", logData[3]["msg"])
	}

	if len(transmissions) != 0 {
		t.Errorf("Expected no transmissions to be queued, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithTitleAlreadyTransmittedForPreview(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 50000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{20}, []FenixStatusResponse{}, []string{})
	defer mockFenixServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		CacheExpiry:                 5 * time.Minute,
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{
		{
			Title:            "example",
			StreamId:         "abc-123",
			State:            "queued",
			Resolution:       1183600,
			Preview:          true,
			TransmissionTime: time.Now(),
		},
	}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[3]["msg"] != "Title already transmitted, skipping" {
		t.Errorf("Expected log message 'Title already transmitted, skipping', but got '%v'", logData[3]["msg"])
	}

	if len(transmissions) != 1 {
		t.Errorf("Expected one transmission, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithTitleAlreadyTransmittedForFullTransmission(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 70000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{20}, []FenixStatusResponse{}, []string{})
	defer mockFenixServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		CacheExpiry:                 5 * time.Minute,
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{
		{
			Title:            "example",
			StreamId:         "abc-123",
			State:            "queued",
			Resolution:       1183600,
			Preview:          false,
			TransmissionTime: time.Now(),
		},
	}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[3]["msg"] != "Title already transmitted, skipping" {
		t.Errorf("Expected log message 'Title already transmitted, skipping', but got '%v'", logData[3]["msg"])
	}

	if len(transmissions) != 1 {
		t.Errorf("Expected one transmission, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesWithTitleAlreadyTransmittedForPreviewAndFullTransmissionAlreadyDone(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 40000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{20}, []FenixStatusResponse{}, []string{})
	defer mockFenixServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		CacheExpiry:                 5 * time.Minute,
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{
		{
			Title:            "example",
			StreamId:         "abc-123",
			State:            "queued",
			Resolution:       1183600,
			Preview:          false,
			TransmissionTime: time.Now(),
		},
	}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[3]["msg"] != "Title already transmitted, skipping" {
		t.Errorf("Expected log message 'Title already transmitted, skipping', but got '%v'", logData[3]["msg"])
	}

	if len(transmissions) != 1 {
		t.Errorf("Expected one transmission, but got %v", len(transmissions))
	}
}

func TestTransmitPopularTitlesForFullTransmissionWithPreviewAlreadyDone(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 70000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{2000000}, []FenixStatusResponse{}, []string{"abc-123"})
	defer mockFenixServer.Close()
	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		CacheExpiry:                 5 * time.Minute,
		AvailableTitles: map[string]types.TitleInfo{
			"example": {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600}},
		},
	}

	transmissions := []types.TitleTransmission{
		{
			Title:            "example",
			StreamId:         "abc-123",
			State:            "queued",
			Resolution:       1183600,
			Preview:          true,
			TransmissionTime: time.Now(),
		},
	}

	err := TransmitPopularTitles(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[4]["msg"] != "Transmitted title" {
		t.Errorf("Expected log message 'Transmitted title', but got '%v'", logData[4]["msg"])
	}

	if len(transmissions) != 2 {
		t.Errorf("Expected two transmissions, but got %v", len(transmissions))
	}

	if transmissions[0].Title != transmissions[1].Title || transmissions[0].Resolution != transmissions[1].Resolution {
		t.Errorf("Expected both transmissions to be for the same title and resolution, but got '%v' and '%v'", transmissions[0].Title, transmissions[1].Title)
	}

	if !transmissions[0].Preview && transmissions[1].Preview {
		t.Errorf("Expected both transmissions to have a different preview status, but got '%v' and '%v'", transmissions[0].Preview, transmissions[1].Preview)
	}
}

func TestFullTransmissionScenario(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signozQueries := []SignozQueryV4Response{
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 40000,
			},
		}),
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 50000,
			},
			{
				Title:              "example2",
				TotalWatchDuration: 70000,
			},
		}),
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 70000,
			},
			{
				Title:              "example2",
				TotalWatchDuration: 80000,
			},
		}),
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 90000,
			},
			{
				Title:              "example2",
				TotalWatchDuration: 40000,
			},
		}),
		newSignozQueryV4Response([]SignozDataResponse{
			{
				Title:              "example",
				TotalWatchDuration: 95000,
			},
			{
				Title:              "example2",
				TotalWatchDuration: 45000,
			},
		}),
	}
	mockSignozServer := runSignozQuery(signozQueries)
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{
		2261650, 100,
		3000000, 3000000, 3000000,
		3000000, 3000000,
		3000000, 3000000, 3000000, 3000000,
	}, []FenixStatusResponse{}, []string{
		"example-preview-highres", "example-preview-lowres",
		"example2-full-highres", "example2-full-lowres",
		"example-full-highres", "example-full-lowres",
		"example-full-highres", "example-full-lowres",
		"example2-preview-highres", "example2-preview-lowres",
	})
	defer mockFenixServer.Close()

	cfg := config.Config{
		SignozUrl:                   mockSignozServer.URL,
		FenixUrl:                    mockFenixServer.URL,
		PreviewTransmissionSegments: 4,
		MinimumPreviewWatchDuration: 30 * time.Second,
		MinimumWatchDuration:        60 * time.Second,
		Breakpoint:                  true,
		CacheExpiry:                 2 * time.Second,
		TopItems:                    10,
		SignozApiVersion:            "v4",
		LogLevel:                    "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example":  {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{1183600, 2261600}},
			"example2": {Title: "example2", Url: "https://example.com/test2.m3u8", Resolutions: []int{1183600, 2261600}},
		},
	}

	transmissions := []types.TitleTransmission{}

	for i := 0; i < len(signozQueries); i++ {
		if i == 4 {
			// Sleep for a duration longer than cache expiry to ensure cache is expired for the next iteration
			time.Sleep(3 * time.Second)
		}
		err := TransmitPopularTitles(ctx, cfg, &transmissions)
		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	/*for i, logEntry := range logData {
		fmt.Printf("[%d] Log entry: %v, Title: %v, Resolution: %v, Preview: %v, StreamId: %v\n", i, logEntry["msg"], logEntry["title"], logEntry["resolution"], logEntry["preview"], logEntry["streamId"])
	}

	for i, transmission := range transmissions {
		fmt.Printf("[%d] Transmission - Title: %v, Resolution: %v, Preview: %v, StreamId: %v\n", i, transmission.Title, transmission.Resolution, transmission.Preview, transmission.StreamId)
	}*/

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if transmissions[0].Title != "example" || transmissions[0].Resolution != 2261600 || !transmissions[0].Preview || transmissions[0].StreamId != "example-preview-highres" {
		t.Errorf("Expected first transmission to be preview of example at 2261600 with streamId 'example-preview-highres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[0].Title, transmissions[0].Resolution, transmissions[0].Preview, transmissions[0].StreamId)
	}

	if logData[5]["msg"] != "Not enough satellite bandwidth" || logData[5]["title"] != "example" || logData[5]["resolution"] != 1183600 {
		t.Errorf("Expected log message 'Not enough satellite bandwidth' with title 'example', resolution 1183600, but got '%v' with title '%v', and resolution '%v'", logData[5]["msg"], logData[5]["title"], logData[5]["resolution"])
	}

	if logData[11]["msg"] != "Title already transmitted, skipping" || logData[11]["title"] != "example" || logData[11]["resolution"] != 2261600 {
		t.Errorf("Expected log message 'Title already transmitted, skipping' with title 'example', resolution 2261600, but got '%v' with title '%v', and resolution '%v'", logData[11]["msg"], logData[11]["title"], logData[11]["resolution"])
	}

	if transmissions[1].Title != "example" || transmissions[1].Resolution != 1183600 || !transmissions[1].Preview || transmissions[1].StreamId != "example-preview-lowres" {
		t.Errorf("Expected second transmission to be preview of example at 1183600 with streamId 'example-preview-lowres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[1].Title, transmissions[1].Resolution, transmissions[1].Preview, transmissions[1].StreamId)
	}

	if transmissions[2].Title != "example2" || transmissions[2].Resolution != 2261600 || transmissions[2].Preview || transmissions[2].StreamId != "example2-full-highres" {
		t.Errorf("Expected third transmission to be full of example2 at 2261600 with streamId 'example2-full-highres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[2].Title, transmissions[2].Resolution, transmissions[2].Preview, transmissions[2].StreamId)
	}

	if transmissions[3].Title != "example2" || transmissions[3].Resolution != 1183600 || transmissions[3].Preview || transmissions[3].StreamId != "example2-full-lowres" {
		t.Errorf("Expected fourth transmission to be full of example2 at 1183600 with streamId 'example2-full-lowres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[3].Title, transmissions[3].Resolution, transmissions[3].Preview, transmissions[3].StreamId)
	}

	if transmissions[4].Title != "example" || transmissions[4].Resolution != 2261600 || transmissions[4].Preview || transmissions[4].StreamId != "example-full-highres" {
		t.Errorf("Expected fifth transmission to be full of example at 2261600 with streamId 'example-full-highres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[4].Title, transmissions[4].Resolution, transmissions[4].Preview, transmissions[4].StreamId)
	}

	if transmissions[5].Title != "example" || transmissions[5].Resolution != 1183600 || transmissions[5].Preview || transmissions[5].StreamId != "example-full-lowres" {
		t.Errorf("Expected sixth transmission to be full of example at 1183600 with streamId 'example-full-lowres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[5].Title, transmissions[5].Resolution, transmissions[5].Preview, transmissions[5].StreamId)
	}

	if logData[27]["msg"] != "Title already transmitted, skipping" || logData[27]["title"] != "example2" || logData[27]["resolution"] != 2261600 {
		t.Errorf("Expected log message 'Title already transmitted, skipping' with title 'example2', resolution 2261600, but got '%v' with title '%v', and resolution '%v'", logData[27]["msg"], logData[27]["title"], logData[27]["resolution"])
	}

	if logData[29]["msg"] != "Title already transmitted, skipping" || logData[29]["title"] != "example2" || logData[29]["resolution"] != 1183600 {
		t.Errorf("Expected log message 'Title already transmitted, skipping' with title 'example2', resolution 1183600, but got '%v' with title '%v', and resolution '%v'", logData[29]["msg"], logData[29]["title"], logData[29]["resolution"])
	}

	if transmissions[6].Title != "example" || transmissions[6].Resolution != 2261600 || transmissions[6].Preview || transmissions[6].StreamId != "example-full-highres" {
		t.Errorf("Expected seventh transmission to be full of example at 2261600 with streamId 'example-full-highres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[6].Title, transmissions[6].Resolution, transmissions[6].Preview, transmissions[6].StreamId)
	}

	if transmissions[7].Title != "example" || transmissions[7].Resolution != 1183600 || transmissions[7].Preview || transmissions[7].StreamId != "example-full-lowres" {
		t.Errorf("Expected eighth transmission to be full of example at 1183600 with streamId 'example-full-lowres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[7].Title, transmissions[7].Resolution, transmissions[7].Preview, transmissions[7].StreamId)
	}

	if transmissions[8].Title != "example2" || transmissions[8].Resolution != 2261600 || !transmissions[8].Preview || transmissions[8].StreamId != "example2-preview-highres" {
		t.Errorf("Expected ninth transmission to be preview of example2 at 2261600 with streamId 'example2-preview-highres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[8].Title, transmissions[8].Resolution, transmissions[8].Preview, transmissions[8].StreamId)
	}

	if transmissions[9].Title != "example2" || transmissions[9].Resolution != 1183600 || !transmissions[9].Preview || transmissions[9].StreamId != "example2-preview-lowres" {
		t.Errorf("Expected tenth transmission to be preview of example2 at 1183600 with streamId 'example2-preview-lowres', but got '%v' at resolution '%v' with preview status '%v' and streamId '%v'", transmissions[9].Title, transmissions[9].Resolution, transmissions[9].Preview, transmissions[9].StreamId)
	}
}

func TestCancelUnpopularTransmissions(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	signozResponse := newSignozQueryV4Response([]SignozDataResponse{
		{
			Title:              "example",
			TotalWatchDuration: 5000,
		},
		{
			Title:              "example3",
			TotalWatchDuration: 12000,
		},
	})

	mockSignozServer := runSignozQuery([]SignozQueryV4Response{signozResponse})
	defer mockSignozServer.Close()
	mockFenixServer := runFenixQuery([]int{2000000}, []FenixStatusResponse{}, []string{"abc-123"})
	defer mockFenixServer.Close()
	cfg := config.Config{
		SignozUrl:                         mockSignozServer.URL,
		FenixUrl:                          mockFenixServer.URL,
		UnpopularityThreshold:             10 * time.Second,
		UnpopularityMaxProgressPercentage: 40,
		UnpopularityMinAge:                5 * time.Minute,
		Breakpoint:                        true,
		SignozApiVersion:                  "v4",
		LogLevel:                          "debug",
		AvailableTitles: map[string]types.TitleInfo{
			"example":  {Title: "example", Url: "https://example.com/test.m3u8", Resolutions: []int{123, 456}},
			"example2": {Title: "example2", Url: "https://example.com/test2.m3u8", Resolutions: []int{123, 456}},
			"example3": {Title: "example3", Url: "https://example.com/test3.m3u8", Resolutions: []int{123, 456}},
			"example4": {Title: "example4", Url: "https://example.com/test4.m3u8", Resolutions: []int{123}},
		},
	}

	transmissions := []types.TitleTransmission{
		{
			Title:                 "example",
			StreamId:              "abc-123",
			State:                 "transmitting",
			Resolution:            123,
			PercentageTransmitted: 30,
			TransmissionTime:      time.Now().Add(-10 * time.Minute),
		},
		{
			Title:                 "example",
			StreamId:              "abc-456",
			State:                 "transmitting",
			Resolution:            456,
			PercentageTransmitted: 50,
			TransmissionTime:      time.Now().Add(-10 * time.Minute),
		},
		{
			Title:                 "example2",
			StreamId:              "abc-789",
			State:                 "transmitting",
			Resolution:            456,
			PercentageTransmitted: 50,
			TransmissionTime:      time.Now().Add(-10 * time.Minute),
		},
		{
			Title:                 "example3",
			StreamId:              "xyz-123",
			State:                 "transmitting",
			Resolution:            123,
			PercentageTransmitted: 50,
			TransmissionTime:      time.Now().Add(-10 * time.Minute),
		},
		{
			Title:                 "example3",
			StreamId:              "xyz-456",
			State:                 "transmitting",
			Resolution:            456,
			PercentageTransmitted: 1,
			TransmissionTime:      time.Now(),
		},
		{
			Title:                 "example4",
			StreamId:              "ex4",
			State:                 "finished",
			Resolution:            123,
			FilesTransmitted:      100,
			FilesTotal:            100,
			PercentageTransmitted: 100,
			TransmissionTime:      time.Now(),
		},
	}

	err := CancelUnpopularTransmissions(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[6]["msg"] != "Transmission progress below unpopularity thresholds, canceling" || logData[6]["title"] != "example" || logData[6]["resolution"] != 123 || logData[6]["streamId"] != "abc-123" {
		t.Errorf("Expected log message 'Transmission progress below unpopularity thresholds, canceling', but got '%v'", logData[6]["msg"])
	}

	if logData[8]["msg"] != "Title still popular enough, not canceling" || logData[8]["title"] != "example" || logData[8]["resolution"] != 456 || logData[8]["streamId"] != "abc-456" {
		t.Errorf("Expected log message 'Title still popular enough, not canceling', but got '%v'", logData[7]["msg"])
	}

	if logData[9]["msg"] != "Title no longer found in list of popular titles, canceling" || logData[9]["title"] != "example2" || logData[9]["resolution"] != 456 || logData[9]["streamId"] != "abc-789" {
		t.Errorf("Expected log message 'Title no longer found in list of popular titles, canceling', but got '%v'", logData[9]["msg"])
	}

	if logData[11]["msg"] != "Title still popular enough, not canceling" || logData[11]["title"] != "example3" || logData[11]["resolution"] != 123 || logData[11]["streamId"] != "xyz-123" {
		t.Errorf("Expected log message 'Title still popular enough, not canceling', but got '%v'", logData[11]["msg"])
	}

	if logData[13]["msg"] != "Title still popular enough, not canceling" || logData[13]["title"] != "example3" || logData[13]["resolution"] != 456 || logData[13]["streamId"] != "xyz-456" {
		t.Errorf("Expected log message 'Title still popular enough, not canceling', but got '%v'", logData[13]["msg"])
	}

	if logData[14]["msg"] != "Title already fully transmitted, skipping" || logData[14]["title"] != "example4" || logData[14]["resolution"] != 123 || logData[14]["streamId"] != "ex4" {
		t.Errorf("Expected log message 'Title already fully transmitted, skipping', but got '%v'", logData[14]["msg"])
	}

	if len(transmissions) != 4 {
		t.Errorf("Expected 4 transmissions, but got %v", len(transmissions))
	}

	if transmissions[0].Title != "example" || transmissions[0].StreamId != "abc-456" || transmissions[0].Resolution != 456 {
		t.Errorf("Expected transmission to be for the title 'example' at resolution 456 with streamId 'abc-456', but got '%v' at resolution '%v' with streamId '%v'", transmissions[0].Title, transmissions[0].Resolution, transmissions[0].StreamId)
	}

	if transmissions[1].Title != "example3" || transmissions[1].StreamId != "xyz-123" {
		t.Errorf("Expected transmission two to be for the title 'example3' with streamId 'xyz-123', but got '%v' with streamId '%v'", transmissions[1].Title, transmissions[1].StreamId)
	}

	if transmissions[2].Title != "example3" || transmissions[2].StreamId != "xyz-456" {
		t.Errorf("Expected transmission three to be for the title 'example3' with streamId 'xyz-456', but got '%v' with streamId '%v'", transmissions[2].Title, transmissions[2].StreamId)
	}
	if transmissions[3].Title != "example4" || transmissions[3].StreamId != "ex4" {
		t.Errorf("Expected transmission four to be for the title 'example4' with streamId 'ex4', but got '%v' with streamId '%v'", transmissions[3].Title, transmissions[3].StreamId)
	}
}

func TestPruneExpiredTransmissions(t *testing.T) {
	var jsonBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&jsonBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.Config{
		CacheExpiry:      1 * time.Hour,
		Breakpoint:       true,
		SignozApiVersion: "v4",
		LogLevel:         "debug",
	}

	transmissions := []types.TitleTransmission{
		{
			Title:            "example",
			StreamId:         "abc-123",
			FilesTransmitted: 10,
			FilesTotal:       10,
			TransmissionTime: time.Now().Add(-24 * time.Hour),
		},
		{
			Title:            "example2",
			StreamId:         "abc-456",
			FilesTransmitted: 10,
			FilesTotal:       10,
			TransmissionTime: time.Now(),
		},
		{
			Title:            "example3",
			StreamId:         "xyz-123",
			FilesTransmitted: 1,
			FilesTotal:       10,
			TransmissionTime: time.Now().Add(-24 * time.Hour),
		},
	}

	err := PruneExpiredTransmissions(ctx, cfg, &transmissions)
	defer cancel()

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	logData, err := parseJsonLogData(&jsonBuffer)

	if err != nil {
		t.Errorf("Failed to parse JSON log data: %v", err)
	}

	if logData[1]["msg"] != "Pruning expired transmission" || logData[1]["title"] != "example" || logData[1]["streamId"] != "abc-123" {
		t.Errorf("Expected log message 'Pruning expired transmission' for title 'example' with streamId 'abc-123', but got '%v'", logData[1]["msg"])
	}

	if logData[4]["msg"] != "Transmission not fully transmitted yet, skipping pruning" || logData[4]["title"] != "example3" || logData[4]["streamId"] != "xyz-123" {
		t.Errorf("Expected log message 'Transmission not fully transmitted yet, skipping pruning' for title 'example3' with streamId 'xyz-123', but got '%v'", logData[4]["msg"])
	}

	if len(transmissions) != 2 {
		t.Errorf("Expected 2 transmissions, but got %v", len(transmissions))
	}

	if transmissions[0].Title != "example2" || transmissions[0].StreamId != "abc-456" {
		t.Errorf("Expected transmission one to be for the title 'example2' with streamId 'abc-456', but got '%v' with streamId '%v'", transmissions[0].Title, transmissions[0].StreamId)
	}
	if transmissions[1].Title != "example3" || transmissions[1].StreamId != "xyz-123" {
		t.Errorf("Expected transmission two to be for the title 'example3' with streamId 'xyz-123', but got '%v' with streamId '%v'", transmissions[1].Title, transmissions[1].StreamId)
	}
}
