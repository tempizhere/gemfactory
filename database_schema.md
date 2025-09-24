# База данных GemFactory

## Общая информация

- **Название базы данных**: `gemfactory`
- **СУБД**: PostgreSQL
- **ORM**: Bun (github.com/uptrace/bun)

## Структура таблиц

### 1. Таблица `artists` (Артисты)

**Назначение**: Хранение информации о K-pop артистах и группах

| Поле | Тип | Ограничения | Описание |
|------|-----|-------------|----------|
| `artist_id` | SERIAL | PRIMARY KEY | Уникальный идентификатор |
| `name` | VARCHAR(255) | UNIQUE, NOT NULL | Имя артиста/группы |
| `gender` | VARCHAR(10) | NOT NULL, DEFAULT 'male' | Пол: 'female', 'male', 'mixed' |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Активность артиста |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата создания |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата обновления |

**Индексы**:

- `idx_artists_name` - поиск по имени
- `idx_artists_gender` - фильтрация по полу
- `idx_artists_is_active` - фильтрация активных
- `idx_artists_gender_active` - комбинированный индекс

**Триггеры**:

- `update_artists_updated_at` - автоматическое обновление `updated_at`

---

### 2. Таблица `release_types` (Типы релизов)

**Назначение**: Справочник типов релизов

| Поле | Тип | Ограничения | Описание |
|------|-----|-------------|----------|
| `release_type_id` | SERIAL | PRIMARY KEY | Уникальный идентификатор |
| `name` | VARCHAR(20) | UNIQUE, NOT NULL | Название типа: 'single', 'album', 'ep' |
| `description` | TEXT | NULL | Описание типа |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата создания |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата обновления |

**Индексы**:

- `idx_release_types_name` - поиск по имени

**Триггеры**:

- `update_release_types_updated_at` - автоматическое обновление `updated_at`

---

### 3. Таблица `releases` (Релизы)

**Назначение**: Хранение информации о K-pop релизах (синглы, альбомы, EP)

| Поле | Тип | Ограничения | Описание |
|------|-----|-------------|----------|
| `release_id` | SERIAL | PRIMARY KEY | Уникальный идентификатор |
| `artist_id` | INTEGER | NOT NULL, FK → artists(artist_id) | Ссылка на артиста |
| `release_type_id` | INTEGER | NOT NULL, FK → release_types(release_type_id) | Ссылка на тип релиза |
| `title` | VARCHAR(500) | NOT NULL | Название релиза |
| `title_track` | VARCHAR(255) | NULL | Название титульного трека |
| `album_name` | VARCHAR(255) | NULL | Название альбома |
| `mv` | TEXT | NULL | Ссылка на MV |
| `date` | VARCHAR(50) | NOT NULL | Дата релиза (DD.MM.YYYY) |
| `time_msk` | VARCHAR(10) | NULL | Время в MSK |
| `month` | VARCHAR(20) | NOT NULL | Месяц релиза |
| `year` | INTEGER | NOT NULL | Год релиза |
| `is_active` | BOOLEAN | NOT NULL, DEFAULT TRUE | Активность релиза |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата создания |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата обновления |

**Индексы**:

- `idx_releases_month` - поиск по месяцу
- `idx_releases_gender` - фильтрация по полу
- `idx_releases_artist` - поиск по артисту
- `idx_releases_type` - фильтрация по типу
- `idx_releases_is_active` - фильтрация активных
- `idx_releases_year` - поиск по году
- `idx_releases_type_gender` - комбинированный индекс
- `idx_releases_month_gender` - комбинированный индекс
- `idx_releases_album_name` - поиск по названию альбома
- `idx_releases_title_track` - поиск по титульному треку
- `idx_releases_time_msk` - поиск по времени

**Триггеры**:

- `update_releases_updated_at` - автоматическое обновление `updated_at`

---

### 4. Таблица `homework` (Домашние задания)

**Назначение**: Хранение домашних заданий пользователей (треки для прослушивания)

| Поле | Тип | Ограничения | Описание |
|------|-----|-------------|----------|
| `homework_id` | SERIAL | PRIMARY KEY | Уникальный идентификатор |
| `user_id` | BIGINT | NOT NULL | ID пользователя Telegram |
| `track_id` | VARCHAR(255) | NOT NULL | ID трека в Spotify |
| `artist` | VARCHAR(255) | NOT NULL | Имя артиста |
| `title` | VARCHAR(500) | NOT NULL | Название трека |
| `play_count` | INTEGER | NOT NULL, DEFAULT 1 | Количество прослушиваний |
| `completed` | BOOLEAN | NOT NULL, DEFAULT FALSE | Статус выполнения |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата создания |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата обновления |

**Индексы**:

- `idx_homework_user_id` - поиск по пользователю
- `idx_homework_completed` - фильтрация по статусу

**Триггеры**:

- `update_homework_updated_at` - автоматическое обновление `updated_at`

---

### 5. Таблица `playlists` (Плейлисты)

**Назначение**: Хранение информации о плейлистах Spotify

| Поле | Тип | Ограничения | Описание |
|------|-----|-------------|----------|
| `id` | SERIAL | PRIMARY KEY | Уникальный идентификатор |
| `spotify_id` | VARCHAR(255) | UNIQUE, NOT NULL | ID плейлиста в Spotify |
| `name` | VARCHAR(500) | NOT NULL | Название плейлиста |
| `description` | TEXT | NULL | Описание плейлиста |
| `owner` | VARCHAR(255) | NOT NULL | Владелец плейлиста |
| `track_count` | INTEGER | NOT NULL, DEFAULT 0 | Количество треков |
| `last_updated` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Последнее обновление |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата создания |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата обновления |

**Индексы**:

- `idx_playlists_spotify_id` - поиск по Spotify ID

**Триггеры**:

- `update_playlists_updated_at` - автоматическое обновление `updated_at`

---

### 6. Таблица `config` (Конфигурация)

**Назначение**: Хранение конфигурации приложения в базе данных

| Поле | Тип | Ограничения | Описание |
|------|-----|-------------|----------|
| `id` | SERIAL | PRIMARY KEY | Уникальный идентификатор |
| `key` | VARCHAR(255) | UNIQUE, NOT NULL | Ключ конфигурации |
| `value` | TEXT | NOT NULL | Значение конфигурации |
| `description` | TEXT | NULL | Описание параметра |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата создания |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | Дата обновления |

**Индексы**:

- `idx_config_key` - быстрый поиск по ключу

**Триггеры**:

- `update_config_updated_at` - автоматическое обновление `updated_at`

**Предустановленные ключи конфигурации**:

- `RATE_LIMIT_REQUESTS` - лимит запросов
- `RATE_LIMIT_WINDOW` - окно rate limiting
- `SCRAPER_DELAY` - задержка скрапера
- `SCRAPER_TIMEOUT` - таймаут скрапера
- `LOG_LEVEL` - уровень логирования
- `PLAYLIST_UPDATE_HOURS` - интервал обновления плейлиста
- `BOT_TOKEN` - токен Telegram бота
- `ADMIN_USERNAME` - имя администратора
- `SPOTIFY_CLIENT_ID` - ID клиента Spotify
- `SPOTIFY_CLIENT_SECRET` - секрет клиента Spotify
- `PLAYLIST_URL` - URL плейлиста
- `DB_DSN` - DSN базы данных
- `HEALTH_PORT` - порт health check

---

## Функции и триггеры

### Функция `update_updated_at_column()`

```sql
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';
```

**Назначение**: Автоматическое обновление поля `updated_at` при изменении записи

---

## Связи между таблицами

1. **artists ↔ releases**: Связь по внешнему ключу `artist_id` (FK)
2. **release_types ↔ releases**: Связь по внешнему ключу `release_type_id` (FK)
3. **homework**: Независимая таблица для домашних заданий
4. **playlists**: Независимая таблица для Spotify интеграции
5. **config**: Независимая таблица для конфигурации

---

## Особенности дизайна

### 1. Нормализованная структура с внешними ключами

- Связи между таблицами реализованы через внешние ключи
- Обеспечивает целостность данных и ссылочную целостность
- Позволяет эффективно работать с связанными данными через JOIN'ы

### 2. Гибкая система пола

- Поддержка трех значений: 'female', 'male', 'mixed'
- Поле `is_active` для мягкого удаления записей

### 3. Расширенные поля релизов

- `album_name` - для альбомов
- `title_track` - для титульных треков
- `time_msk` - для точного времени релиза
- `mv` - для ссылок на музыкальные видео

### 4. Конфигурация в БД

- Все настройки приложения хранятся в таблице `config`
- Динамическое изменение параметров без перезапуска
- Административные команды для управления конфигурацией

---

## Миграции

Файлы миграций в порядке выполнения:

1. `001_init.sql` - создание основных таблиц
2. `002_artists.sql` - начальные данные артистов
3. `003_releases.sql` - примеры релизов
4. `004_config.sql` - таблица конфигурации
5. `000002_add_release_fields.sql` - дополнительные поля релизов
6. `000003_simplify_artists.sql` - упрощение структуры артистов
7. `000004_improve_models.sql` - финальная структура с enum'ами

---

## Примеры запросов

### Получение релизов за месяц с информацией об артистах

```sql
SELECT r.*, a.name as artist_name, rt.name as release_type_name
FROM releases r
JOIN artists a ON r.artist_id = a.artist_id
JOIN release_types rt ON r.release_type_id = rt.release_type_id
WHERE r.month = 'january'
AND r.is_active = true
ORDER BY r.date ASC;
```

### Получение женских артистов

```sql
SELECT * FROM artists
WHERE gender = 'female'
AND is_active = true
ORDER BY name ASC;
```

### Получение релизов по типу

```sql
SELECT r.*, a.name as artist_name
FROM releases r
JOIN artists a ON r.artist_id = a.artist_id
JOIN release_types rt ON r.release_type_id = rt.release_type_id
WHERE rt.name = 'single'
ORDER BY r.date DESC;
```

### Получение конфигурации

```sql
SELECT key, value FROM config
WHERE key IN ('BOT_TOKEN', 'SPOTIFY_CLIENT_ID');
```

### Обновление конфигурации

```sql
UPDATE config
SET value = 'new_value', updated_at = CURRENT_TIMESTAMP
WHERE key = 'LOG_LEVEL';
```
