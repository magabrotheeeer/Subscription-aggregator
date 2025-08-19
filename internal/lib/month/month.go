// Package month содержит вспомогательные функции для работы с датами и временем,
// включая вычисление количества месяцев между датами и нахождение максимальных/минимальных дат.
package month

import "time"

func FullMonthsInOverlap(startSub, endSub, filterStart, filterEnd time.Time) int {
    // Пересечение подписки и фильтра
    start := maxDate(startSub, filterStart)
    end := minDate(endSub, filterEnd)

    if end.Before(start) {
        return 0
    }

    count := 0
    current := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

    // Движемся вперед по месяцам, пока не превысим end
    for current.Before(end) || current.Equal(end) {
        // Начало следующего месяца
        nextMonth := current.AddDate(0, 1, 0).Add(-time.Nanosecond) // Конец текущего месяца

        // Если конец текущего месяца входит в интервал пересечения, считаем этот месяц полным
        if !nextMonth.After(end) {
            count++
        } else {
            break
        }
        current = current.AddDate(0, 1, 0) // Переходим к следующему месяцу
    }
    return count
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