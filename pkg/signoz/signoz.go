package signoz

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
)

func formatQueryV5Response(result []byte) ([]any, error) {
	var jsonData any
	if err := json.Unmarshal(result, &jsonData); err != nil {
		return nil, err
	}

	dataMap, ok := jsonData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's JSON root is not an object")
	}

	dataField, ok := dataMap["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's 'data' field is not an object in the JSON output")
	}

	dataData, ok := dataField["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's 'data.data' field is not an object in the JSON output")
	}

	results, ok := dataData["results"].([]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's 'data.data.results' field is not an array	 in the JSON output")
	}

	if len(results) == 0 {
		return results, nil
	}

	firstResult, ok := results[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's first result is not an object")
	}

	dataArray, ok := firstResult["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's 'data' field in first result is not an array")
	}
	return dataArray, nil
}

func formatQueryV4Response(result []byte) ([]any, error) {
	var jsonData any
	if err := json.Unmarshal(result, &jsonData); err != nil {
		return nil, err
	}

	dataMap, ok := jsonData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's JSON root is not an object")
	}

	dataField, ok := dataMap["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's 'data' field is not an object in the JSON output")
	}

	resultField, ok := dataField["result"].([]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's 'data.result' field is not an array	 in the JSON output")
	}

	if len(resultField) == 0 {
		return resultField, nil
	}

	firstResult, ok := resultField[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("Signoz result's first result is not an object")
	}

	list, ok := firstResult["list"].([]any)
	if !ok {
		return []any{}, nil
	}

	if len(list) == 0 {
		return list, nil
	}

	dataArray := make([]any, 0, len(list))

	for _, item := range list {
		itemMap, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Signoz result's item in list is not an object")
		}
		itemDataField, ok := itemMap["data"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Signoz result's 'data' field in item is not an object")
		}
		title, ok := itemDataField["title"].(string)
		if !ok {
			continue
		}
		duration, ok := itemDataField["total_watch_duration"].(float64)
		if !ok {
			duration = 0
		}
		dataArray = append(dataArray, []any{title, duration})

	}

	return dataArray, nil
}

func formatAuthenticationResponse(result []byte) (string, error) {
	var jsonData any
	if err := json.Unmarshal(result, &jsonData); err != nil {
		return "", err
	}

	dataMap, ok := jsonData.(map[string]any)
	if !ok {
		return "", fmt.Errorf("Signoz authentication JSON root is not an object")
	}

	accessJwtField, ok := dataMap["accessJwt"].(string)
	if !ok {
		dataField, ok := dataMap["data"].(map[string]any)
		if !ok {
			return "", fmt.Errorf("Signoz authentication 'data' field is not an object in the JSON output")
		}
		accessJwtField, ok := dataField["accessJwt"].(string)
		if !ok {
			return "", fmt.Errorf("Signoz authentication 'data.accessJwt' field is not a string in the JSON output")
		}
		return accessJwtField, nil
	}
	return accessJwtField, nil
}

func Authenticate(cfg config.Config) (string, error) {
	requestBody := map[string]string{
		"email":    cfg.SignozUsername,
		"password": cfg.SignozPassword,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", cfg.SignozUrl+"/api/v1/login", strings.NewReader(string(jsonBody)))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return "", err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return "", fmt.Errorf("Signoz authentication failed with status %s: %s", resp.Status, body)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	formattedBody, err := formatAuthenticationResponse(body)
	if err != nil {
		return "", err
	}
	return formattedBody, nil
}

func queryV4(cfg config.Config, query string) ([]any, error) {
	jwt, err := Authenticate(cfg)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixNano() / int64(time.Millisecond)
	requestBody := map[string]interface{}{
		"start":     now,
		"end":       now + 1,
		"variables": map[string]interface{}{},
		"compositeQuery": map[string]interface{}{
			"panelType": "list",
			"queryType": "clickhouse_sql",
			"chQueries": map[string]interface{}{
				"popular_video_content": map[string]interface{}{
					"query":    strings.ReplaceAll(strings.ReplaceAll(query, "\n", " "), "\t", " "),
					"disabled": false,
					"name":     "popular_video_content",
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", cfg.SignozUrl+"/api/v4/query_range", strings.NewReader(string(jsonBody)))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("Signoz query failed with status %s: %s", resp.Status, body)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	formattedBody, err := formatQueryV4Response(body)
	if err != nil {
		return nil, err
	}
	return formattedBody, nil
}

func queryV5(cfg config.Config, query string) ([]any, error) {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	requestBody := map[string]interface{}{
		"start":       now,
		"end":         now + 1,
		"requestType": "scalar",
		"variables":   map[string]interface{}{},
		"compositeQuery": map[string]interface{}{
			"queries": []map[string]interface{}{
				{
					"type": "clickhouse_sql",
					"spec": map[string]interface{}{
						"name":  "popular_video_content",
						"query": strings.ReplaceAll(strings.ReplaceAll(query, "\n", " "), "\t", " "),
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", cfg.SignozUrl+"/api/v5/query_range", strings.NewReader(string(jsonBody)))
	req.Header.Set("SIGNOZ-API-KEY", cfg.SignozApiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("Signoz query failed with status %s: %s", resp.Status, body)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	formattedBody, err := formatQueryV5Response(body)
	if err != nil {
		return nil, err
	}
	return formattedBody, nil
}

func Query(cfg config.Config, query string) ([]any, error) {
	switch cfg.SignozApiVersion {
	case "v4":
		return queryV4(cfg, query)
	default:
		return queryV5(cfg, query)
	}
}
