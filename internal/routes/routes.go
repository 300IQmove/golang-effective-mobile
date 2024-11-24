package routes

import (
	"database/sql"

	"github.com/300IQmove/golang-effective-mobile/internal/handlers"
	"github.com/julienschmidt/httprouter"
)

// SetupRoutes регистрирует маршруты приложения
func SetupRoutes(db *sql.DB) *httprouter.Router {
	router := httprouter.New()

	h := &handlers.Handler{DB: db}

	router.GET("/songs", h.ListSongsHandler)         // Получение списка песен
	router.GET("/songs/:id", h.GetSongHandler)       // Получение текста песни
	router.POST("/songs", h.AddSongHandler)          // Добавление новой песни
	router.PUT("/songs/:id", h.UpdateSongHandler)    // Обновление данных о песне
	router.DELETE("/songs/:id", h.DeleteSongHandler) // Удаление песни

	return router
}
