package types

import "time"

type TitleInfo struct {
	Title         string        `json:"title"`
	WatchDuration time.Duration `json:"watchDuration"`
	Url           string        `json:"url"`
	Resolutions   []int         `json:"resolutions"`
	Preview       bool          `json:"preview"`
}

type TitleTransmission struct {
	Title                 string    `json:"title"`
	Resolution            int       `json:"resolution"`
	StreamId              string    `json:"streamId"`
	State                 string    `json:"state"`
	FilesTransmitted      int       `json:"filesTransmitted"`
	FilesTotal            int       `json:"filesTotal"`
	PercentageTransmitted float64   `json:"percentageTransmitted"`
	Preview               bool      `json:"preview"`
	TransmissionTime      time.Time `json:"transmissionTime"`
}
