// Package router содержит интерфейсы для роутера команд Telegram-бота.
package router

import "gemfactory/internal/telegrambot/bot/types"

// Interface определяет интерфейс для роутера команд Telegram-бота.
type Interface interface {
	// Use добавляет middleware к роутеру
	Use(middleware types.Middleware)

	// Handle регистрирует обработчик команды
	Handle(command string, handler types.HandlerFunc)

	// Dispatch отправляет команду к обработчику
	Dispatch(ctx types.Context) error
}
