package config

import (
	"os"
	"strconv"
	"time"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

const (
	megabit = 1000000 // 1 Mbps in bits per second
)

type Config struct {
	QueryRange                        time.Duration
	Sleep                             time.Duration
	TopItems                          int
	DisablePrometheus                 bool
	MinimumPreviewWatchDuration       time.Duration
	PreviewTransmissionSegments       int
	MinimumWatchDuration              time.Duration
	UnpopularityThreshold             time.Duration
	UnpopularityMaxProgressPercentage float64
	UnpopularityMinAge                time.Duration
	CacheExpiry                       time.Duration
	TitlesFile                        string
	Breakpoint                        bool
	FenixUrl                          string
	FenixApiKey                       string
	FenixMockMode                     bool
	FenixMaxBandwidth                 int
	SignozUrl                         string
	SignozApiKey                      string
	SignozApiVersion                  string
	SignozUsername                    string
	SignozPassword                    string
	SignozOtelQueryService            string
	LogLevel                          string
	AvailableTitles                   map[string]types.TitleInfo
	OtelMetricsScrapeInterval         time.Duration
	ApiServerPort                     int
}

func defaultConfig() *Config {
	return &Config{
		QueryRange:                        15 * time.Second,
		Sleep:                             10 * time.Second,
		TopItems:                          10,
		DisablePrometheus:                 false,
		MinimumPreviewWatchDuration:       30 * time.Second,
		PreviewTransmissionSegments:       10,
		MinimumWatchDuration:              60 * time.Second,
		UnpopularityThreshold:             10 * time.Second,
		UnpopularityMaxProgressPercentage: 70,
		UnpopularityMinAge:                5 * time.Minute,
		CacheExpiry:                       5 * time.Minute,
		TitlesFile:                        "titles.json",
		Breakpoint:                        false,
		FenixUrl:                          "http://localhost:8888",
		FenixApiKey:                       "",
		FenixMockMode:                     false,
		FenixMaxBandwidth:                 100 * megabit,
		SignozUrl:                         "http://localhost:8080",
		SignozApiKey:                      "",
		SignozApiVersion:                  "v4",
		SignozUsername:                    "",
		SignozPassword:                    "",
		SignozOtelQueryService:            "varnish",
		LogLevel:                          "info",
		AvailableTitles: map[string]types.TitleInfo{
			"5GBroadcast":        {Title: "5GBroadcast", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5GBroadcast/5GBroadcast.m3u8", Resolutions: []int{1183600, 2261600}},
			"5g-records":         {Title: "5g-records", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5g-records/5g-records.m3u8", Resolutions: []int{1183600, 2261600}},
			"5g-tours":           {Title: "5g-tours", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5g-tours/5g-tours.m3u8", Resolutions: []int{1183600, 2261600}},
			"SGiovanni":          {Title: "SGiovanni", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/SGiovanni/SGiovanni.m3u8", Resolutions: []int{1183600, 2261600}},
			"TorinoJazzFestival": {Title: "TorinoJazzFestival", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/TorinoJazzFestival/TorinoJazzFestival.m3u8", Resolutions: []int{1183600, 2261600}},
			"5G-Emerge":          {Title: "5G-Emerge", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5G-Emerge/5G-Emerge.m3u8", Resolutions: []int{1183600, 2261600}},
			"5G-MAG":             {Title: "5G-MAG", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/5G-MAG/5G-MAG.m3u8"},
			"CustomizedAI":       {Title: "CustomizedAI", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/CustomizedAI/CustomizedAI.m3u8", Resolutions: []int{1183600, 2261600}},
			"DVB-I":              {Title: "DVB-I", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/DVB-I/DVB-I.m3u8", Resolutions: []int{1183600, 2261600}},
			"DynamicMedia":       {Title: "DynamicMedia", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/DynamicMedia/DynamicMedia.m3u8", Resolutions: []int{1183600, 2261600}},
			"EBU2023":            {Title: "EBU2023", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/EBU2023/EBU2023.m3u8", Resolutions: []int{1183600, 2261600}},
			"HQVideo":            {Title: "HQVideo", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/HQVideo/HQVideo.m3u8", Resolutions: []int{1183600, 2261600}},
			"NeoEurovox":         {Title: "NeoEurovox", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/NeoEurovox/NeoEurovox.m3u8", Resolutions: []int{1183600, 2261600}},
			"Peach":              {Title: "Peach", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/Peach/Peach.m3u8", Resolutions: []int{1183600, 2261600}},
			"STADIEM":            {Title: "STADIEM", Url: "https://rai.gcdn.co/5G-EMERGE/VOD/hls/STADIEM/STADIEM.m3u8", Resolutions: []int{1183600, 2261600}},
		},
		OtelMetricsScrapeInterval: 15 * time.Second,
		ApiServerPort:             8888,
	}
}

func ConfigFromEnv() *Config {
	cfg := defaultConfig()

	if val, err := time.ParseDuration(os.Getenv("QUERY_RANGE")); err == nil {
		cfg.QueryRange = val
	}

	if val, err := time.ParseDuration(os.Getenv("SLEEP")); err == nil {
		cfg.Sleep = val
	}

	if val := os.Getenv("TOP_ITEMS"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			if intVal < 0 {
				intVal = 0
			}
			cfg.TopItems = intVal
		}
	}

	if val, err := time.ParseDuration(os.Getenv("MINIMUM_PREVIEW_WATCH_DURATION")); err == nil {
		cfg.MinimumPreviewWatchDuration = val
	}

	if val, err := time.ParseDuration(os.Getenv("UNPOPULARITY_THRESHOLD")); err == nil {
		cfg.UnpopularityThreshold = val
	}

	if val := os.Getenv("UNPOPULARITY_MAX_PROGRESS_PERCENTAGE"); val != "" {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			if floatVal < 0 {
				floatVal = 0
			} else if floatVal > 100 {
				floatVal = 100
			}
			cfg.UnpopularityMaxProgressPercentage = floatVal
		}
	}

	if val, err := time.ParseDuration(os.Getenv("UNPOPULARITY_MIN_AGE")); err == nil {
		cfg.UnpopularityMinAge = val
	}

	if val := os.Getenv("PREVIEW_TRANSMISSION_SEGMENTS"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			if intVal < 0 {
				intVal = 0
			}
			cfg.PreviewTransmissionSegments = intVal
		}
	}

	if val, err := time.ParseDuration(os.Getenv("MINIMUM_WATCH_DURATION")); err == nil {
		cfg.MinimumWatchDuration = val
	}

	if val, err := time.ParseDuration(os.Getenv("CACHE_EXPIRY")); err == nil {
		cfg.CacheExpiry = val
	}

	if val := os.Getenv("TITLES_FILE"); val != "" {
		cfg.TitlesFile = val
	}

	if val := os.Getenv("BREAKPOINT"); val == "true" || val == "1" {
		cfg.Breakpoint = true
	}

	if val := os.Getenv("FENIX_URL"); val != "" {
		cfg.FenixUrl = val
	}

	if val := os.Getenv("FENIX_API_KEY"); val != "" {
		cfg.FenixApiKey = val
	}

	if val := os.Getenv("FENIX_MOCK_MODE"); val == "true" || val == "1" {
		cfg.FenixMockMode = true
	}

	if val := os.Getenv("SIGNOZ_URL"); val != "" {
		cfg.SignozUrl = val
	}

	if val := os.Getenv("SIGNOZ_API_KEY"); val != "" {
		cfg.SignozApiKey = val
	}

	if val := os.Getenv("SIGNOZ_API_VERSION"); val != "" {
		switch val {
		case "v4", "v5":
			// valid versions
		default:
			val = "v4" // default to v4 if an invalid version is provided
		}
		cfg.SignozApiVersion = val
	}

	if val := os.Getenv("SIGNOZ_USERNAME"); val != "" {
		cfg.SignozUsername = val
	}

	if val := os.Getenv("SIGNOZ_PASSWORD"); val != "" {
		cfg.SignozPassword = val
	}

	if val := os.Getenv("SIGNOZ_QUERY_SERVICE"); val != "" {
		cfg.SignozOtelQueryService = val
	}

	if val := os.Getenv("LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	if val, err := time.ParseDuration(os.Getenv("OTEL_METRIC_SCRAPING_INTERVAL")); err == nil {
		cfg.OtelMetricsScrapeInterval = val
	}

	if val := os.Getenv("API_SERVER_PORT"); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			if intVal < 0 {
				intVal = 0
			}
			cfg.ApiServerPort = intVal
		}
	}

	return cfg
}
