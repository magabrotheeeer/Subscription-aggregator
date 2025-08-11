package month

import "time"

func MonthsBetween(from, to time.Time) int {
	y1, m1 := from.Year(), from.Month()
	y2, m2 := to.Year(), to.Month()
	return (y2-y1)*12 + int(m2-m1) + 1
}

func MaxDate(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func MinDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
