package titles

import (
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/clickhouse"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/signoz"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

func LoadTitles(titlesFile string) (map[string]types.TitleInfo, error) {
	slog.Debug("Loading titles from file", "task", "LoadTitles", "titles_file", titlesFile)
	if titlesFile == "" {
		return nil, nil
	}

	// Try to load from JSON file
	file, err := os.ReadFile(titlesFile)
	if err != nil {
		return nil, err
	}

	var titles map[string]types.TitleInfo
	err = json.Unmarshal(file, &titles)
	if err != nil {
		return nil, err
	}

	for k, v := range titles {
		if len(v.Resolutions) > 1 {
			sort.Sort(sort.Reverse(sort.IntSlice(v.Resolutions)))
			titles[k] = v
		}
	}
	return titles, nil
}

func parseTitles(cfg config.Config, dataArray []any) map[string]types.TitleInfo {
	titles := make(map[string]types.TitleInfo)
	for _, row := range dataArray {
		rowArray, ok := row.([]any)
		if !ok || len(rowArray) < 2 {
			slog.Warn("Invalid row format, skipping", "row", row)
			continue
		}

		title, ok := rowArray[0].(string)
		if !ok {
			slog.Warn("Title is not a string, skipping", "title", rowArray[0])
			continue
		}

		var watchDurationMs int64
		switch v := rowArray[1].(type) {
		case float64:
			watchDurationMs = int64(v)
		case int64:
			watchDurationMs = v
		default:
			slog.Warn("Watch duration is not a number for title '%s', skipping", "title", title, "watch_duration", rowArray[1])
			continue
		}

		titleInfo := types.TitleInfo{}
		titleInfo.Title = title
		titleInfo.WatchDuration = time.Duration(watchDurationMs) * time.Millisecond
		slog.Info("Parsed log entry", "title", titleInfo.Title, "watch_duration_seconds", titleInfo.WatchDuration.Seconds())

		titleInfoExtra, exists := cfg.AvailableTitles[titleInfo.Title]
		if !exists {
			slog.Warn("Title not found in predefined titles, skipping", "title", titleInfo.Title)
			continue
		}

		titleInfo.Url = titleInfoExtra.Url
		titleInfo.Resolutions = titleInfoExtra.Resolutions
		sort.Sort(sort.Reverse(sort.IntSlice(titleInfo.Resolutions)))
		titleInfo.Preview = true

		if titleInfo.WatchDuration >= cfg.MinimumPreviewWatchDuration {
			titleInfo.Preview = false
		}
		titles[titleInfo.Title] = titleInfo
	}
	return titles
}

func GetTitlesWithWatchDuration(cfg config.Config) (map[string]types.TitleInfo, error) {
	slog.Debug("Fetching title popularity from Signoz", "task", "GetTitlesWithWatchDuration")
	dataArray, err := signoz.Query(cfg, clickhouse.GetSql(cfg))
	if err != nil {
		return nil, err
	}
	titles := parseTitles(cfg, dataArray)
	slog.Debug("Fetched popular titles from Signoz", "task", "GetTitlesWithWatchDuration", "numTitles", len(titles))
	return titles, nil
}
