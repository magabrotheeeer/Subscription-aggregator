// Package month содержит вспомогательные функции для работы с датами и временем,
// включая вычисление количества месяцев между датами и нахождение максимальных/минимальных дат.
package month

import "time"

// FullMonthsInOverlap считает количество целых месяцев пересечения двух периодов (подписка и фильтр).
func CountMonthsSimple(subStart time.Time, subMonths int, filterStart time.Time) int {
	if filterStart.Before(subStart) || filterStart.Equal(subStart) {
		return subMonths
	}

	yearDiff := filterStart.Year() - subStart.Year()
	monthDiff := int(filterStart.Month()) - int(subStart.Month())
	monthsDiff := yearDiff*12 + monthDiff

	// Если фильтр в том же месяце, но день фильтра позже дня начала подписки —
	// уменьшаем количество месяцев на 1
	if yearDiff == 0 && monthDiff == 0 && filterStart.Day() > subStart.Day() {
		monthsDiff = 1
	}

	remainingMonths := subMonths - monthsDiff
	if remainingMonths < 0 {
		remainingMonths = 0
	}
	return remainingMonths
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
