package cache

// CommandCacheInterface определяет интерфейс для кэша команд
type CommandCacheInterface interface {
	// Get получает значение из кэша
	Get(key string) (any, bool)

	// Set устанавливает значение в кэш
	Set(key string, data any)

	// Delete удаляет значение из кэша
	Delete(key string)

	// Clear очищает весь кэш
	Clear()

	// Stats возвращает статистику кэша
	Stats() map[string]any
}
