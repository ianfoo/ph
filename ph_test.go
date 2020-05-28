package main

import (
	"testing"
	"time"
)

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

func TestTrack_PhishinURL(t *testing.T) {
	tt := []struct {
		desc  string
		title string
		want  string
	}{
		{
			desc:  "not Phish",
			title: "Grateful Dead - Deal (3-26-85)",
			want:  "",
		},
		{
			desc:  "no date",
			title: "Phish - Sigma Oasis",
			want:  "",
		},
		{
			desc:  "invalid date",
			title: "Phish - Lushington (5-32-87)",
			want:  "",
		},
		{
			desc:  "has date (dashes)",
			title: "Phish - Lushington (5-20-87)",
			want:  "https://phish.in/1987-05-20",
		},
		{
			desc:  "has date (slashes)",
			title: "Phish - Lushington (5/20/87)",
			want:  "https://phish.in/1987-05-20",
		},
		{
			desc:  "has date (dots)",
			title: "Phish - Lushington (5.20.87)",
			want:  "https://phish.in/1987-05-20",
		},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			track := Track{Title: tc.title}
			if got := track.PhishinURL(); tc.want != got {
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
			desc:  "with start time",
			track: Track{Title: "Phish - Mercury (7-14-19)", StartTime: time.Now().Add(-dur)},
			want:  "Phish - Mercury (7-14-19) (started 1m30s ago)\nhttps://phish.in/2019-07-14",
		},
		{
			desc:  "no start time",
			track: Track{Title: "Phish - Mercury (7-14-19)"},
			want:  "Phish - Mercury (7-14-19)",
		},
	}
	for _, tc := range tt {
		t.Run(tc.desc, func(t *testing.T) {
			if got := tc.track.String(); got != tc.want {
				t.Errorf("wanted %q, but got %q", tc.want, got)
			}
		})
	}
}

func TestNewTrackFromResponseTrack(t *testing.T) {
	tt := []responseTrack{
		{Title: "Phish - Chalk Dust Torture (7-18-14)", StartTime: "2014-07-14T08:01:32+00:00"},
		{Title: "Phish - Chalk Dust Torture (7-18-14)"},
		{Title: "Phish - Chalk Dust Torture (7-18-14)", StartTime: "invalid date"},
	}
	for _, tc := range tt {
		t.Run(tc.Title, func(t *testing.T) {
			got := NewTrackFromResponseTrack(tc)
			if got.Title != tc.Title {
				t.Errorf("wanted title %q, but got %q", tc.Title, got.Title)
			}
			if tc.StartTime == "" && !got.StartTime.IsZero() {
				t.Errorf("expected start time to be time.Time zero value, but got %v", got.StartTime)
				return
			}
			date, err := time.Parse(time.RFC3339, tc.StartTime)
			if err != nil && !got.StartTime.IsZero() {
				t.Errorf(
					"expected start time to be time.Time zero value with invalid source start time, "+
						"but got %v",
					got.StartTime)
				return
			}
			if !got.StartTime.Equal(date) {
				t.Errorf("expected start date %v, but got %v", date, got.StartTime)
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
