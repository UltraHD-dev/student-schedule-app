#!/bin/bash

# Скрипт для генерации Go кода из proto файлов

set -e

echo "Генерация Go кода из proto файлов..."

# Создаем папку для сгенерированного кода
mkdir -p proto/gen

# Генерируем Go код из users.proto
# Важно: proto_path должен указывать на корень, где лежит proto файл
# go_out и go-grpc_out указывают, куда класть сгенерированные файлы относительно proto_path
protoc --proto_path=. \
  --go_out=. \
  --go-grpc_out=. \
  proto/users.proto

echo "Генерация завершена успешно!"
