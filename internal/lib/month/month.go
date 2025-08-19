package month

import (
	"time"
)

// CountMonthsSimple считает количество месяцев подписки, попадающих в период фильтра
func CountMonths(subStart time.Time, subMonths int, filterStart time.Time) int {
	subEnd := subStart.AddDate(0, subMonths, 0)

	// Если фильтр начинается после окончания подписки
	if !filterStart.Before(subEnd) {
		return 0
	}

	// Если фильтр начинается до или в день начала подписки
	if !filterStart.After(subStart) {
		return subMonths
	}

	// Рассчитываем разницу в полных месяцах между началом подписки и началом фильтра
	monthsDiff := (filterStart.Year()-subStart.Year())*12 +
		int(filterStart.Month()) - int(subStart.Month())

	// Если день фильтра позже дня подписки, вычитаем еще один месяц
	if filterStart.Day() > subStart.Day() {
		monthsDiff++
	}

	remaining := subMonths - monthsDiff
	if remaining < 0 {
		return 0
	}

	return remaining
}
