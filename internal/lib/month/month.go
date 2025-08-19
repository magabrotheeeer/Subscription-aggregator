// Package month содержит вспомогательные функции для работы с датами и временем,
// включая вычисление количества месяцев между датами и нахождение максимальных/минимальных дат.
package month

import "time"

// FullMonthsInOverlap считает количество целых месяцев пересечения двух периодов (подписка и фильтр).
func FullMonthsInOverlap(startSub, endSub, filterStart, filterEnd time.Time) int {
	start := maxDate(startSub, filterStart)
	end := minDate(endSub, filterEnd)

	if end.Before(start) || end.Equal(start) {
		return 0
	}

	yearStart, monthStart, _ := start.Date()
	yearEnd, monthEnd, _ := end.Date()

	months := (yearEnd-yearStart)*12 + int(monthEnd) - int(monthStart)

	// Проверяем, если день конца меньше дня начала — уменьшаем месяц на один
	if end.Day() < start.Day() {
		months--
	}
	if months < 0 {
		return 0
	}
	return months + 1 // +1 чтобы считать текущий месяц за полный
}

func maxDate(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
