package month

import (
	"testing"
	"time"
)

func TestCountMonths_TableTests(t *testing.T) {
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		subStart    time.Time
		subMonths   int
		filterStart time.Time
		want        int
	}{
		{
			name:        "filter before subscription start",
			subStart:    time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			subMonths:   12,
			filterStart: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want:        12,
		},
		{
			name:        "filter exactly at subscription start",
			subStart:    baseDate,
			subMonths:   6,
			filterStart: baseDate,
			want:        6,
		},

		{
			name:        "filter after subscription end",
			subStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			subMonths:   3,
			filterStart: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			want:        0,
		},
		{
			name:        "filter at subscription end",
			subStart:    baseDate,
			subMonths:   3,
			filterStart: baseDate.AddDate(0, 3, 0),
			want:        0,
		},

		{
			name:        "filter in middle of subscription",
			subStart:    time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			subMonths:   12,
			filterStart: time.Date(2024, 4, 10, 0, 0, 0, 0, time.UTC),
			want:        9,
		},
		{
			name:        "filter day after subscription day",
			subStart:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			subMonths:   6,
			filterStart: time.Date(2024, 3, 16, 0, 0, 0, 0, time.UTC),
			want:        3,
		},
		{
			name:        "filter day before subscription day",
			subStart:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			subMonths:   6,
			filterStart: time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC),
			want:        4,
		},

		{
			name:        "single month subscription with filter at start",
			subStart:    baseDate,
			subMonths:   1,
			filterStart: baseDate,
			want:        1,
		},
		{
			name:        "zero months subscription",
			subStart:    baseDate,
			subMonths:   0,
			filterStart: baseDate,
			want:        0,
		},

		{
			name:        "year transition",
			subStart:    time.Date(2024, 11, 20, 0, 0, 0, 0, time.UTC),
			subMonths:   6,
			filterStart: time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC),
			want:        3,
		},

		{
			name:        "remaining months negative",
			subStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			subMonths:   2,
			filterStart: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			want:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountMonths(tt.subStart, tt.subMonths, tt.filterStart)
			if got != tt.want {
				t.Errorf("CountMonths(%v, %d, %v) = %d, want %d",
					tt.subStart, tt.subMonths, tt.filterStart, got, tt.want)
			}
		})
	}
}

func TestCountMonths_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		subStart    time.Time
		subMonths   int
		filterStart time.Time
		want        int
	}{
		{
			name:        "filter at last day of subscription",
			subStart:    time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			subMonths:   1,
			filterStart: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			want:        0,
		},
		{
			name:        "filter one day before subscription end",
			subStart:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			subMonths:   3,
			filterStart: time.Date(2024, 4, 14, 0, 0, 0, 0, time.UTC),
			want:        0,
		},
		{
			name:        "last day of month filter",
			subStart:    time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			subMonths:   3,
			filterStart: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			want:        2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountMonths(tt.subStart, tt.subMonths, tt.filterStart)
			if got != tt.want {
				t.Errorf("CountMonths(%v, %d, %v) = %d, want %d",
					tt.subStart, tt.subMonths, tt.filterStart, got, tt.want)
			}
		})
	}
}
