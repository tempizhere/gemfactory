package router

import "gemfactory/internal/telegrambot/bot/types"

// RouterInterface определяет интерфейс для роутера команд
type RouterInterface interface {
	// Use добавляет middleware к роутеру
	Use(middleware types.Middleware)

	// Handle регистрирует обработчик команды
	Handle(command string, handler types.HandlerFunc)

	// Dispatch отправляет команду к обработчику
	Dispatch(ctx types.Context) error
}
