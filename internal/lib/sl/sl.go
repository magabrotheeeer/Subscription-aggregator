// Package sl содержит вспомогательные функции для работы с логгером slog.
// Основная цель — упростить формирование структурированных полей лога,
// например, для передачи информации об ошибках.
package sl

import "log/slog"

// Err возвращает slog.Attr с ключом "error" и значением текста ошибки.
// Удобно использовать в логировании для единообразного вывода ошибок.
//
// Пример:
//
//	log.Error("failed to do something", sl.Err(err))
func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}
