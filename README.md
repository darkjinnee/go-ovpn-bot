# OpenVPN Telegram Bot

Telegram бот для управления VPN конфигурациями OpenVPN. Позволяет пользователям создавать и удалять VPN конфигурации через удобный интерфейс Telegram.

## 🚀 Возможности

- **Создание конфигураций**: Генерация имен с настраиваемым префиксом и случайными символами
- **Удаление конфигураций**: Интерактивное удаление через inline клавиатуру
- **Система лимитов**: Контроль количества конфигураций на пользователя
- **Коды активации**: Одноразовые коды для увеличения лимита конфигураций
- **База данных**: SQLite для хранения информации о пользователях, конфигурациях и кодах
- **Безопасность**: Интеграция с существующими скриптами OpenVPN

## 📋 Требования

- Go 1.21+
- OpenVPN сервер с установленными скриптами
- Telegram Bot Token
- Права sudo для выполнения скриптов OpenVPN

## 🛠 Установка

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd go-ovpn-bot
```

### 2. Инициализация проекта

```bash
make init
```

### 3. Настройка конфигурации

Отредактируйте файл `.env`:

```bash
nano .env
```

Обязательные параметры:
- `BOT_TOKEN` - токен вашего Telegram бота
- `SCRIPTS_PATH` - путь к скриптам OpenVPN (по умолчанию: ./scripts)
- `CONFIGS_PATH` - путь к директории с .ovpn файлами (по умолчанию: ./.ovpn)
- `CONFIG_PREFIX` - префикс для имен конфигурационных файлов (по умолчанию: VPN)

### 4. Создание Telegram бота

1. Найдите [@BotFather](https://t.me/botfather) в Telegram
2. Отправьте команду `/newbot`
3. Следуйте инструкциям для создания бота
4. Скопируйте полученный токен в файл `.env`

### 5. Запуск

```bash
make run
```

## 📁 Структура проекта

```
go-ovpn-bot/
├── cmd/bot/           # Точка входа приложения
├── internal/
│   ├── bot/           # Telegram Bot логика
│   ├── config/        # Конфигурация
│   ├── database/      # SQLite база данных
│   └── ovpn/          # OpenVPN сервис
├── scripts/           # Скрипты OpenVPN
├── .ovpn/            # Конфигурационные файлы
├── data/             # База данных SQLite
└── Makefile          # Команды сборки
```

## 🔧 Конфигурация

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `BOT_TOKEN` | Токен Telegram бота | - |
| `DATABASE_PATH` | Путь к SQLite базе | `./data/bot.db` |
| `SCRIPTS_PATH` | Путь к скриптам OpenVPN | `./scripts` |
| `CONFIGS_PATH` | Путь к .ovpn файлам | `./.ovpn` |
| `CONFIG_PREFIX` | Префикс для имен конфигураций | `VPN` |
| `DEBUG` | Режим отладки (true/false) | `false` |

### Формат имен конфигураций

Имена конфигурационных файлов генерируются по следующему шаблону:
- **Формат**: `{CONFIG_PREFIX}{8_случайных_символов}`
- **Символы**: латинские буквы (A-Z) и цифры (0-9)
- **Примеры**: `VPNoQugmyIG`, `VPNoDJTpcgh`, `VPNK9EO0fmH`

Префикс настраивается через переменную окружения `CONFIG_PREFIX` (по умолчанию: `VPN`).

### Настройка OpenVPN

Убедитесь что у вас установлен OpenVPN сервер и скрипты в директории `scripts/`:

- `add.sh` - для создания клиентов
- `remove.sh` - для удаления клиентов

## 🤖 Команды бота

- `/start` - Приветствие и информация о боте (показывает текущий лимит)
- `/add` - Создать новую VPN конфигурацию (проверяет лимит)
- `/remove` - Удалить существующую конфигурацию
- `/code` - Активировать код для увеличения лимита конфигураций

## 🔑 Система лимитов и кодов активации

### Принцип работы

1. **Новые пользователи** получают лимит = 0 по умолчанию
2. **Для создания конфигураций** необходимо активировать код командой `/code`
3. **Коды активации** - одноразовые, состоят из 10 символов (латинские буквы + цифры)
4. **Каждый код** имеет поле `limit`, которое добавляется к текущему лимиту пользователя
5. **Проверка лимита** происходит при каждой попытке создать конфигурацию

### Управление кодами

Администратор может создавать коды активации с помощью утилиты:

```bash
# Создать 5 кодов с лимитом 1
make generate-codes

# Создать коды с кастомными параметрами
make generate-codes-custom LIMIT=5 COUNT=10
```

### Структура кодов

- **Формат**: 10 символов (a-z, A-Z, 0-9)
- **Статус**: `active` (активный) или `used` (использованный)
- **Лимит**: количество конфигураций, которое добавляется к лимиту пользователя

## 🗄️ База данных

Проект использует SQLite для хранения информации о пользователях и их конфигурациях.

### Структура таблиц

#### Таблица `users`
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id INTEGER UNIQUE NOT NULL,
    username TEXT,
    limit_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### Таблица `configs`
```sql
CREATE TABLE configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
```

#### Таблица `activation_codes`
```sql
CREATE TABLE activation_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT UNIQUE NOT NULL,
    status TEXT DEFAULT 'active',
    limit_count INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### Индексы
```sql
CREATE INDEX idx_configs_user_id ON configs (user_id);
CREATE INDEX idx_activation_codes_code ON activation_codes (code);
```

### Связи между таблицами

- **Один ко многим**: Один пользователь может иметь несколько конфигураций
- **CASCADE DELETE**: При удалении пользователя автоматически удаляются все его конфигурации
- **Независимые**: Коды активации не связаны с пользователями напрямую

### Примеры данных

**Таблица users:**
```
id | telegram_id | username | limit_count | created_at
---|-------------|----------|-------------|-------------------
1  | 123456789   | john_doe | 2           | 2024-01-15 10:30:00
2  | 987654321   | jane_sm  | 0           | 2024-01-15 11:45:00
```

**Таблица configs:**
```
id | user_id | name        | file_path                 | created_at
---|---------|-------------|---------------------------|-------------------
1  | 1       | VPN3A0XYTS  | ./.ovpn/VPN3A0XYTS.ovpn   | 2024-01-15 10:35:00
2  | 1       | VPNJ0XOJ51  | ./.ovpn/VPNJ0XOJ51.ovpn   | 2024-01-15 12:20:00
```

**Таблица activation_codes:**
```
id | code        | status | limit_count | created_at
---|-------------|--------|-------------|-------------------
1  | AbC123XyZ9  | used   | 2           | 2024-01-15 09:00:00
2  | MxN456PqR7  | active | 1           | 2024-01-15 10:00:00
3  | KfG789LmN2  | active | 5           | 2024-01-15 11:00:00
```

### Работа с базой данных

```bash
# Подключиться к базе
sqlite3 data/bot.db

# Показать все таблицы
.tables

# Показать структуру таблиц
.schema users
.schema configs

# Показать данные
SELECT * FROM users;
SELECT * FROM configs;

# Показать конфигурации конкретного пользователя
SELECT c.* FROM configs c 
JOIN users u ON c.user_id = u.id 
WHERE u.telegram_id = 123456789;
```

## 🔒 Безопасность

- Бот использует существующие скрипты OpenVPN с правами sudo
- Каждый пользователь может управлять только своими конфигурациями
- Конфигурационные файлы хранятся в защищенной директории

## 🚀 Развертывание

### Systemd сервис

Создайте файл `/etc/systemd/system/ovpn-bot.service`:

```ini
[Unit]
Description=OpenVPN Telegram Bot
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/path/to/go-ovpn-bot
ExecStart=/path/to/go-ovpn-bot/bin/ovpn-bot
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Запустите сервис:

```bash
sudo systemctl enable ovpn-bot
sudo systemctl start ovpn-bot
```

### Docker (опционально)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o bot cmd/bot/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bot .
COPY --from=builder /app/scripts ./scripts
CMD ["./bot"]
```

## 🐛 Отладка

### Режим отладки

Для включения подробного логирования установите переменную окружения `DEBUG=true`:

```bash
# В .env файле
DEBUG=true

# Или при запуске
DEBUG=true ./bin/ovpn-bot
```

В режиме отладки бот выводит:
- Подробную информацию о полученных обновлениях
- Детали обработки сообщений от пользователей
- Дополнительные логи для диагностики

### Логи

Бот выводит логи в stdout. Для systemd:

```bash
sudo journalctl -u ovpn-bot -f
```

### Проверка базы данных

```bash
sqlite3 data/bot.db
.tables
SELECT * FROM users;
SELECT * FROM configs;
```

### Тестирование скриптов

```bash
# Проверка скрипта добавления
sudo ./scripts/add.sh test_client ./test_configs

# Проверка скрипта удаления
sudo ./scripts/remove.sh test_client ./test_configs/test_client.ovpn
```

## 📝 Разработка

### Установка зависимостей

```bash
make deps
```

### Сборка

```bash
make build
```

### Тесты

```bash
make test
```

### Очистка

```bash
make clean
```

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для новой функции
3. Внесите изменения
4. Создайте Pull Request

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. См. файл [LICENSE](LICENSE) для подробностей.

## 🆘 Поддержка

При возникновении проблем:

1. Проверьте логи бота
2. Убедитесь что OpenVPN сервер работает
3. Проверьте права доступа к скриптам
4. Создайте issue в репозитории