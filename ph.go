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
	zeroes   = regexp.MustCompile(`(?:^|(\D))0[hms]`)
	jempDate = regexp.MustCompile(`\((\d{1,2}(?P<separator>[-./])\d{1,2}[-./]\d{2})\)$`)
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
	flag.UintVar(&lastN, "last", 1, "Show this many latest songs (0 shows entire available history)")
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

	if lastN == 1 {
		fmt.Println(status.CurrentTrack)
		if streamURL := status.CurrentTrack.StreamingURL(); streamURL != "" {
			fmt.Println(streamURL)
		}
		return nil
	}
	lastNTracks := status.LastN(lastN)
	for i, t := range lastNTracks {
		fmt.Printf("%2d. %s", i+1, t)
		if streamURL := t.StreamingURL(); streamURL != "" {
			fmt.Print(" - ", streamURL)
		}
		fmt.Println()
	}
	return nil
}

type statusResponseBody struct {
	CurrentTrack Track   `json:"current_track"`
	History      []Track `json:"history"`
}

// LastN returns the last n tracks from the status history.
// If n is zero, then the entire history is returned.
func (srb statusResponseBody) LastN(n uint) []Track {
	if n == 0 {
		return srb.History
	}
	if l := uint(len(srb.History)); n > l {
		n = l
	}
	return srb.History[:n]
}

// Track represents a track being played on radio.co.
type Track struct {
	Band            string    `json:"band,omitempty"`
	Title           string    `json:"title"`
	StartTime       time.Time `json:"start_time,omitempty"`
	PerformanceTime time.Time `json:"performance_time,omitempty"`
}

// UnmarshalJSON implementes json.Unmarshaler in order to handle
// the conversion of JSON data into a Track struct.
func (t *Track) UnmarshalJSON(b []byte) error {
	var respTrack struct {
		Title     string `json:"title"`
		StartTime string `json:"start_time"`
	}
	if err := json.Unmarshal(b, &respTrack); err != nil {
		return err
	}
	t.Title = respTrack.Title
	t.maybeSetPerformanceTime()
	bandSplit := strings.SplitN(t.Title, " - ", 2)
	if len(bandSplit) == 2 {
		t.Band = bandSplit[0]
		t.Title = bandSplit[1]
	}

	if respTrack.StartTime == "" {
		return nil
	}
	startTime, err := time.Parse(time.RFC3339, respTrack.StartTime)
	if err != nil {
		return err
	}
	t.StartTime = startTime
	return nil
}

// maybeSetPerformanceTime will set the PerformanceTime field if a recognizable
// performance date can be be located in the track title, in accordance with
// the convention JEMP Radio uses to render performance dates into track
// titles. If a performance date is found, it is removed from the title since
// it's then stored in the PerformanceTime field.
func (t *Track) maybeSetPerformanceTime() {
	matches := jempDate.FindStringSubmatch(t.Title)
	if len(matches) == 0 {
		return
	}
	const jempDateParseFormatTemplate = "1%s2%s06"
	var (
		dateStr     = matches[1]
		separator   = matches[2]
		parseFormat = fmt.Sprintf(jempDateParseFormatTemplate, separator, separator)
	)
	date, err := time.Parse(parseFormat, dateStr)
	if err != nil {
		// Just don't set the performance time if there's any trouble parsing it.
		return
	}
	t.PerformanceTime = date
	newTitle := jempDate.ReplaceAllString(t.Title, "")
	t.Title = strings.TrimSpace(newTitle)
}

// Elapsed returns a duration indicating how long ago playback of the track
// started if the track has a start time. If it does not, then a zero duration
// is returned.
func (t Track) Elapsed() time.Duration {
	if st := t.StartTime; !st.IsZero() {
		return time.Since(st).Round(time.Second)
	}
	return 0
}

// StreamingURL returns a link to the streaming page for the currently-playing
// show, if the track has a perfomance date set and the band is one of a set of
// selected bands. There is no guarantee that the link will refer to a valid
// show, since it is possible that a given show is not available for streaming.
func (t Track) StreamingURL() string {
	if t.Band == "" || t.PerformanceTime.IsZero() {
		return ""
	}
	streamableAs := func() (string, bool) {
		// Bands is a set of bands that are commonly played on JEMP Radio that
		// are available for streaming via ReListen. The map values are the URL
		// path element that corresponds to the band's name that appears in the
		// track title. Unfortunately, I cannot find an easily-linkable
		// streaming source for Trey Anastasio Band or Jerry Garcia Band, which
		// get a fair amount of play on JEMP Radio.
		bands := map[string]string{
			"Goose":                   "goose",
			"Grateful Dead":           "grateful-dead",
			"Joe Russo's Almost Dead": "jrad",
			"JRAD":                    "jrad",
			"KVHW":                    "kvhw",
			"Phish":                   "phish",
			"Spafford":                "spafford",
			"Steve Kimock":            "steve-kimock",
			"Steve Kimock Band":       "steve-kimock-band",
			"Widespread Panic":        "wsp",
		}
		path, ok := bands[t.Band]
		return path, ok
	}
	bandPathElem, streamable := streamableAs()
	if !streamable {
		return ""
	}
	var (
		d   = t.PerformanceTime
		url = fmt.Sprintf("https://relisten.net/%s/%4d/%02d/%02d", bandPathElem, d.Year(), d.Month(), d.Day())
	)
	return url
}

// String returns a string representation of a track, including the title,
// and--if a start time is defined--how long ago the track started playing.
func (t Track) String() string {
	str := t.Band
	if str != "" {
		str += " - "
	}
	str += t.Title
	if d := t.PerformanceTime; !d.IsZero() {
		str += fmt.Sprintf(" (%s)", d.Format("Mon 2-Jan-2006"))
	}
	if elapsed := t.Elapsed(); elapsed != 0 {
		str += fmt.Sprintf(" (started %s)", StartedString(elapsed))
	}
	return str
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
