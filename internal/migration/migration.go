package migration

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

// Обработка миграций, по умолчанию "up"
// Опциональный вызов для "down" миграции таблицы или базы данных соответственно:
// "go run main.go migrate down table" и "go run main.go migrate down database"
func HandleMigrations(args []string, dsnNoDB string) {
	switch {
	case len(args) == 0:
		log.Fatal("Укажите параметр 'down' для команды 'migrate'.")
	case args[0] == "up":
		// Проверка и создание базы данных
		CreateDatabase(dsnNoDB, os.Getenv("DB_NAME"))

		// Обогащение Data Source Name именем базы данных для подключения
		dsnWithDB := fmt.Sprintf("%s dbname=%s", dsnNoDB, os.Getenv("DB_NAME"))

		// Миграция таблиц
		RunSQLFile(dsnWithDB, "internal/migrations/sql/001_create_songs_table.up.sql")
		log.Println("Миграции 'up' успешно выполнены!")
	// Обработка "down" миграций
	case len(args) > 0 && args[0] == "down":
		if len(args) < 2 {
			log.Fatal("Укажите параметр 'database' или 'table' для команды 'down'.")
		}

		switch args[1] {
		// Откат базы данных
		case "database":
			DropDatabase(dsnNoDB, os.Getenv("DB_NAME"))
			log.Println("Миграция 'down' для базы данных успешно выполнена!")
		// Откат таблицы
		case "table":
			dsnWithDB := fmt.Sprintf("%s dbname=%s", dsnNoDB, os.Getenv("DB_NAME"))
			RunSQLFile(dsnWithDB, "internal/migrations/sql/001_create_songs_table.down.sql")
			log.Println("Миграция 'down' для таблицы успешно выполнена!")
		default:
			log.Fatalf("Неизвестный параметр для 'down': %s", args[1])
		}
	default:
		log.Fatalf("Неизвестная команда для миграций: %s", args[0])
	}
}

func CreateDatabase(dsn, dbName string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	defer db.Close()

	// Проверка существования базы данных
	query := fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s'", dbName)
	var exists int
	err = db.QueryRow(query).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("Ошибка проверки существования базы данных: %v", err)
	}

	// Создание базы данных, если не существует
	if exists == 0 {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			log.Fatalf("Ошибка создания базы данных: %v", err)
		}
		log.Printf("База данных '%s' успешно создана.", dbName)
	} else {
		log.Printf("База данных '%s' уже существует.", dbName)
	}
}

func DropDatabase(dsn, dbName string) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	defer db.Close()

	// Удаление базу данных, если существует
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	if err != nil {
		log.Fatalf("Ошибка удаления базы данных: %v", err)
	}
	log.Printf("База данных '%s' успешно удалена.", dbName)
}

// Обработчик "up" и "down" скриптов для таблицы
func RunSQLFile(dsn, filePath string) {
	// Подключение к PostgreSQL
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	defer db.Close()

	// Чтение SQL-файла
	query, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Ошибка чтения SQL-файла: %v", err)
	}

	// Выполнение SQL-запроса
	_, err = db.Exec(string(query))
	if err != nil {
		log.Fatalf("Ошибка выполнения SQL: %v", err)
	}

	log.Printf("SQL-скрипт из %s выполнен успешно.", filePath)
}
