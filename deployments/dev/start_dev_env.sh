#!/bin/bash

# Скрипт для запуска локального окружения разработки

set -e

echo "Starting development environment..."

# Проверяем, установлен ли docker-compose
if ! command -v docker-compose &>/dev/null; then
  echo "docker-compose could not be found. Please install Docker and Docker Compose."
  exit 1
fi

# Запускаем сервисы в фоновом режиме
docker-compose up -d

echo "Waiting for services to be ready..."

# Проверяем, готов ли PostgreSQL
until docker-compose exec postgres pg_isready -U student_user -d student_schedule_dev >/dev/null 2>&1; do
  sleep 1
done

echo "PostgreSQL is ready!"

# Проверяем, готов ли Redis
until docker-compose exec redis redis-cli ping >/dev/null 2>&1; do
  sleep 1
done

echo "Redis is ready!"

echo "Development environment is up and running!"
echo "PostgreSQL: localhost:5432"
echo "Redis: localhost:6379"
