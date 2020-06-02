package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

const (
	urlJEMP = "https://public.radio.co/stations/sd71de59b3/status"

	patJEMPDate         = `(?P<date>\d{1,2}(?P<separator>[-./])\d{1,2}[-./]\d{2})`
	patJEMPRegularTrack = `^(?P<artist>.+)\s+-\s+(?P<title>.+?)(?:\s+\(` + patJEMPDate + `(?:\s+(?P<location>.+))?\))?$`
	patJEMPFullShow     = `^(?P<artist>.+)\s+-\s+` + patJEMPDate +
		`\s+(?P<set>(?:Set \d+(?:\s?\+\s?E)?)|Encore)\s+\((?P<location>.+)\)$`
	patJEMPStationArtist = `^(?:www\.)?jempradio\.com`
)

// zeros regexp detects cases zero-value units in duration strings, so
// that, for example, the duration "1h0m30s," as would be rendered by
// default, can be presented more compactly as "1h30s."
var zeroes = regexp.MustCompile(`(?:^|(\D))0[hms]`)

var (
	jempDate         = regexp.MustCompile(`\((` + patJEMPDate + `)\)$`)
	jempStationBreak = regexp.MustCompile(patJEMPStationArtist)

	// Order is important! Consider "studio track" a fallthrough that will
	// match anything not matched by the previous expressions.
	regexJEMPTrack = []*regexp.Regexp{
		regexp.MustCompile(patJEMPFullShow),
		regexp.MustCompile(patJEMPRegularTrack),
	}
)

func main() {
	if err := run(); err != nil {
		log.SetPrefix("error: ")
		log.SetFlags(0)
		log.Fatal(err)
	}
}

func run() error {
	var (
		lastN   uint
		history bool
		format  string
	)
	flag.UintVarP(&lastN, "last", "l", 1, "Show this many latest songs")
	flag.BoolVar(&history, "history", false, "Show entire available history")
	flag.StringVarP(&format, "format", "f", "text", "output format (text, json, yaml)")
	flag.Parse()

	writeOutput, err := getRenderer(format)
	if err != nil {
		return err
	}
	resp, err := http.Get(urlJEMP)
	if err != nil {
		return fmt.Errorf("get JEMP Radio status: %w", err)
	}
	defer resp.Body.Close()
	var status statusResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("parsing status response: %w", err)
	}

	// NOTE Current track might be a JEMP station break.
	if lastN == 1 {
		writeOutput(status.CurrentTrack)
		return nil
	}

	noJEMPStationBreaks := func(artist string) bool {
		return !jempStationBreak.MatchString(artist)
	}
	if history {
		lastN = 0
	}
	lastNTracks := status.History.FilterArtist(noJEMPStationBreaks).LastN(lastN)
	writeOutput(lastNTracks)
	return nil
}

type statusResponseBody struct {
	CurrentTrack Track     `json:"current_track"`
	History      TrackList `json:"history"`
}

type TrackList []Track

// LastN returns the last n tracks from the TrackList. If n is zero, then the
// entire TrackList is returned.
func (tl TrackList) LastN(n uint) TrackList {
	if n == 0 {
		return tl
	}
	if l := uint(len(tl)); n > l {
		n = l
	}
	return tl[:n]
}

// FilterArtist will return a TrackList of those tracks for which filterFunc
// returns true when passed the artist name.
func (tl TrackList) FilterArtist(filterFunc func(string) bool) TrackList {
	out := make(TrackList, 0, len(tl))
	for _, t := range tl {
		if filterFunc(t.Artist) {
			out = append(out, t)
		}
	}
	return out
}

// String renders the tracklist as a text table.
func (tl TrackList) String() string {
	if len(tl) == 0 {
		return ""
	}
	const (
		headingArtist         = "ARTIST"
		headingTitle          = "TITLE"
		headingPeformanceTime = "PERFORMED ON"
		headlingStreamingURL  = "STREAM"
	)
	const (
		dateFormat = "Mon _2-Jan-2006"
		maxLenDate = len(dateFormat) + 1
	)
	var (
		maxLenArtist = len(headingArtist)
		maxLenTitle  = len(headingTitle)
	)
	for _, t := range tl {
		if l := len(t.Artist); l > maxLenArtist {
			maxLenArtist = l
		}
		if l := len(t.Title); l > maxLenTitle {
			maxLenTitle = l
		}
	}
	var (
		numTracks     = float64(len(tl))
		maxLenIndex   = int(math.Floor(math.Log10(numTracks))) + 1
		baseFormat    = fmt.Sprintf("%%-%ds  %%-%ds  %%-%ds  %%s\n", maxLenArtist, maxLenTitle, maxLenDate)
		headingFormat = strings.Repeat(" ", maxLenIndex+1) + baseFormat
		itemFormat    = fmt.Sprintf("%%%dd %s", maxLenIndex, baseFormat)

		builder strings.Builder
	)
	builder.WriteString(fmt.Sprintf(
		headingFormat,
		headingArtist,
		headingTitle,
		headingPeformanceTime,
		headlingStreamingURL))
	for i, t := range tl {
		var perfTimeStr string
		if pt := t.PerformanceTime; !pt.IsZero() {
			perfTimeStr = pt.Format(dateFormat)
		}
		builder.WriteString(fmt.Sprintf(itemFormat, i+1, t.Artist, t.Title, perfTimeStr, t.StreamingURL()))
	}
	s := builder.String()
	return s[:len(s)-1]
}

// Track represents a track being played on radio.co.
type Track struct {
	Artist          string    `json:"artist,omitempty"`
	Title           string    `json:"title"`
	StartTime       time.Time `json:"start_time,omitempty" yaml:"start_time,omitempty"`
	PerformanceTime time.Time `json:"performance_time,omitempty" yaml:"performance_time,omitempty"`
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
	t.parseRawTitle(respTrack.Title)

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

func (t *Track) parseRawTitle(title string) {
	var (
		matches       []string
		matchedRegexp *regexp.Regexp
	)
	for _, re := range regexJEMPTrack {
		m := re.FindStringSubmatch(title)
		if len(m) > 1 {
			matches = m
			matchedRegexp = re
			break
		}
	}

	// Didn't match any of our expected formats.
	if matchedRegexp == nil {
		t.Title = title
		return
	}
	var (
		perfTimeStr string
		perfTimeSep string
		location    string
		set         string
	)
	for i, subexp := range matchedRegexp.SubexpNames() {
		switch subexp {
		case "artist":
			t.Artist = strings.TrimSpace(matches[i])
		case "title":
			t.Title = strings.TrimSpace(matches[i])
		case "date":
			perfTimeStr = matches[i]
		case "separator":
			perfTimeSep = matches[i]
		case "location":
			location = strings.TrimSpace(matches[i])
		case "set":
			set = strings.TrimSpace(matches[i])
		}
	}
	if perfTimeStr != "" && perfTimeSep != "" {
		parseFormat := fmt.Sprintf("1%s2%s06", perfTimeSep, perfTimeSep)
		perfTime, err := time.Parse(parseFormat, perfTimeStr)
		if err == nil {
			t.PerformanceTime = perfTime
		}
	}

	// We are finished if this is not a full show title.
	if set == "" || t.PerformanceTime.IsZero() {
		return
	}
	perfTimeStr = t.PerformanceTime.Format("2-Jan-2006")
	if location != "" {
		t.Title = perfTimeStr + " " + location + " " + set
		return
	}
	t.Title = perfTimeStr + " " + set
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
	if t.Artist == "" || t.PerformanceTime.IsZero() {
		return ""
	}
	streamableAs := func() (string, bool) {
		// Bands is a set of bands that are commonly played on JEMP Radio that
		// are available for streaming via Relisten. The map values are the URL
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
		path, ok := bands[t.Artist]
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

// PhishNetURL returns a URL pointing to the setlist on phish.net for the show
// that this track is from, if the track is a live Phish track.
func (t Track) PhishNetURL() string {
	if t.Artist != "Phish" || t.PerformanceTime.IsZero() {
		return ""
	}
	return "https://phish.net/setlists/?d=" + t.PerformanceTime.Format("2006-01-02")
}

// String returns a string representation of a track, including the title,
// and--if a start time is defined--how long ago the track started playing.
func (t Track) String() string {
	str := t.Artist
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
	if stream := t.StreamingURL(); stream != "" {
		str += "\n" + stream
	}
	if pnet := t.PhishNetURL(); pnet != "" {
		str += "\n" + pnet
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

func getRenderer(format string) (func(interface{}) error, error) {
	switch format {
	case "text":
		f := func(v interface{}) error {
			_, err := fmt.Println(v)
			return err
		}
		return f, nil
	case "json":
		f := func(v interface{}) error {
			return json.NewEncoder(os.Stdout).Encode(v)
		}
		return f, nil
	case "yaml":
		f := func(v interface{}) error {
			return yaml.NewEncoder(os.Stdout).Encode(v)
		}
		return f, nil
	default:
		return nil, fmt.Errorf("invalid output format %q", format)
	}
}
