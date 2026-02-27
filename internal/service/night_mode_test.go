package service

import (
	"testing"
	"time"
)

func TestParseNightHour(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "zero", input: "0", want: 0},
		{name: "max", input: "23", want: 23},
		{name: "trim spaces", input: " 8 ", want: 8},
		{name: "empty", input: "", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "overflow", input: "24", wantErr: true},
		{name: "invalid", input: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNightHour(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %d, got %d", tt.want, got)
			}
		})
	}
}

func TestIsNightWindowNow(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		startHour int
		endHour   int
		want      bool
	}{
		{
			name:      "normal range in window",
			now:       time.Date(2026, time.January, 1, 1, 30, 0, 0, time.UTC),
			startHour: 0,
			endHour:   8,
			want:      true,
		},
		{
			name:      "normal range out of window",
			now:       time.Date(2026, time.January, 1, 9, 0, 0, 0, time.UTC),
			startHour: 0,
			endHour:   8,
			want:      false,
		},
		{
			name:      "cross day late night",
			now:       time.Date(2026, time.January, 1, 23, 0, 0, 0, time.UTC),
			startHour: 22,
			endHour:   6,
			want:      true,
		},
		{
			name:      "cross day early morning",
			now:       time.Date(2026, time.January, 1, 5, 59, 0, 0, time.UTC),
			startHour: 22,
			endHour:   6,
			want:      true,
		},
		{
			name:      "cross day daytime",
			now:       time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC),
			startHour: 22,
			endHour:   6,
			want:      false,
		},
		{
			name:      "same hour means full day",
			now:       time.Date(2026, time.January, 1, 15, 0, 0, 0, time.UTC),
			startHour: 10,
			endHour:   10,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isNightWindowNow(0, tt.startHour, tt.endHour, tt.now)
			if got != tt.want {
				t.Fatalf("want %v, got %v", tt.want, got)
			}
		})
	}
}
