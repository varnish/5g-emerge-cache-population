package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jroimartin/gocui"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/config"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/titles"
	"github.com/varnish/5ge-arcticspace/cache-population/pkg/types"
)

var transmissions []types.TitleTransmission
var titlesWithDuration map[string]types.TitleInfo

func fetchTransmissions() ([]types.TitleTransmission, error) {
	resp, err := http.Get("http://localhost:8888/transmissions")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result []types.TitleTransmission
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchTitlesWithDuration(cfg config.Config) (map[string]types.TitleInfo, error) {
	return titles.GetTitlesWithWatchDuration(cfg)
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	// Top: transmissions
	if v, err := g.SetView("main", 0, 0, maxX-1, maxY/2-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Transmissions"
		v.Wrap = true
	}
	// Bottom: titles with watch duration
	if v, err := g.SetView("titles", 0, maxY/2, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Titles & Watch Duration"
		v.Wrap = true
	}
	return nil
}

func formatRemainingTime(expiry time.Time) string {
	remaining := time.Until(expiry)
	if remaining < 0 {
		return "expired"
	}
	seconds := int(remaining.Seconds())
	if seconds >= 86400 {
		days := seconds / 86400
		return fmt.Sprintf("%dd", days)
	} else if seconds >= 3600 {
		h := seconds / 3600
		m := (seconds % 3600) / 60
		s := seconds % 60
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if seconds >= 60 {
		m := seconds / 60
		s := seconds % 60
		return fmt.Sprintf("%dm %ds", m, s)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

func updateView(g *gocui.Gui, cfg *config.Config) {
	g.Update(func(gui *gocui.Gui) error {
		// Left panel
		v, err := gui.View("main")
		if err != nil {
			return err
		}
		v.Clear()
		fmt.Fprintf(v, "%-15s %-10s %-35s %-10s %-8s %-10s %-10s\n", "Title", "Resolution", "StreamId", "Transmitted", "Total", "Preview", "Expiration")
		for _, t := range transmissions {
			expiry := t.TransmissionTime.Add(cfg.CacheExpiry)
			fmt.Fprintf(v, "%-15s %-10d %-35s %-10d %-8d %-10t %-10s\n", t.Title, t.Resolution, t.StreamId, t.FilesTransmitted, t.FilesTotal, t.Preview, formatRemainingTime(expiry))
		}
		// Right panel
		tv, err := gui.View("titles")
		if err != nil {
			return err
		}
		tv.Clear()
		fmt.Fprintf(tv, "%-30s %-15s\n", "Title", "Watch Duration")
		for _, info := range titlesWithDuration {
			fmt.Fprintf(tv, "%-30s %-15s\n", info.Title, info.WatchDuration.String())
		}
		return nil
	})
}

func suppressLogs() {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})
	slog.SetDefault(slog.New(h))
}

func main() {
	suppressLogs()
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize gocui: %v\n", err)
		os.Exit(1)
	}
	defer g.Close()
	g.SetManagerFunc(layout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(*gocui.Gui, *gocui.View) error {
		cancel()
		g.Close()
		os.Exit(0)
		return gocui.ErrQuit
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Keybinding error: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	cfg := config.ConfigFromEnv()
	cfg.AvailableTitles, err = titles.LoadTitles(cfg.TitlesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load titles: %v\n", err)
		os.Exit(1)
	}

	go func() {
		<-sigCh
		cancel()
		g.Close()
		os.Exit(0)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				newTransmissions, err := fetchTransmissions()
				if err == nil {
					transmissions = newTransmissions
				}
				titlesMap, err := fetchTitlesWithDuration(*cfg)
				if err == nil {
					titlesWithDuration = titlesMap
				}
				updateView(g, cfg)
				time.Sleep(1 * time.Second)
			}
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
