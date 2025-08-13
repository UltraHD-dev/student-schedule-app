package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Драйвер PostgreSQL для Goose
	"github.com/pressly/goose/v3"
)

const (
	defaultDBString = "host=localhost port=5432 user=student_user password=student_pass dbname=student_schedule_dev sslmode=disable"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", "./migrations", "directory with migration files")
)

func main() {
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) < 1 {
		log.Fatal("Ошибка: необходимо указать подкоманду (up, down, status и т.д.)")
	}

	db, err := sql.Open("postgres", defaultDBString)
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка закрытия соединения БД: %v", err)
		}
	}()

	// Установим диалект для Goose
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Ошибка установки диалекта Goose: %v", err)
	}

	command := args[0]
	arguments := []string{}
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}

	if err := goose.Run(command, db, *dir, arguments...); err != nil {
		log.Fatalf("Ошибка выполнения команды goose '%s': %v", command, err)
	}

	fmt.Printf("Команда goose '%s' выполнена успешно\n", command)
}
