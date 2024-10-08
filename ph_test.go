package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestTrack_UnmarshalJSON(t *testing.T) {
	tt := []struct {
		desc    string
		payload string
		want    Track
		wantErr error
	}{
		{
			desc:    "title and start time",
			payload: `{"title": "Phish - Chalk Dust Torture (7-18-14)", "start_time": "2020-05-28T08:01:32+00:00"}`,
			want: Track{
				Artist:          "Phish",
				Title:           "Chalk Dust Torture",
				StartTime:       mustParseDate("2020-05-28T08:01:32"),
				PerformanceTime: mustParseDate("2014-07-18"),
			},
		},
		{
			desc:    "no start time",
			payload: `{"title": "Phish - Chalk Dust Torture (7-18-14)"}`,
			want: Track{
				Artist:          "Phish",
				Title:           "Chalk Dust Torture",
				PerformanceTime: mustParseDate("2014-07-18"),
			},
		},
		{
			desc:    "invalid start time",
			payload: `{"title": "Phish - Chalk Dust Torture (7-18-14)", "start_time": "invalid date"}`,
			want: Track{
				Artist:          "Phish",
				Title:           "Chalk Dust Torture",
				PerformanceTime: mustParseDate("2014-07-18"),
			},
			wantErr: &time.ParseError{},
		},
		{
			desc:    "has performance date (dashes)",
			payload: `{"title": "Phish - Lushington (5-20-87)"}`,
			want: Track{
				Artist:          "Phish",
				Title:           "Lushington",
				PerformanceTime: mustParseDate("1987-05-20"),
			},
		},
		{
			desc:    "has performance date (slashes)",
			payload: `{"title": "Phish - Lushington (5/20/87)"}`,
			want: Track{
				Artist:          "Phish",
				Title:           "Lushington",
				PerformanceTime: mustParseDate("1987-05-20"),
			},
		},
		{
			desc:    "has performance date (dots)",
			payload: `{"title": "Phish - Lushington (5.20.87)"}`,
			want: Track{
				Artist:          "Phish",
				Title:           "Lushington",
				PerformanceTime: mustParseDate("1987-05-20"),
			},
		},
		{
			desc:    "has date, but not performance date",
			payload: `{"title": "Alex Grosby - The Phishsonian Hour 5-28-20"}`,
			want: Track{
				Artist: "Alex Grosby",
				Title:  "The Phishsonian Hour 5-28-20",
			},
		},
		{
			desc:    "no identifiable artist name field",
			payload: `{"title": "No Separator Band Foo Foo (1-1-20)"}`,
			want: Track{
				Title:           "No Separator Band Foo Foo",
				PerformanceTime: mustParseDate("2020-01-01"),
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			var got Track
			if err := json.Unmarshal([]byte(tc.payload), &got); err != nil {
				if tc.wantErr == nil {
					t.Fatalf("unexpected error unmarshaling JSON (test data error?): %v", err)
					return
				}
				// Just compare error types here, since the only test case that should
				// have an error is the invalid start date case, so we know it'll be a
				// time.ParseError.
				if want, got := reflect.TypeOf(tc.wantErr), reflect.TypeOf(err); want != got {
					t.Fatalf("expected error of type %v, but got error of type %v: %v", want, got, err)
					return
				}
			}
			if !cmp.Equal(tc.want, got) {
				t.Errorf("got unexpected result (-want +got):\n%s", cmp.Diff(tc.want, got))
			}
		})
	}
}

func TestTrack_Elapsed(t *testing.T) {
	dur := time.Duration(30 * time.Second)
	tt := []struct {
		start time.Time
		want  time.Duration
	}{
		{start: time.Now().Add(-dur), want: dur},
		{want: 0},
	}
	for _, tc := range tt {
		t.Run(tc.start.String(), func(t *testing.T) {
			var (
				track = Track{StartTime: tc.start}
				got   = track.Elapsed()
			)
			if got != tc.want {
				t.Fatalf("wanted duration %v, but got %v", tc.want, got)
			}
		})
	}
}

func TestTrack_StreamingURL(t *testing.T) {
	tt := []struct {
		desc  string
		track Track
		want  string
	}{
		{
			desc: "no date",
			track: Track{
				Artist: "Phish",
				Title:  "Phish - Sigma Oasis",
			},
			want: "",
		},
		{
			desc: "no artist",
			track: Track{
				Title:           "Phish - Sigma Oasis",
				PerformanceTime: mustParseDate("2020-01-01"),
			},
			want: "",
		},
		{
			desc: "Phish",
			track: Track{
				Artist:          "Phish",
				Title:           "Phish - Mercury (7-14-19)",
				PerformanceTime: mustParseDate("2019-07-14"),
			},
			want: "https://relisten.net/phish/2019/07/14",
		},
		{
			desc: "Grateful Dead",
			track: Track{
				Artist:          "Grateful Dead",
				Title:           "Grateful Dead - Deal (1985-03-26)",
				PerformanceTime: mustParseDate("1985-03-26"),
			},
			want: "https://relisten.net/grateful-dead/1985/03/26",
		},
	}
	// TODO Use a locally-persisted "golden" copy of the artists map.
	// TODO Make an artists map from a byte slice, to decouple it from the HTTP client.
	relistenArtists, err := relistenGetArtists(http.DefaultClient)
	if err != nil {
		t.Fatalf("unable to get relisten artists: %v", err)
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			if got := tc.track.StreamingURL(relistenArtists); tc.want != got {
				t.Errorf("wanted %q, but got %q", tc.want, got)
			}
		})
	}
}

func TestTrack_String(t *testing.T) {
	dur := time.Duration(90 * time.Second)
	tt := []struct {
		desc  string
		track Track
		want  string
	}{
		{
			desc: "with start time and performance time",
			track: Track{
				Artist:          "Phish",
				Title:           "Mercury",
				StartTime:       time.Now().Add(-dur),
				PerformanceTime: mustParseDate("2019-07-14"),
			},
			want: "Phish - Mercury (Sun 14-Jul-2019) (started 1m30s ago)\n" +
				"https://relisten.net/phish/2019/07/14\n" +
				"https://phish.net/setlists/?d=2019-07-14",
		},
		{
			desc: "no start time",
			track: Track{
				Artist:          "Phish",
				Title:           "Mercury",
				PerformanceTime: mustParseDate("2019-07-14"),
			},
			want: "Phish - Mercury (Sun 14-Jul-2019)\n" +
				"https://relisten.net/phish/2019/07/14\n" +
				"https://phish.net/setlists/?d=2019-07-14",
		},
		{
			desc: "no performance time",
			track: Track{
				Artist: "Phish",
				Title:  "Mercury",
			},
			want: "Phish - Mercury",
		},
		{
			desc:  "no artist name",
			track: Track{Title: "Dogs Stole Things"},
			want:  "Dogs Stole Things",
		},
	}

	// TODO Get rid of the package-level variable for relistenArtists.
	// Allow tracks to be stringified without it.
	var err error
	relistenArtists, err = relistenGetArtists(http.DefaultClient)
	if err != nil {
		t.Fatalf("unable to get relisten artists: %v", err)
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			if got := tc.track.String(); got != tc.want {
				t.Errorf("wanted %q, but got %q", tc.want, got)
			}
		})
	}
}

func TestStartedString(t *testing.T) {
	tt := []struct {
		in   time.Duration
		want string
	}{
		{time.Second, "1s ago"},
		{time.Minute, "1m ago"},
		{67 * time.Second, "1m7s ago"},
		{90 * time.Second, "1m30s ago"},
		{67 * time.Minute, "1h7m ago"},
		{3607 * time.Second, "1h7s ago"},
		{0, "just now"},
		{1000, "just now"},
	}
	for _, tc := range tt {
		t.Run(tc.in.String(), func(t *testing.T) {
			got := StartedString(tc.in)
			if got != tc.want {
				t.Fatalf("%s: wanted %q, but got %q", tc.in, tc.want, got)
			}
		})
	}
}

func TestTrackList_FilterArtist(t *testing.T) {
	tt := []struct {
		desc       string
		in         TrackList
		filterFunc func(string) bool
		want       TrackList
	}{
		{
			desc: "Phish only",
			in: TrackList{
				Track{Artist: "Phish"},
				Track{Artist: "Grateful Dead"},
				Track{Artist: "Phish"},
			},
			filterFunc: func(s string) bool {
				return s == "Phish"
			},
			want: TrackList{
				Track{Artist: "Phish"},
				Track{Artist: "Phish"},
			},
		},
		{
			desc: "Exclude JEMP Radio meta-tracks",
			in: TrackList{
				Track{Artist: "Phish"},
				Track{Artist: "www.jempradio.com"},
				Track{Artist: "jempradio.com"},
				Track{Artist: "Phish"},
			},
			filterFunc: func(s string) bool {
				return s != "jempradio.com" && s != "www.jempradio.com"
			},
			want: TrackList{
				Track{Artist: "Phish"},
				Track{Artist: "Phish"},
			},
		},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.in.FilterArtist(tc.filterFunc)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("wanted %v but got %v", tc.want, got)
			}
		})
	}
}

func mustParseDate(dateStr string) time.Time {
	if !strings.Contains(dateStr, "T") {
		dateStr += "T00:00:00"
	}
	if !strings.Contains(dateStr, "+") {
		dateStr += "+00:00"
	}
	d, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		panic(fmt.Sprintf("unable to parse test date %q: %v", dateStr, err))
	}
	return d
}
