package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.QueryRange != 15*time.Second {
		t.Errorf("expected QueryRange to be 15s, got %s", cfg.QueryRange)
	}
	if cfg.Sleep != 10*time.Second {
		t.Errorf("expected Sleep to be 10s, got %s", cfg.Sleep)
	}
	if cfg.TopItems != 10 {
		t.Errorf("expected TopItems to be 10, got %d", cfg.TopItems)
	}
	if cfg.DisablePrometheus != false {
		t.Errorf("expected DisablePrometheus to be false, got %t", cfg.DisablePrometheus)
	}
	if cfg.MinimumPreviewWatchDuration != 30*time.Second {
		t.Errorf("expected MinimumPreviewWatchDuration to be 30s, got %s", cfg.MinimumPreviewWatchDuration)
	}
	if cfg.PreviewTransmissionSegments != 10 {
		t.Errorf("expected PreviewTransmissionSegments to be 10, got %d", cfg.PreviewTransmissionSegments)
	}
	if cfg.MinimumWatchDuration != 60*time.Second {
		t.Errorf("expected MinimumWatchDuration to be 60s, got %s", cfg.MinimumWatchDuration)
	}
	if cfg.UnpopularityThreshold != 10*time.Second {
		t.Errorf("expected UnpopularityThreshold to be 10s, got %s", cfg.UnpopularityThreshold)
	}
	if cfg.UnpopularityMaxProgressPercentage != 70 {
		t.Errorf("expected UnpopularityMaxProgressPercentage to be 70, got %f", cfg.UnpopularityMaxProgressPercentage)
	}
	if cfg.UnpopularityMinAge != 5*time.Minute {
		t.Errorf("expected UnpopularityMinAge to be 5m, got %s", cfg.UnpopularityMinAge)
	}
	if cfg.CacheExpiry != 5*time.Minute {
		t.Errorf("expected CacheExpiry to be 5m, got %s", cfg.CacheExpiry)
	}
	if cfg.TitlesFile != "titles.json" {
		t.Errorf("expected TitlesFile to be 'titles.json', got '%s'", cfg.TitlesFile)
	}
	if cfg.Breakpoint != false {
		t.Errorf("expected Breakpoint to be false, got %t", cfg.Breakpoint)
	}
	if cfg.FenixUrl != "http://localhost:8888" {
		t.Errorf("expected FenixUrl to be 'http://localhost:8888', got '%s'", cfg.FenixUrl)
	}
	if cfg.FenixApiKey != "" {
		t.Errorf("expected FenixApiKey to be empty, got '%s'", cfg.FenixApiKey)
	}
	if cfg.FenixMockMode != false {
		t.Errorf("expected FenixMockMode to be false, got %t", cfg.FenixMockMode)
	}
	if cfg.FenixMaxBandwidth != 100*megabit {
		t.Errorf("expected FenixMaxBandwidth to be %d, got %d", 100*megabit, cfg.FenixMaxBandwidth)
	}
	if cfg.SignozUrl != "http://localhost:8080" {
		t.Errorf("expected SignozUrl to be 'http://localhost:8080', got '%s'", cfg.SignozUrl)
	}
	if cfg.SignozApiKey != "" {
		t.Errorf("expected SignozApiKey to be empty, got '%s'", cfg.SignozApiKey)
	}
	if cfg.SignozApiVersion != "v4" {
		t.Errorf("expected SignozApiVersion to be 'v4', got '%s'", cfg.SignozApiVersion)
	}
	if cfg.SignozUsername != "" {
		t.Errorf("expected SignozUsername to be empty, got '%s'", cfg.SignozUsername)
	}
	if cfg.SignozPassword != "" {
		t.Errorf("expected SignozPassword to be empty, got '%s'", cfg.SignozPassword)
	}
	if cfg.SignozOtelQueryService != "varnish" {
		t.Errorf("expected SignozOtelQueryService to be 'varnish', got '%s'", cfg.SignozOtelQueryService)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}
	if len(cfg.AvailableTitles) != 15 {
		t.Errorf("expected AvailableTitles to have 15 entries, got %d", len(cfg.AvailableTitles))
	}
	if cfg.OtelMetricsScrapeInterval != 15*time.Second {
		t.Errorf("expected OtelMetricsScrapeInterval to be 15s, got %s", cfg.OtelMetricsScrapeInterval)
	}
}

func TestConfigFromEnvForQueryRange(t *testing.T) {
	os.Setenv("QUERY_RANGE", "1h")
	defer os.Unsetenv("QUERY_RANGE")
	cfg := ConfigFromEnv()
	if cfg.QueryRange != 1*time.Hour {
		t.Errorf("expected QueryRange to be 1h, got %s", cfg.QueryRange)
	}
}

func TestConfigFromEnvForQueryRangeWithError(t *testing.T) {
	os.Setenv("QUERY_RANGE", "invalid")
	defer os.Unsetenv("QUERY_RANGE")
	cfg := ConfigFromEnv()
	if cfg.QueryRange != 15*time.Second {
		t.Errorf("expected QueryRange to be 15s due to error with invalid value, got %s", cfg.QueryRange)
	}
}

func TestConfigFromEnvForSleep(t *testing.T) {
	os.Setenv("SLEEP", "33s")
	defer os.Unsetenv("SLEEP")
	cfg := ConfigFromEnv()
	if cfg.Sleep != 33*time.Second {
		t.Errorf("expected Sleep to be 33s, got %s", cfg.Sleep)
	}
}

func TestConfigFromEnvForSleepWithError(t *testing.T) {
	os.Setenv("SLEEP", "invalid")
	defer os.Unsetenv("SLEEP")
	cfg := ConfigFromEnv()
	if cfg.Sleep != 10*time.Second {
		t.Errorf("expected Sleep to be 10s due to error with invalid value, got %s", cfg.Sleep)
	}
}

func TestConfigFromEnvForTopItems(t *testing.T) {
	os.Setenv("TOP_ITEMS", "33")
	defer os.Unsetenv("TOP_ITEMS")
	cfg := ConfigFromEnv()
	if cfg.TopItems != 33 {
		t.Errorf("expected TopItems to be 33, got %d", cfg.TopItems)
	}
}

func TestConfigFromEnvForTopItemsWithNegativeValue(t *testing.T) {
	os.Setenv("TOP_ITEMS", "-1")
	defer os.Unsetenv("TOP_ITEMS")
	cfg := ConfigFromEnv()
	if cfg.TopItems != 0 {
		t.Errorf("expected TopItems to be 0 due to negative value, got %d", cfg.TopItems)
	}
}

func TestConfigFromEnvForMinimumPreviewWatchDuration(t *testing.T) {
	os.Setenv("MINIMUM_PREVIEW_WATCH_DURATION", "33s")
	defer os.Unsetenv("MINIMUM_PREVIEW_WATCH_DURATION")
	cfg := ConfigFromEnv()
	if cfg.MinimumPreviewWatchDuration != 33*time.Second {
		t.Errorf("expected MinimumPreviewWatchDuration to be 33s, got %s", cfg.MinimumPreviewWatchDuration)
	}
}

func TestConfigFromEnvForMinimumPreviewWatchDurationWithError(t *testing.T) {
	os.Setenv("MINIMUM_PREVIEW_WATCH_DURATION", "invalid")
	defer os.Unsetenv("MINIMUM_PREVIEW_WATCH_DURATION")
	cfg := ConfigFromEnv()
	if cfg.MinimumPreviewWatchDuration != 30*time.Second {
		t.Errorf("expected MinimumPreviewWatchDuration to be 30s due to error with invalid value, got %s", cfg.MinimumPreviewWatchDuration)
	}
}

func TestConfigFromEnvForUnpopularityThreshold(t *testing.T) {
	os.Setenv("UNPOPULARITY_THRESHOLD", "33s")
	defer os.Unsetenv("UNPOPULARITY_THRESHOLD")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityThreshold != 33*time.Second {
		t.Errorf("expected UnpopularityThreshold to be 33s, got %s", cfg.UnpopularityThreshold)
	}
}

func TestConfigFromEnvForUnpopularityThresholdWithError(t *testing.T) {
	os.Setenv("UNPOPULARITY_THRESHOLD", "invalid")
	defer os.Unsetenv("UNPOPULARITY_THRESHOLD")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityThreshold != 10*time.Second {
		t.Errorf("expected UnpopularityThreshold to be 10s due to error with invalid value, got %s", cfg.UnpopularityThreshold)
	}
}

func TestConfigFromEnvForUnpopularityMaxProgressPercentage(t *testing.T) {
	os.Setenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE", "33")
	defer os.Unsetenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityMaxProgressPercentage != 33 {
		t.Errorf("expected UnpopularityMaxProgressPercentage to be 33, got %f", cfg.UnpopularityMaxProgressPercentage)
	}
}

func TestConfigFromEnvForUnpopularityMaxProgressPercentageWithNegativeValue(t *testing.T) {
	os.Setenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE", "-1")
	defer os.Unsetenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityMaxProgressPercentage != 0 {
		t.Errorf("expected UnpopularityMaxProgressPercentage to be 0 due to negative value, got %f", cfg.UnpopularityMaxProgressPercentage)
	}
}

func TestConfigFromEnvForUnpopularityMaxProgressPercentageWithTooHighValue(t *testing.T) {
	os.Setenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE", "101")
	defer os.Unsetenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityMaxProgressPercentage != 100 {
		t.Errorf("expected UnpopularityMaxProgressPercentage to be 100 due to value exceeding 100, got %f", cfg.UnpopularityMaxProgressPercentage)
	}
}

func TestConfigFromEnvForUnpopularityMinAge(t *testing.T) {
	os.Setenv("UNPOPULARITY_MIN_AGE", "33s")
	defer os.Unsetenv("UNPOPULARITY_MIN_AGE")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityMinAge != 33*time.Second {
		t.Errorf("expected UnpopularityMinAge to be 33s, got %s", cfg.UnpopularityMinAge)
	}
}

func TestConfigFromEnvForUnpopularityMinAgeWithError(t *testing.T) {
	os.Setenv("UNPOPULARITY_MIN_AGE", "invalid")
	defer os.Unsetenv("UNPOPULARITY_MIN_AGE")
	cfg := ConfigFromEnv()
	if cfg.UnpopularityMinAge != 5*time.Minute {
		t.Errorf("expected UnpopularityMinAge to be 5m due to error with invalid value, got %s", cfg.UnpopularityMinAge)
	}
}

func TestConfigFromEnvForPreviewTransmissionSegments(t *testing.T) {
	os.Setenv("PREVIEW_TRANSMISSION_SEGMENTS", "33")
	defer os.Unsetenv("PREVIEW_TRANSMISSION_SEGMENTS")
	cfg := ConfigFromEnv()
	if cfg.PreviewTransmissionSegments != 33 {
		t.Errorf("expected PreviewTransmissionSegments to be 33, got %d", cfg.PreviewTransmissionSegments)
	}
}

func TestConfigFromEnvForPreviewTransmissionSegmentsWithNegativeValue(t *testing.T) {
	os.Setenv("PREVIEW_TRANSMISSION_SEGMENTS", "-1")
	defer os.Unsetenv("PREVIEW_TRANSMISSION_SEGMENTS")
	cfg := ConfigFromEnv()
	if cfg.PreviewTransmissionSegments != 0 {
		t.Errorf("expected PreviewTransmissionSegments to be 0 due to negative value, got %d", cfg.PreviewTransmissionSegments)
	}
}

func TestConfigFromEnvForMinimumWatchDuration(t *testing.T) {
	os.Setenv("MINIMUM_WATCH_DURATION", "33s")
	defer os.Unsetenv("MINIMUM_WATCH_DURATION")
	cfg := ConfigFromEnv()
	if cfg.MinimumWatchDuration != 33*time.Second {
		t.Errorf("expected MinimumWatchDuration to be 33s, got %s", cfg.MinimumWatchDuration)
	}
}

func TestConfigFromEnvForMinimumWatchDurationWithError(t *testing.T) {
	os.Setenv("MINIMUM_WATCH_DURATION", "invalid")
	defer os.Unsetenv("MINIMUM_WATCH_DURATION")
	cfg := ConfigFromEnv()
	if cfg.MinimumWatchDuration != 60*time.Second {
		t.Errorf("expected MinimumWatchDuration to be 60s due to error with invalid value, got %s", cfg.MinimumWatchDuration)
	}
}

func TestConfigFromEnvForCacheExpiry(t *testing.T) {
	os.Setenv("CACHE_EXPIRY", "33s")
	defer os.Unsetenv("CACHE_EXPIRY")
	cfg := ConfigFromEnv()
	if cfg.CacheExpiry != 33*time.Second {
		t.Errorf("expected CacheExpiry to be 33s, got %s", cfg.CacheExpiry)
	}
}

func TestConfigFromEnvForCacheExpiryWithError(t *testing.T) {
	os.Setenv("CACHE_EXPIRY", "invalid")
	defer os.Unsetenv("CACHE_EXPIRY")
	cfg := ConfigFromEnv()
	if cfg.CacheExpiry != 5*time.Minute {
		t.Errorf("expected CacheExpiry to be 5m due to error with invalid value, got %s", cfg.CacheExpiry)
	}
}

func TestConfigFromEnvForTitlesFile(t *testing.T) {
	os.Setenv("TITLES_FILE", "custom_titles.json")
	defer os.Unsetenv("TITLES_FILE")
	cfg := ConfigFromEnv()
	if cfg.TitlesFile != "custom_titles.json" {
		t.Errorf("expected TitlesFile to be 'custom_titles.json', got %s", cfg.TitlesFile)
	}
}

func TestConfigFromEnvForTitlesFileWithEmptyValue(t *testing.T) {
	os.Setenv("TITLES_FILE", "")
	defer os.Unsetenv("TITLES_FILE")
	cfg := ConfigFromEnv()
	if cfg.TitlesFile != "titles.json" {
		t.Errorf("expected TitlesFile to be 'titles.json' due to empty value, got %s", cfg.TitlesFile)
	}
}

func TestConfigFromEnvForBreakpointTrue(t *testing.T) {
	os.Setenv("BREAKPOINT", "true")
	defer os.Unsetenv("BREAKPOINT")
	cfg := ConfigFromEnv()
	if !cfg.Breakpoint {
		t.Errorf("expected Breakpoint to be true, got %v", cfg.Breakpoint)
	}
}

func TestConfigFromEnvForBreakpointFalse(t *testing.T) {
	os.Setenv("BREAKPOINT", "false")
	defer os.Unsetenv("BREAKPOINT")
	cfg := ConfigFromEnv()
	if cfg.Breakpoint {
		t.Errorf("expected Breakpoint to be false, got %v", cfg.Breakpoint)
	}
}

func TestConfigFromEnvForBreakpointOne(t *testing.T) {
	os.Setenv("BREAKPOINT", "1")
	defer os.Unsetenv("BREAKPOINT")
	cfg := ConfigFromEnv()
	if !cfg.Breakpoint {
		t.Errorf("expected Breakpoint to be true, got %v", cfg.Breakpoint)
	}
}

func TestConfigFromEnvForBreakpointZero(t *testing.T) {
	os.Setenv("BREAKPOINT", "0")
	defer os.Unsetenv("BREAKPOINT")
	cfg := ConfigFromEnv()
	if cfg.Breakpoint {
		t.Errorf("expected Breakpoint to be false, got %v", cfg.Breakpoint)
	}
}

func TestConfigFromEnvForBreakpointOther(t *testing.T) {
	os.Setenv("BREAKPOINT", "other")
	defer os.Unsetenv("BREAKPOINT")
	cfg := ConfigFromEnv()
	if cfg.Breakpoint {
		t.Errorf("expected Breakpoint to be false, got %v", cfg.Breakpoint)
	}
}

func TestConfigFromEnvForFenixUrl(t *testing.T) {
	os.Setenv("FENIX_URL", "https://custom.fenix.url")
	defer os.Unsetenv("FENIX_URL")
	cfg := ConfigFromEnv()
	if cfg.FenixUrl != "https://custom.fenix.url" {
		t.Errorf("expected FenixUrl to be 'https://custom.fenix.url', got %s", cfg.FenixUrl)
	}
}

func TestConfigFromEnvForFenixUrlWithEmptyValue(t *testing.T) {
	os.Setenv("FENIX_URL", "")
	defer os.Unsetenv("FENIX_URL")
	cfg := ConfigFromEnv()
	if cfg.FenixUrl != "http://localhost:8888" {
		t.Errorf("expected FenixUrl to be 'http://localhost:8888' due to empty value, got %s", cfg.FenixUrl)
	}
}

func TestConfigFromEnvForFenixApiKey(t *testing.T) {
	os.Setenv("FENIX_API_KEY", "custom_api_key")
	defer os.Unsetenv("FENIX_API_KEY")
	cfg := ConfigFromEnv()
	if cfg.FenixApiKey != "custom_api_key" {
		t.Errorf("expected FenixApiKey to be 'custom_api_key', got %s", cfg.FenixApiKey)
	}
}

func TestConfigFromEnvForFenixApiKeyWithEmptyValue(t *testing.T) {
	os.Setenv("FENIX_API_KEY", "")
	defer os.Unsetenv("FENIX_API_KEY")
	cfg := ConfigFromEnv()
	if cfg.FenixApiKey != "" {
		t.Errorf("expected FenixApiKey to be empty, got %s", cfg.FenixApiKey)
	}
}

func TestConfigFromEnvForFenixMockModeTrue(t *testing.T) {
	os.Setenv("FENIX_MOCK_MODE", "true")
	defer os.Unsetenv("FENIX_MOCK_MODE")
	cfg := ConfigFromEnv()
	if !cfg.FenixMockMode {
		t.Errorf("expected FenixMockMode to be true, got %v", cfg.FenixMockMode)
	}
}

func TestConfigFromEnvForFenixMockModeFalse(t *testing.T) {
	os.Setenv("FENIX_MOCK_MODE", "false")
	defer os.Unsetenv("FENIX_MOCK_MODE")
	cfg := ConfigFromEnv()
	if cfg.FenixMockMode {
		t.Errorf("expected FenixMockMode to be false, got %v", cfg.FenixMockMode)
	}
}

func TestConfigFromEnvForFenixMockModeOne(t *testing.T) {
	os.Setenv("FENIX_MOCK_MODE", "1")
	defer os.Unsetenv("FENIX_MOCK_MODE")
	cfg := ConfigFromEnv()
	if !cfg.FenixMockMode {
		t.Errorf("expected FenixMockMode to be true, got %v", cfg.FenixMockMode)
	}
}

func TestConfigFromEnvForFenixMockModeZero(t *testing.T) {
	os.Setenv("FENIX_MOCK_MODE", "0")
	defer os.Unsetenv("FENIX_MOCK_MODE")
	cfg := ConfigFromEnv()
	if cfg.FenixMockMode {
		t.Errorf("expected FenixMockMode to be false, got %v", cfg.FenixMockMode)
	}
}

func TestConfigFromEnvForFenixMockModeOther(t *testing.T) {
	os.Setenv("FENIX_MOCK_MODE", "other")
	defer os.Unsetenv("FENIX_MOCK_MODE")
	cfg := ConfigFromEnv()
	if cfg.FenixMockMode {
		t.Errorf("expected FenixMockMode to be false, got %v", cfg.FenixMockMode)
	}
}

func TestConfigFromEnvForSignozUrl(t *testing.T) {
	os.Setenv("SIGNOZ_URL", "https://custom.signoz.url")
	defer os.Unsetenv("SIGNOZ_URL")
	cfg := ConfigFromEnv()
	if cfg.SignozUrl != "https://custom.signoz.url" {
		t.Errorf("expected SignozUrl to be 'https://custom.signoz.url', got %s", cfg.SignozUrl)
	}
}

func TestConfigFromEnvForSignozUrlWithEmptyValue(t *testing.T) {
	os.Setenv("SIGNOZ_URL", "")
	defer os.Unsetenv("SIGNOZ_URL")
	cfg := ConfigFromEnv()
	if cfg.SignozUrl != "http://localhost:8080" {
		t.Errorf("expected SignozUrl to be 'http://localhost:8080' due to empty value, got %s", cfg.SignozUrl)
	}
}

func TestConfigFromEnvForSignozApiKey(t *testing.T) {
	os.Setenv("SIGNOZ_API_KEY", "custom_api_key")
	defer os.Unsetenv("SIGNOZ_API_KEY")
	cfg := ConfigFromEnv()
	if cfg.SignozApiKey != "custom_api_key" {
		t.Errorf("expected SignozApiKey to be 'custom_api_key', got %s", cfg.SignozApiKey)
	}
}

func TestConfigFromEnvForSignozApiKeyWithEmptyValue(t *testing.T) {
	os.Setenv("SIGNOZ_API_KEY", "")
	defer os.Unsetenv("SIGNOZ_API_KEY")
	cfg := ConfigFromEnv()
	if cfg.SignozApiKey != "" {
		t.Errorf("expected SignozApiKey to be empty due to empty value, got %s", cfg.SignozApiKey)
	}
}

func TestConfigFromEnvForSignozApiVersionV4(t *testing.T) {
	os.Setenv("SIGNOZ_API_VERSION", "v4")
	defer os.Unsetenv("SIGNOZ_API_VERSION")
	cfg := ConfigFromEnv()
	if cfg.SignozApiVersion != "v4" {
		t.Errorf("expected SignozApiVersion to be 'v4', got %s", cfg.SignozApiVersion)
	}
}

func TestConfigFromEnvForSignozApiVersionV5(t *testing.T) {
	os.Setenv("SIGNOZ_API_VERSION", "v5")
	defer os.Unsetenv("SIGNOZ_API_VERSION")
	cfg := ConfigFromEnv()
	if cfg.SignozApiVersion != "v5" {
		t.Errorf("expected SignozApiVersion to be 'v5', got %s", cfg.SignozApiVersion)
	}
}

func TestConfigFromEnvForSignozApiVersionWithInvalidValue(t *testing.T) {
	os.Setenv("SIGNOZ_API_VERSION", "invalid")
	defer os.Unsetenv("SIGNOZ_API_VERSION")
	cfg := ConfigFromEnv()
	if cfg.SignozApiVersion != "v4" {
		t.Errorf("expected SignozApiVersion to be 'v4' due to invalid value, got %s", cfg.SignozApiVersion)
	}
}

func TestConfigFromEnvForSignozApiVersionWithEmptyValue(t *testing.T) {
	os.Setenv("SIGNOZ_API_VERSION", "")
	defer os.Unsetenv("SIGNOZ_API_VERSION")
	cfg := ConfigFromEnv()
	if cfg.SignozApiVersion != "v4" {
		t.Errorf("expected SignozApiVersion to be 'v4' due to empty value, got %s", cfg.SignozApiVersion)
	}
}

func TestConfigFromEnvForSignozUsername(t *testing.T) {
	os.Setenv("SIGNOZ_USERNAME", "custom_username")
	defer os.Unsetenv("SIGNOZ_USERNAME")
	cfg := ConfigFromEnv()
	if cfg.SignozUsername != "custom_username" {
		t.Errorf("expected SignozUsername to be 'custom_username', got %s", cfg.SignozUsername)
	}
}

func TestConfigFromEnvForSignozUsernameWithEmptyValue(t *testing.T) {
	os.Setenv("SIGNOZ_USERNAME", "")
	defer os.Unsetenv("SIGNOZ_USERNAME")
	cfg := ConfigFromEnv()
	if cfg.SignozUsername != "" {
		t.Errorf("expected SignozUsername to be empty due to empty value, got %s", cfg.SignozUsername)
	}
}

func TestConfigFromEnvForSignozPassword(t *testing.T) {
	os.Setenv("SIGNOZ_PASSWORD", "custom_password")
	defer os.Unsetenv("SIGNOZ_PASSWORD")
	cfg := ConfigFromEnv()
	if cfg.SignozPassword != "custom_password" {
		t.Errorf("expected SignozPassword to be 'custom_password', got %s", cfg.SignozPassword)
	}
}

func TestConfigFromEnvForSignozPasswordWithEmptyValue(t *testing.T) {
	os.Setenv("SIGNOZ_PASSWORD", "")
	defer os.Unsetenv("SIGNOZ_PASSWORD")
	cfg := ConfigFromEnv()
	if cfg.SignozPassword != "" {
		t.Errorf("expected SignozPassword to be empty due to empty value, got %s", cfg.SignozPassword)
	}
}

func TestConfigFromEnvForSignozOtelQueryService(t *testing.T) {
	os.Setenv("SIGNOZ_QUERY_SERVICE", "custom_service")
	defer os.Unsetenv("SIGNOZ_QUERY_SERVICE")
	cfg := ConfigFromEnv()
	if cfg.SignozOtelQueryService != "custom_service" {
		t.Errorf("expected SignozOtelQueryService to be 'custom_service', got %s", cfg.SignozOtelQueryService)
	}
}

func TestConfigFromEnvForSignozOtelQueryServiceWithEmptyValue(t *testing.T) {
	os.Setenv("SIGNOZ_QUERY_SERVICE", "")
	defer os.Unsetenv("SIGNOZ_QUERY_SERVICE")
	cfg := ConfigFromEnv()
	if cfg.SignozOtelQueryService != "varnish" {
		t.Errorf("expected SignozOtelQueryService to be 'varnish' due to empty value, got %s", cfg.SignozOtelQueryService)
	}
}

func TestConfigFromEnvForLogLevel(t *testing.T) {
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")
	cfg := ConfigFromEnv()
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel to be 'debug', got %s", cfg.LogLevel)
	}
}

func TestConfigFromEnvForLogLevelWithEmptyValue(t *testing.T) {
	os.Setenv("LOG_LEVEL", "")
	defer os.Unsetenv("LOG_LEVEL")
	cfg := ConfigFromEnv()
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel to be 'info' due to empty value, got %s", cfg.LogLevel)
	}
}

func TestConfigFromEnvForOtelMetricsScrapeInterval(t *testing.T) {
	os.Setenv("OTEL_METRIC_SCRAPING_INTERVAL", "1h")
	defer os.Unsetenv("OTEL_METRIC_SCRAPING_INTERVAL")
	cfg := ConfigFromEnv()
	if cfg.OtelMetricsScrapeInterval != 1*time.Hour {
		t.Errorf("expected OtelMetricsScrapeInterval to be 1h, got %s", cfg.OtelMetricsScrapeInterval)
	}
}

func TestConfigFromEnvForOtelMetricsScrapeIntervalWithError(t *testing.T) {
	os.Setenv("OTEL_METRIC_SCRAPING_INTERVAL", "invalid")
	defer os.Unsetenv("OTEL_METRIC_SCRAPING_INTERVAL")
	cfg := ConfigFromEnv()
	if cfg.OtelMetricsScrapeInterval != 15*time.Second {
		t.Errorf("expected OtelMetricsScrapeInterval to be 15s due to error with invalid value, got %s", cfg.OtelMetricsScrapeInterval)
	}
}
