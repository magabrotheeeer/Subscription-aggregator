// Package month содержит вспомогательные функции для работы с датами и временем,
// включая вычисление количества месяцев между датами и нахождение максимальных/минимальных дат.
package month

import "time"

// MonthsBetween возвращает количество месяцев между двумя датами (включительно).
// Если from и to находятся в одном месяце, результат будет 1.
// Формула учитывает как разницу лет, так и разницу месяцев.
func MonthsBetween(from, to time.Time) int {
	y1, m1 := from.Year(), from.Month()
	y2, m2 := to.Year(), to.Month()
	return (y2-y1)*12 + int(m2-m1) + 1
}

// MaxDate возвращает более позднюю из двух дат.
// Если даты равны, возвращается любая из них.
func MaxDate(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// MinDate возвращает более раннюю из двух дат.
// Если даты равны, возвращается любая из них.
func MinDate(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
