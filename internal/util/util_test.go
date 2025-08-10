package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonthsBetween(t *testing.T) {
	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want int
	}{
		{
			name: "same month",
			from: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want: 1,
		},
		{
			name: "three months",
			from: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			want: 3,
		},
		{
			name: "december to january",
			from: time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			want: 2, // так считает твоя формула
		},
		{
			name: "full year",
			from: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			want: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MonthsBetween(tt.from, tt.to)
			assert.Equal(t, tt.want, got, "months mismatch")
		})
	}
}

func TestMaxDate(t *testing.T) {
	d1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)

	assert.True(t, MaxDate(d1, d2).Equal(d1))
	assert.True(t, MaxDate(d2, d1).Equal(d1))
	assert.True(t, MaxDate(d1, d1).Equal(d1))
}

func TestMinDate(t *testing.T) {
	d1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)

	assert.True(t, MinDate(d1, d2).Equal(d2))
	assert.True(t, MinDate(d2, d1).Equal(d2))
	assert.True(t, MinDate(d2, d2).Equal(d2))
}
