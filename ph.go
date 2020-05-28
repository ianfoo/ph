package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const urlJEMP = "https://public.radio.co/stations/sd71de59b3/status"

var (
	zeroes   = regexp.MustCompile(`^0[hms]|(\D)0[hms]`)
	jempDate = regexp.MustCompile(`\d{1,2}(?P<separator>[-.\/])\d{1,2}[-.\/]\d{2}`)
)

func main() {
	if err := run(); err != nil {
		log.SetPrefix("error: ")
		log.SetFlags(0)
		log.Fatal(err)
	}
}

func run() error {
	var lastN uint
	flag.UintVar(&lastN, "last", 0, "Show this many latest songs (0 means current song)")
	flag.Parse()

	resp, err := http.Get(urlJEMP)
	if err != nil {
		return fmt.Errorf("get JEMP Radio status: %w", err)
	}
	defer resp.Body.Close()
	var status statusResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("parsing status response: %w", err)
	}

	if lastN == 0 {
		t := NewTrackFromResponseTrack(status.CurrentTrack)
		fmt.Println(t)
		return nil
	}
	for i := 0; i < int(lastN); i++ {
		if i >= len(status.History) {
			break
		}
		t := NewTrackFromResponseTrack(status.History[i])
		fmt.Println(t)
	}
	return nil
}

type (
	responseTrack struct {
		Title     string `json:"title"`
		StartTime string `json:"start_time"`
	}
	statusResponseBody struct {
		CurrentTrack responseTrack   `json:"current_track"`
		History      []responseTrack `json:"history"`
	}

	// Track represents a track being played on radio.co.
	Track struct {
		Title     string    `json:"title"`
		StartTime time.Time `json:"start_time,omitempty"`
	}
)

// Elapsed returns a duration indicating how long ago playback of the track
// started if the track has a start time. If it does not, then a zero duration
// is returned.
func (t Track) Elapsed() time.Duration {
	if st := t.StartTime; !st.IsZero() {
		return time.Since(st).Round(time.Second)
	}
	return 0
}

// PhishinURL returns a link to the Phish.in page for the currently-playing
// show, if the track is determined to be a Phish track with an identifiable
// show date in JEMP Radio's mm-dd-yy format.
func (t Track) PhishinURL() string {
	if !strings.HasPrefix(t.Title, "Phish ") {
		return ""
	}
	matches := jempDate.FindStringSubmatch(t.Title)
	if len(matches) == 0 {
		return ""
	}
	var (
		dateStr   = matches[0]
		separator = matches[1]
	)
	date, err := time.Parse(fmt.Sprintf("1%s2%s06", separator, separator), dateStr)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return fmt.Sprintf("https://phish.in/%4d-%02d-%02d", date.Year(), date.Month(), date.Day())
}

// String returns a string representation of a track, including the title,
// and--if a start time is defined--how long ago the track started playing.
func (t Track) String() string {
	var (
		str     = t.Title
		elapsed = t.Elapsed()
	)
	if elapsed == 0 {
		return str
	}
	str += fmt.Sprintf(" (started %s)", StartedString(elapsed))
	if phishinURL := t.PhishinURL(); phishinURL != "" {
		str += "\n" + phishinURL
	}
	return str
}

// NewTrackFromResponseTrack generates a new Track struct from the track
// representation that is returned in the radio.co response body.
func NewTrackFromResponseTrack(rt responseTrack) Track {
	track := Track{Title: rt.Title}
	if rt.StartTime == "" {
		return track
	}
	startTime, err := time.Parse(time.RFC3339, rt.StartTime)
	if err != nil {
		return track
	}
	track.StartTime = startTime
	return track
}

// StartedString converts a duration into a human-friendly string represntation
// of how long ago the duration was.
func StartedString(d time.Duration) string {
	dstr := zeroes.ReplaceAllString(d.Truncate(time.Second).String(), "$1")
	if dstr != "" {
		return dstr + " ago"
	}
	return "just now"
}
