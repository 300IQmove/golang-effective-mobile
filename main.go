package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/300IQmove/golang-effective-mobile/docs"
	"github.com/300IQmove/golang-effective-mobile/internal/migration"
	"github.com/300IQmove/golang-effective-mobile/internal/routes"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	// Загрузка переменных окружения
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	// Сборка Data Source Name для подключение к PostgreSQL без указания имени базы данных
	dsnNoDB := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_SSLMODE"),
	)

	// Запуск миграции через аргументы без запуска сервера
	args := os.Args
	if len(args) > 1 {
		switch args[1] {
		case "migrate":
			migration.HandleMigrations(args[2:], dsnNoDB)
		default:
			log.Printf("Неизвестная команда: %s", args[1])
		}
		return
	}

	// Применение Up миграций по умолчанию
	migration.HandleMigrations([]string{"up"}, dsnNoDB)

	// Запуск сервера
	runServer(dsnNoDB)
}

func runServer(dsnNoDB string) {
	// Обогащение Data Source Name именем базы данных для подключения
	dsnWithDB := fmt.Sprintf("%s dbname=%s", dsnNoDB, os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", dsnWithDB)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Настройка маршрутов
	router := routes.SetupRoutes(db)
	router.Handler("GET", "/swagger/*any", httpSwagger.WrapHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Сервер запущен на порту %s", port)
	log.Println("Swagger документация: http://localhost:8080/swagger/index.html")
	log.Fatal(http.ListenAndServe(":"+port, router))
}
