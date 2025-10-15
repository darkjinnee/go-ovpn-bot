# OpenVPN Client Management Scripts

Набор скриптов для управления клиентами OpenVPN сервера. Позволяет добавлять и удалять клиентов с автоматической генерацией конфигурационных файлов.

## 📁 Файлы

- `add.sh` - Добавление нового клиента OpenVPN
- `remove.sh` - Удаление существующего клиента OpenVPN
- `openvpn-install.sh` - Основной скрипт установки OpenVPN сервера

## 🔧 Требования

- Linux система с установленным OpenVPN сервером
- Права root (sudo) для выполнения скриптов
- Установленный easy-rsa для управления сертификатами

## 📋 Скрипт add.sh

### Описание
Создает нового клиента OpenVPN с генерацией сертификатов и конфигурационного файла.

### Синтаксис
```bash
sudo ./add.sh <client_name> [output_directory] [password_protected]
```

### Параметры
- `client_name` (обязательный) - Имя клиента (только буквы, цифры, подчеркивания, дефисы)
- `output_directory` (опциональный) - Директория для сохранения конфига (по умолчанию: папка со скриптом)
- `password_protected` (опциональный) - Строка "password" для защиты клиента паролем

### Примеры использования
```bash
# Создать клиента без пароля в папке со скриптом
CONFIG_PATH=$(sudo ./add.sh myclient)
echo "Config saved to: $CONFIG_PATH"

# Создать клиента в указанной директории
CONFIG_PATH=$(sudo ./add.sh myclient /tmp/vpn-configs)

# Создать клиента с паролем в указанной директории
CONFIG_PATH=$(sudo ./add.sh myclient /tmp/vpn-configs password)
```

### Коды возврата
- `0` - Успешное создание клиента
- `1` - Ошибка выполнения

### Вывод
- **Успех**: Путь к созданному конфигурационному файлу
- **Ошибка**: Сообщение об ошибке в stderr

### Возможные ошибки
- `❌ This script must be run as root (use sudo)` - Недостаточно прав
- `❌ OpenVPN server is not installed!` - OpenVPN не установлен
- `❌ Invalid client name` - Некорректное имя клиента
- `❌ Client 'name' already exists!` - Клиент уже существует
- `❌ Cannot create output directory` - Не удалось создать директорию
- `❌ Output directory is not writable` - Нет прав на запись в директорию
- `❌ Cannot access /etc/openvpn/easy-rsa/` - Ошибка доступа к easy-rsa

## 🗑️ Скрипт remove.sh

### Описание
Удаляет существующего клиента OpenVPN, отзывает сертификат и удаляет конфигурационные файлы.

### Синтаксис
```bash
sudo ./remove.sh <client_name> [config_file_path]
sudo ./remove.sh --list
```

### Параметры
- `client_name` (обязательный) - Имя клиента для удаления
- `config_file_path` (опциональный) - Путь к конфигурационному файлу для удаления
- `--list` или `-l` - Показать список всех клиентов

### Примеры использования
```bash
# Удалить клиента (поиск конфига в папке со скриптом)
if sudo ./remove.sh myclient; then
    echo "Client removed successfully"
else
    echo "Failed to remove client"
fi

# Удалить клиента с указанием пути к конфигу
sudo ./remove.sh myclient /path/to/myclient.ovpn

# Показать список всех клиентов
sudo ./remove.sh --list
```

### Коды возврата
- `0` - Успешное удаление клиента
- `1` - Ошибка выполнения

### Вывод
- **Успех**: Никакого вывода (только код возврата 0)
- **Ошибка**: Сообщение об ошибке в stderr

### Возможные ошибки
- `❌ This script must be run as root (use sudo)` - Недостаточно прав
- `❌ OpenVPN server is not installed!` - OpenVPN не установлен
- `❌ Client 'name' not found!` - Клиент не найден
- `❌ Cannot access /etc/openvpn/easy-rsa/` - Ошибка доступа к easy-rsa

## 🔄 Рабочий процесс

### Создание клиента
1. Выполните `add.sh` с именем клиента и опционально директорией/паролем
2. Скрипт создаст сертификат и конфиг файл
3. Получите путь к конфигу из вывода скрипта
4. Используйте конфиг для подключения клиента

### Удаление клиента
1. Выполните `remove.sh` с именем клиента
2. Скрипт отзовет сертификат и удалит конфиг
3. Клиент больше не сможет подключиться

### Пример полного цикла
```bash
#!/bin/bash

# Создать клиента
echo "Creating client..."
CONFIG_PATH=$(sudo ./add.sh testclient /tmp/vpn-configs password)
if [[ $? -eq 0 ]]; then
    echo "✅ Client created: $CONFIG_PATH"
else
    echo "❌ Failed to create client"
    exit 1
fi

# Использовать клиента...

# Удалить клиента
echo "Removing client..."
if sudo ./remove.sh testclient "$CONFIG_PATH"; then
    echo "✅ Client removed successfully"
else
    echo "❌ Failed to remove client"
    exit 1
fi
```

## 📁 Структура файлов

```
/opt/vpn-scripts/
├── add.sh                 # Скрипт добавления клиентов
├── remove.sh              # Скрипт удаления клиентов
├── openvpn-install.sh     # Основной скрипт установки
└── client1.ovpn          # Конфигурационные файлы клиентов
    client2.ovpn
    ...
```

## ⚠️ Важные замечания

1. **Безопасность**: Всегда запускайте скрипты с правами root
2. **Резервные копии**: Скрипты создают резервные копии важных файлов
3. **Необратимость**: Отзыв сертификата необратим
4. **Совместимость**: Скрипты работают с существующими установками OpenVPN

## 🐛 Отладка

### Проверка установки OpenVPN
```bash
ls -la /etc/openvpn/server.conf
```

### Проверка easy-rsa
```bash
ls -la /etc/openvpn/easy-rsa/
```

### Просмотр логов
```bash
journalctl -u openvpn@server -f
```

### Проверка сертификатов
```bash
ls -la /etc/openvpn/easy-rsa/pki/issued/
```

## 📞 Поддержка

При возникновении проблем проверьте:
1. Права доступа (запуск с sudo)
2. Установку OpenVPN сервера
3. Доступность easy-rsa
4. Корректность параметров командной строки