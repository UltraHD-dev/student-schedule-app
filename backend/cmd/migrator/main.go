// cmd/migrator/main.go
package main

import (
	"context"
	"database/sql" // <-- Добавлено
	"encoding/csv" // <-- Добавлено
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/config"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/scraper/gsheets"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func main() {
	// Определяем флаги командной строки
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}

	command := args[0]

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig("../../configs/config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключаемся к базе данных
	db, err := sql.Open("postgres", cfg.Database.GetDSN())
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Проверяем подключение к БД
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Ошибка проверки подключения к БД: %v", err)
	}

	log.Println("Успешное подключение к базе данных")

	switch command {
	case "up":
		if err := goose.Up(db, "../../migrations"); err != nil {
			log.Fatalf("Ошибка применения миграций: %v", err)
		}
		fmt.Println("Миграции успешно применены")
	case "down":
		if err := goose.Down(db, "../../migrations"); err != nil {
			log.Fatalf("Ошибка отката миграций: %v", err)
		}
		fmt.Println("Миграции успешно откачены")
	case "status":
		if err := goose.Status(db, "../../migrations"); err != nil {
			log.Fatalf("Ошибка получения статуса миграций: %v", err)
		}
	case "download-changes":
		// Новая команда для скачивания таблицы изменений
		if len(args) < 2 {
			log.Fatalf("Необходимо указать URL таблицы изменений")
		}
		changesURL := args[1]

		// Создаем клиент gsheets
		gsheetClient := gsheets.NewClient(cfg.Scraper.MainScheduleGIDs)

		// Скачиваем таблицу изменений в CSV
		ctx := context.Background()
		csvRecords, err := gsheetClient.ExportToCSVChanges(ctx, changesURL, cfg.Scraper.ChangesGID)
		if err != nil {
			log.Fatalf("Ошибка экспорта таблицы изменений: %v", err)
		}

		// Сохраняем в файл
		filename := fmt.Sprintf("changes_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
		file, err := os.Create(filename)
		if err != nil {
			log.Fatalf("Ошибка создания файла: %v", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		for _, record := range csvRecords {
			if err := writer.Write(record); err != nil {
				log.Fatalf("Ошибка записи в файл: %v", err)
			}
		}

		fmt.Printf("Таблица изменений успешно скачана в файл: %s\n", filename)
	case "parse-changes":
		// Новая команда для парсинга таблицы изменений из файла
		if len(args) < 2 {
			log.Fatalf("Необходимо указать путь к CSV файлу с изменениями")
		}
		filename := args[1]

		// Читаем файл
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Ошибка открытия файла: %v", err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		csvRecords, err := reader.ReadAll()
		if err != nil {
			log.Fatalf("Ошибка чтения CSV из файла: %v", err)
		}

		// Создаем клиент gsheets
		gsheetClient := gsheets.NewClient(cfg.Scraper.MainScheduleGIDs)

		// Парсим изменения
		changeRecords, err := gsheetClient.ParseChangeRecords(csvRecords)
		if err != nil {
			log.Fatalf("Ошибка парсинга изменений: %v", err)
		}

		fmt.Printf("Успешно распаршено %d записей изменений\n", len(changeRecords))
		for _, record := range changeRecords {
			fmt.Printf("Группа: %s, Дата: %s, Предмет: %s, Тип: %s\n",
				record.GroupName, record.Date.Format("02.01.2006"), record.Subject, record.ChangeType)
		}
	default:
		fmt.Printf("Неизвестная команда: %s\n", command)
		flag.Usage()
	}
}

func usage() {
	fmt.Println("Использование: migrator [команда]")
	fmt.Println("Доступные команды:")
	fmt.Println("  up                   - Применить все непримененные миграции")
	fmt.Println("  down                 - Откатить последнюю миграцию")
	fmt.Println("  status               - Показать статус миграций")
	fmt.Println("  download-changes URL - Скачать таблицу изменений по URL в CSV файл")
	fmt.Println("  parse-changes FILE   - Распарсить CSV файл с изменениями")
	fmt.Println("")
	fmt.Println("Примеры:")
	fmt.Println("  migrator up")
	fmt.Println("  migrator down")
	fmt.Println("  migrator status")
	fmt.Println("  migrator download-changes \"https://docs.google.com/spreadsheets/d/1bT7DPsioPz_OswGPXB6jgvXh20_S-i3d-C5NHCpeuIw/edit?usp=sharing\"")
	fmt.Println("  migrator parse-changes changes_2025-08-16_14-36-07.csv")
}
