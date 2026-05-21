package transmission

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/fenix"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/titles"
	types "github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

func InitTransmission(title string, streamId string, resolution int, preview bool) types.TitleTransmission {
	transmission := types.TitleTransmission{
		Title:                 title,
		StreamId:              streamId,
		State:                 "queued",
		Resolution:            resolution,
		FilesTransmitted:      0,
		FilesTotal:            0,
		PercentageTransmitted: 0.0,
		Preview:               preview,
	}

	return transmission
}

func getTransmissionByTitleAndResolution(transmissions []types.TitleTransmission, title string, resolution int, preview bool) *types.TitleTransmission {

	transactionIndex := -1
	previewtoFull := false
	for i, transmission := range transmissions {
		if transmission.Title == title && transmission.Resolution == resolution {
			transactionIndex = i
			if !preview && transmission.Preview {
				previewtoFull = true
			} else {
				previewtoFull = false
			}
		}
	}
	if transactionIndex > -1 && !previewtoFull {
		return &transmissions[transactionIndex]
	}
	return nil
}

func UpdateTransmissionState(ctx context.Context, cfg config.Config, transmissions *[]types.TitleTransmission) error {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("UpdateTransmissionState: context cancelled, shutting down", "task", "UpdateTransmissionState")
			return ctx.Err()
		default:
		}

		for i, transmission := range *transmissions {
			slog.Debug("Inspecting transmission", "task", "UpdateTransmissionState", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId, "state", transmission.State, "preview", transmission.Preview, "filesTransmitted", transmission.FilesTransmitted, "filesTotal", transmission.FilesTotal)
			//if transmission.State == "queued" || transmission.State == "ongoing" || transmission.FilesTransmitted == 0 || transmission.FilesTotal == 0 || transmission.FilesTransmitted < transmission.FilesTotal {
			if transmission.FilesTransmitted == 0 || transmission.FilesTotal == 0 || transmission.FilesTransmitted < transmission.FilesTotal {
				state, filesTransmitted, filesTotal, err := fenix.GetTransmissionStatus(cfg, transmission.StreamId)
				if err != nil {
					slog.Error("Error getting transmission status", "task", "UpdateTransmissionState", "streamId", transmission.StreamId, "error", err)
					continue
				}

				changed := state != transmission.State || filesTransmitted != transmission.FilesTransmitted || filesTotal != transmission.FilesTotal
				if !changed {
					slog.Debug("No change in transmission status", "task", "UpdateTransmissionState", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId, "state", transmission.State, "filesTransmitted", transmission.FilesTransmitted, "filesTotal", transmission.FilesTotal)
					continue
				}

				transmission.State = state
				transmission.FilesTransmitted = filesTransmitted
				transmission.FilesTotal = filesTotal
				if filesTotal > 0 {
					transmission.PercentageTransmitted = math.Round((float64(filesTransmitted) / float64(filesTotal)) * 100.0)
				} else {
					transmission.PercentageTransmitted = 0.0
				}

				(*transmissions)[i] = transmission
				slog.Info("Updated transmission status", "action", "update", "task", "UpdateTransmissionState", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId, "state", transmission.State, "filesTransmitted", filesTransmitted, "filesTotal", filesTotal, "percentageTransmitted", transmission.PercentageTransmitted)
			}
		}
		if cfg.Breakpoint {
			return nil
		}
		slog.Debug("Sleeping", "task", "UpdateTransmissionState", "duration", cfg.Sleep)
		time.Sleep(cfg.Sleep)
	}
}

func TransmitPopularTitles(ctx context.Context, cfg config.Config, transmissions *[]types.TitleTransmission) error {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("TransmitPopularTitles: context cancelled, shutting down", "task", "TransmitPopularTitles")
			return ctx.Err()
		default:
		}

		titles, err := titles.GetTitlesWithWatchDuration(cfg)
		if err != nil {
			slog.Error("Error getting popular titles", "task", "TransmitPopularTitles", "error", err)
			return err
		}

		index := 1
		for _, titleInfo := range titles {
			if titleInfo.WatchDuration < cfg.MinimumPreviewWatchDuration {
				slog.Debug("Title watch duration below minimum preview watch duration, skipping", "task", "TransmitPopularTitles", "title", titleInfo.Title, "watch_duration_seconds", titleInfo.WatchDuration.Seconds())
				continue
			} else if titleInfo.WatchDuration < cfg.MinimumWatchDuration {
				titleInfo.Preview = true
			} else {
				titleInfo.Preview = false
			}

			for _, resolution := range titleInfo.Resolutions {
				streamId, err := transmitPopularTitle(cfg, titleInfo, resolution, transmissions)
				if err != nil {
					slog.Error("Error transmitting popular title", "task", "TransmitPopularTitles", "title", titleInfo.Title, "resolution", resolution, "error", err)
					continue
				}
				if streamId == "" {
					slog.Warn("No transmission made", "task", "TransmitPopularTitles", "title", titleInfo.Title, "resolution", resolution)
					continue
				}
				index++
				slog.Info("Transmitted title", "action", "transmit", "task", "TransmitPopularTitles", "title", titleInfo.Title, "resolution", resolution, "preview", titleInfo.Preview, "streamId", streamId)
			}
			if index > cfg.TopItems {
				slog.Debug("Reached iteration limit, pausing further transmissions", "task", "TransmitPopularTitles", "top_items", cfg.TopItems)
				break
			}
		}
		if cfg.Breakpoint {
			return nil
		}
		slog.Debug("Sleeping", "task", "TransmitPopularTitles", "duration", cfg.Sleep)
		time.Sleep(cfg.Sleep)
	}
}

func transmitPopularTitle(cfg config.Config, titleInfo types.TitleInfo, resolution int, transmissions *[]types.TitleTransmission) (string, error) {
	transmission := getTransmissionByTitleAndResolution(*transmissions, titleInfo.Title, resolution, titleInfo.Preview)

	if transmission != nil && time.Since(transmission.TransmissionTime) < cfg.CacheExpiry {
		slog.Debug("Title already transmitted, skipping", "task", "TransmitPopularTitle", "title", titleInfo.Title, "resolution", resolution, "streamId", transmission.StreamId, "preview", titleInfo.Preview)
		return "", nil
	}

	availableBandwidth, err := fenix.GetBandwidthAvailable(cfg)
	if err != nil {
		return "", err
	}

	if resolution > availableBandwidth {
		slog.Warn("Not enough satellite bandwidth", "task", "TransmitPopularTitle", "title", titleInfo.Title, "resolution", resolution, "availableBandwidth", availableBandwidth)
		return "", nil
	}

	slog.Debug("Sufficient satellite bandwidth", "task", "TransmitPopularTitle", "title", titleInfo.Title, "resolution", resolution, "availableBandwidth", availableBandwidth)

	streamId, err := fenix.Transmit(cfg, titleInfo, resolution)
	if err != nil {
		return "", err
	}

	if streamId == "" {
		return "", nil
	}

	newTransmission := InitTransmission(titleInfo.Title, streamId, resolution, titleInfo.Preview)
	newTransmission.TransmissionTime = time.Now()
	*transmissions = append(*transmissions, newTransmission)
	return streamId, nil
}

func CancelUnpopularTransmissions(ctx context.Context, cfg config.Config, transmissions *[]types.TitleTransmission) error {
	slog.Debug("Cancel unpopular transmissions", "action", "evaluate", "task", "CancelUnpopularTransmissions")
	for {
		select {
		case <-ctx.Done():
			slog.Debug("CancelUnpopularTransmissions: context cancelled, shutting down", "task", "CancelUnpopularTransmissions")
			return ctx.Err()
		default:
		}

		titles, err := titles.GetTitlesWithWatchDuration(cfg)
		if err != nil {
			slog.Error("Error getting popular titles", "task", "CancelUnpopularTransmissions", "error", err)
			return err
		}

		for _, transmission := range *transmissions {
			var title types.TitleInfo
			found := false
			for _, titleItem := range titles {
				if transmission.Title == titleItem.Title && containsResolution(titleItem.Resolutions, transmission.Resolution) {
					title = titleItem
					found = true
					slog.Debug("Found title in list of popular titles", "task", "CancelUnpopularTransmissions", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId, "watch_duration_seconds", title.WatchDuration.Seconds(), "unpopularity_threshold_seconds", cfg.UnpopularityThreshold.Seconds(), "percentageTransmitted", transmission.PercentageTransmitted, "unpopularity_max_progress_percentage", cfg.UnpopularityMaxProgressPercentage)
					break
				}
			}

			if transmission.FilesTransmitted > 0 && transmission.FilesTransmitted == transmission.FilesTotal {
				slog.Debug("Title already fully transmitted, skipping", "task", "CancelUnpopularTransmissions", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId, "filesTransmitted", transmission.FilesTransmitted, "filesTotal", transmission.FilesTotal)
				continue
			}

			if found && (title.WatchDuration >= cfg.UnpopularityThreshold || transmission.PercentageTransmitted > cfg.UnpopularityMaxProgressPercentage || time.Since(transmission.TransmissionTime) < cfg.UnpopularityMinAge) {
				slog.Debug("Title still popular enough, not canceling", "task", "CancelUnpopularTransmissions", "title", transmission.Title, "resolution",
					transmission.Resolution, "streamId", transmission.StreamId, "watch_duration_seconds", title.WatchDuration.Seconds(), "unpopularity_threshold_seconds", cfg.UnpopularityThreshold.Seconds(), "percentageTransmitted", transmission.PercentageTransmitted, "unpopularity_max_progress_percentage", cfg.UnpopularityMaxProgressPercentage, "time_since_transmission_seconds", time.Since(transmission.TransmissionTime).Seconds(), "unpopularity_min_age_seconds", cfg.UnpopularityMinAge.Seconds())
				continue
			}

			if !found {
				slog.Info("Title no longer found in list of popular titles, canceling", "task", "CancelUnpopularTransmissions", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId)
			} else {
				slog.Info("Transmission progress below unpopularity thresholds, canceling", "task", "CancelUnpopularTransmissions", "title", transmission.Title, "resolution",
					transmission.Resolution, "streamId", transmission.StreamId, "watch_duration_seconds", title.WatchDuration.Seconds(), "unpopularity_threshold_seconds", cfg.UnpopularityThreshold.Seconds(), "percentageTransmitted", transmission.PercentageTransmitted, "unpopularity_max_progress_percentage", cfg.UnpopularityMaxProgressPercentage, "time_since_transmission_seconds", time.Since(transmission.TransmissionTime).Seconds(), "unpopularity_min_age_seconds", cfg.UnpopularityMinAge.Seconds())
			}

			err := fenix.DeleteTransmission(cfg, transmission.StreamId)
			if err != nil {
				return err
			}
			for i, t := range *transmissions {
				if t.Title == transmission.Title && t.Resolution == transmission.Resolution {
					(*transmissions)[i].State = "canceled"
					break
				}
			}
		}

		for i := len(*transmissions) - 1; i >= 0; i-- {
			if (*transmissions)[i].State == "canceled" {
				*transmissions = append((*transmissions)[:i], (*transmissions)[i+1:]...)
			}
		}

		if cfg.Breakpoint {
			return nil
		}
		slog.Debug("Sleeping", "task", "CancelUnpopularTransmissions", "duration", cfg.Sleep)
		time.Sleep(cfg.Sleep)
	}
}

func containsResolution(resolutions []int, resolution int) bool {
	for _, r := range resolutions {
		if r == resolution {
			return true
		}
	}
	return false
}

func PruneExpiredTransmissions(ctx context.Context, cfg config.Config, transmissions *[]types.TitleTransmission) error {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("PruneExpiredTransmissions: context cancelled, shutting down", "task", "PruneExpiredTransmissions")
			return ctx.Err()
		default:
		}
		for _, transmission := range *transmissions {
			slog.Debug("Check transmission before pruning", "action", "prune", "task", "PruneExpiredTransmissions", "title", transmission.Title, "resolution", transmission.Resolution, "preview", transmission.Preview, "streamId", transmission.StreamId, "timeLeftSeconds", cfg.CacheExpiry.Seconds()-time.Since(transmission.TransmissionTime).Seconds())
			if transmission.FilesTotal == 0 || transmission.FilesTransmitted < transmission.FilesTotal {
				slog.Debug("Transmission not fully transmitted yet, skipping pruning", "action", "prune", "task", "PruneExpiredTransmissions", "title", transmission.Title, "resolution", transmission.Resolution, "preview", transmission.Preview, "streamId", transmission.StreamId, "filesTransmitted", transmission.FilesTransmitted, "filesTotal", transmission.FilesTotal)
				continue
			}
			if time.Since(transmission.TransmissionTime) > cfg.CacheExpiry {
				slog.Info("Pruning expired transmission", "action", "prune", "task", "PruneExpiredTransmissions", "title", transmission.Title, "resolution", transmission.Resolution, "streamId", transmission.StreamId)
				for i, t := range *transmissions {
					if t.Title == transmission.Title && t.Resolution == transmission.Resolution {
						(*transmissions)[i].State = "expired"
						break
					}
				}
			}
		}
		for i := len(*transmissions) - 1; i >= 0; i-- {
			if (*transmissions)[i].State == "expired" {
				*transmissions = append((*transmissions)[:i], (*transmissions)[i+1:]...)
			}
		}

		if cfg.Breakpoint {
			return nil
		}
		slog.Debug("Sleeping", "task", "PruneExpiredTransmissions", "duration", cfg.Sleep)
		time.Sleep(cfg.Sleep)
	}

}
