// Package month предоставляет функции для вычисления количества месяцев подписки,
// попадающих в заданный период фильтра.
//
// CountMonthsSimple рассчитывает количество месяцев подписки, которые остаются активны
// на момент начала фильтра (filterStart), учитывая дату начала подписки (subStart) и срок подписки в месяцах.
package month

import (
	"time"
)

// CountMonths вычисляет количество месяцев подписки, остающихся активными в момент начала фильтра.
//
// subStart — дата начала подписки.
// subMonths — длительность подписки в месяцах.
// filterStart — дата начала периода фильтрации.
//
// Функция возвращает количество месяцев подписки, которые пересекаются с фильтром.
func CountMonths(subStart time.Time, subMonths int, filterStart time.Time) int {
	subEnd := subStart.AddDate(0, subMonths, 0)

	// Если фильтр начинается после окончания подписки, то подписка не пересекается с фильтром.
	if !filterStart.Before(subEnd) {
		return 0
	}

	// Если фильтр начинается до или в день начала подписки, то считаем полное количество месяцев подписки.
	if !filterStart.After(subStart) {
		return subMonths
	}

	// Вычисляем разницу в месяцах между началом подписки и началом фильтра.
	monthsDiff := (filterStart.Year()-subStart.Year())*12 +
		int(filterStart.Month()) - int(subStart.Month())

	// Если день начала фильтра позже дня начала подписки, добавляем один месяц.
	if filterStart.Day() > subStart.Day() {
		monthsDiff++
	}

	remaining := subMonths - monthsDiff
	if remaining < 0 {
		return 0
	}

	return remaining
}
