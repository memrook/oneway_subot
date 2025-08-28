# 🚀 Quick Start Guide

## Быстрый запуск за 3 минуты

### 1. 📋 Настройка окружения
```bash
# Скопируйте конфигурацию
cp env.example .env

# Отредактируйте переменные окружения
nano .env  # или любой другой редактор
```

**Обязательные переменные:**
- `BOT_TOKEN` - токен вашего Telegram бота
- `CHANNEL_ID` - ID канала для публикации тикетов  
- `SUPERGROUP_ID` - ID группы для обсуждения
- `MONGO_ROOT_PASSWORD` - пароль для MongoDB

### 2. 🚀 Запуск приложения

**Простой запуск:**
```bash
./deploy.sh
```

**Или через Docker Compose:**
```bash
docker-compose up -d --build
```

### 3. ✅ Проверка работы

```bash
# Статус контейнеров
docker-compose ps

# Логи бота
docker-compose logs -f bot

# Проверка здоровья
curl http://localhost:8081/health
```

## 🛠 Разработка

### Локальная разработка
```bash
# Установка инструментов
make setup

# Запуск с hot reload
make dev

# Тестирование
make test

# Форматирование кода
make fmt
```

### Основные команды
```bash
make help           # Показать все команды
make up             # Запуск через Docker
make down           # Остановка сервисов
make logs           # Просмотр логов
make clean          # Очистка
```

## 🐳 Docker команды

```bash
# Сборка и запуск
docker-compose up -d --build

# Только база данных для разработки
docker-compose up -d mongo redis

# Остановка
docker-compose down

# Логи определенного сервиса
docker-compose logs -f bot

# Подключение к контейнеру
docker-compose exec bot sh
docker-compose exec mongo mongosh
```

## 🔧 Полезные ссылки

- **Логи:** `docker-compose logs -f bot`
- **База данных:** `docker-compose exec mongo mongosh -u root`
- **Health check:** http://localhost:8081/health
- **Метрики:** http://localhost:9090/metrics (если включен мониторинг)

## ❗ Troubleshooting

**Проблема:** Контейнеры не запускаются
```bash
# Проверить логи
docker-compose logs

# Пересобрать с нуля
docker-compose down --volumes
docker-compose up -d --build
```

**Проблема:** База данных недоступна
```bash
# Проверить статус MongoDB
docker-compose logs mongo

# Пересоздать volume
docker-compose down --volumes
docker volume prune
docker-compose up -d
```

**Проблема:** Бот не отвечает
1. Проверьте `BOT_TOKEN` в `.env`
2. Убедитесь, что бот добавлен в канал и группу
3. Проверьте права бота (должен быть администратором)

## 🎯 Готово!

Ваш бот техподдержки запущен и готов к работе! 🎉
