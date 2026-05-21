package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/logs"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/titles"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/transmission"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

func main() {
	Run()
}

func ApiServer(cfg *config.Config, transmissions *[]types.TitleTransmission) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Cache Population API"))
	})

	mux.HandleFunc("/transmissions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(transmissions)
	})

	mux.HandleFunc("/titles", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		t, _ := titles.GetTitlesWithWatchDuration(*cfg)
		json.NewEncoder(w).Encode(t)
	})

	// Start HTTP server
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.ApiServerPort), mux); err != nil {
		slog.Error("API server error", "error", err)
		os.Exit(1)
	}
}

func Run() {
	cfg := config.ConfigFromEnv()
	logs.SetupLogger(cfg.LogLevel)
	var err error

	errs := make(chan error)
	var transmissions = make([]types.TitleTransmission, 0)
	cfg.AvailableTitles, err = titles.LoadTitles(cfg.TitlesFile)
	if err != nil {
		slog.Error("Error loading titles", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start API server
	go func() {
		slog.Info("API server started", "addr", ":"+strconv.Itoa(cfg.ApiServerPort))
		ApiServer(cfg, &transmissions)
	}()

	go func() {
		slog.Info("Pruning expired transmissions task started", "task", "PruneExpiredTransmissions")
		errs <- transmission.PruneExpiredTransmissions(ctx, *cfg, &transmissions)
	}()
	go func() {
		slog.Info("Updating transmission states task started", "task", "UpdateTransmissionState")
		errs <- transmission.UpdateTransmissionState(ctx, *cfg, &transmissions)
	}()
	go func() {
		slog.Info("Transmit popular titles task started", "task", "TransmitPopularTitles")
		errs <- transmission.TransmitPopularTitles(ctx, *cfg, &transmissions)
	}()
	go func() {
		slog.Info("Cancel unpopular transmissions task started", "task", "CancelUnpopularTransmissions")
		errs <- transmission.CancelUnpopularTransmissions(ctx, *cfg, &transmissions)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case sig := <-sigCh:
			slog.Info("received signal", "signal", sig)
			slog.Info("shutting down")
			cancel()
			return
		case err := <-errs:
			if err != nil {
				slog.Error("Error occurred", "error", err)
				os.Exit(1)
			}
			return
		case <-ctx.Done():
			cancel()
			return
		}
	}
}
