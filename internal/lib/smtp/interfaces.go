// Package smtp предоставляет интерфейсы для работы с SMTP.
package smtp

import "io"

// Client интерфейс для SMTP клиента.
type Client interface {
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

// TransportInterface интерфейс для SMTP транспорта.
type TransportInterface interface {
	Connect() (Client, error)
	GetSMTPUser() string
}
