#!/bin/bash

# Скрипт для генерации Go кода из proto файлов

set -e

echo "Генерация Go кода из proto файлов..."

# Создаем папку для сгенерированного кода
mkdir -p proto/gen

# Генерируем Go код из users.proto
protoc --proto_path=proto \
  --go_out=proto/gen \
  --go-grpc_out=proto/gen \
  proto/users.proto

# Генерируем Go код из schedule.proto
protoc --proto_path=proto \
  --go_out=proto/gen \
  --go-grpc_out=proto/gen \
  proto/schedule.proto

echo "Генерация завершена успешно!"
