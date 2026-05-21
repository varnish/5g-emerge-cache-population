package fenix

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

func apiGetClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return client
}

func apiRequest(cfg config.Config, endpoint string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", cfg.FenixUrl+endpoint, nil)
	req.Header.Set("FENIX-API-KEY", cfg.FenixApiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return nil, err
	}

	client := apiGetClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected response status from server: %s (%d)- %s", resp.Status, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot read response body")
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, err
	}
	return respData, nil
}

func apiDeletionRequest(cfg config.Config, endpoint string) error {
	req, err := http.NewRequest("DELETE", cfg.FenixUrl+endpoint, nil)
	req.Header.Set("FENIX-API-KEY", cfg.FenixApiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return err
	}

	client := apiGetClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected response status from server: %s (%d)- %s", resp.Status, resp.StatusCode, string(body))
	}

	return nil
}

func apiTransmissionRequest(cfg config.Config, url string, resolution int, startSegment int, endSegment int) (map[string]interface{}, error) {

	payload := map[string]interface{}{
		"sourceUrl": url,
		"bandwidth": resolution,
	}

	if startSegment > 0 || endSegment > startSegment {
		payload["previewWindow"] = map[string]interface{}{
			"startSegment": startSegment,
			"endSegment":   endSegment,
		}
	}

	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", cfg.FenixUrl+"/streams", strings.NewReader(string(jsonData)))
	req.Header.Set("FENIX-API-KEY", cfg.FenixApiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return nil, err
	}

	client := apiGetClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Unexpected response from server: %s (%d)- %s", resp.Status, resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Cannot read response body")
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, err
	}
	return respData, nil
}

func GetBandwidthAvailable(cfg config.Config) (int, error) {
	if cfg.FenixMockMode {
		return rand.Intn(cfg.FenixMaxBandwidth), nil
	}

	respData, err := apiRequest(cfg, "/bandwidth")
	if err != nil {
		return 0, err
	}

	bandwidthAvailable, ok := respData["bandwidthAvailable"].(float64)
	if !ok {
		return 0, fmt.Errorf("bandwidth not found in response: %v", respData)
	}

	if bandwidthAvailable < 0 {
		bandwidthAvailable = 0
	}
	return int(bandwidthAvailable), nil
}

func GetTransmissionStatus(cfg config.Config, streamId string) (string, int, int, error) {
	if cfg.FenixMockMode {
		randomInt := rand.Intn(101)
		state := "queued"
		if randomInt == 100 {
			state = "finished"
		} else if randomInt > 0 {
			state = "ongoing"
		}
		return state, randomInt, 100, nil
	}

	respData, err := apiRequest(cfg, "/status/"+streamId)
	if err != nil {
		return "", 0, 0, err
	}

	state, ok := respData["state"].(string)
	if !ok {
		return "", 0, 0, fmt.Errorf("state property not found in response: %v", respData)
	}

	progress, ok := respData["progress"].(map[string]any)
	if !ok {
		return state, 0, 0, fmt.Errorf("progress property not found in response: %v", respData)
	}

	filesTransmitted, ok := progress["filesTransmitted"].(float64)
	if !ok {
		return state, 0, 0, fmt.Errorf("filesTransmitted property not found in response: %v", respData)
	}

	filesTotal, ok := progress["filesTotal"].(float64)
	if !ok {
		filesTotal = 0
	}
	mode, ok := respData["mode"].(string)
	if !ok {
		mode = "full"
	}

	if mode == "preview" {
		filesTotal = float64(cfg.PreviewTransmissionSegments)
	}

	return state, int(filesTransmitted), int(filesTotal), nil
}

func Transmit(cfg config.Config, title types.TitleInfo, resolution int) (string, error) {
	if cfg.FenixMockMode {
		return fmt.Sprintf("mock-stream-%s-%d", title.Title, resolution), nil
	}

	var respData map[string]interface{}
	var err error
	if title.Preview {
		startSegment := 0
		endSegment := cfg.PreviewTransmissionSegments - 1
		respData, err = apiTransmissionRequest(cfg, title.Url, resolution, startSegment, endSegment)
	} else {
		respData, err = apiTransmissionRequest(cfg, title.Url, resolution, 0, 0)
	}

	if err != nil {
		return "", err
	}

	streamID, ok := respData["streamId"].(string)
	if !ok {
		return "", fmt.Errorf("streamId not found in response: %v", respData)
	}

	return streamID, nil
}

func DeleteTransmission(cfg config.Config, streamId string) error {
	if cfg.FenixMockMode {
		return nil
	}

	err := apiDeletionRequest(cfg, "/streams/"+streamId)
	if err != nil {
		return err
	}
	return nil
}
