package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// Song представляет модель песни
type Song struct {
	ID          int    `json:"id"`
	GroupName   string `json:"group_name"`             // Название группы
	SongTitle   string `json:"song_title"`             // Название песни
	ReleaseDate string `json:"release_date,omitempty"` // Дата выпуска песни
	Text        string `json:"text,omitempty"`         // Текст песни
	Link        string `json:"link,omitempty"`         // Ссылка на песню
}

// AddSongRequest представляет данные для добавления новой песни в формате задания 1
type AddSongRequest struct {
	Group string `json:"group"` // Название группы
	Song  string `json:"song"`  // Название песни
}

// SongDetail описывает ответ внешнего API
type SongDetail struct {
	ReleaseDate string `json:"releaseDate"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}

// Handler содержит ссылку на базу данных
type Handler struct {
	DB *sql.DB
}

// ListSongsHandler возвращает список всех песен с фильтрацией и пагинацией
// @Summary Получение списка песен
// @Description Возвращает массив песен с базовой информацией, включая фильтрацию и пагинацию
// @Tags Songs
// @Produce json
// @Param group_name query string false "Название группы"
// @Param song_title query string false "Название песни"
// @Param release_date query string false "Дата выпуска (YYYY-MM-DD)"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param limit query int false "Количество записей на странице (по умолчанию 10)"
// @Success 200 {array} Song
// @Failure 500 {object} string "Ошибка выполнения запроса"
// @Router /songs [get]
func (h *Handler) ListSongsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Чтение параметров фильтрации
	groupName := r.URL.Query().Get("group_name")
	songTitle := r.URL.Query().Get("song_title")
	releaseDate := r.URL.Query().Get("release_date")

	// Чтение параметров пагинации
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	// SQL-запрос для фильтрации
	query := `SELECT id, group_name, song_title 
              FROM songs
              WHERE ($1 = '' OR group_name ILIKE '%' || $1 || '%')
                AND ($2 = '' OR song_title ILIKE '%' || $2 || '%')
                AND ($3 = '' OR release_date = TO_DATE($3, 'YYYY-MM-DD'))
              LIMIT $4 OFFSET $5`

	rows, err := h.DB.Query(query, groupName, songTitle, releaseDate, limit, offset)
	if err != nil {
		log.Printf("Ошибка выполнения запроса: %v", err)
		http.Error(w, "Ошибка выполнения запроса", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Сканирование результатов
	var songs []Song
	for rows.Next() {
		var song Song
		if err := rows.Scan(&song.ID, &song.GroupName, &song.SongTitle); err != nil {
			log.Printf("Ошибка чтения данных: %v", err)
			http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
			return
		}
		songs = append(songs, song)
	}

	// Отправка результата
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(songs)
}

// GetSongHandler возвращает текст песни с пагинацией по куплетам
// @Summary Получение текста песни
// @Description Возвращает текст песни по ID. Если указан параметр verse, возвращается только соответствующий куплет текста.
// @Tags Songs
// @Produce json
// @Param id path int true "ID песни"
// @Param verse query int false "Номер куплета (по умолчанию полный текст)"
// @Success 200 {object} Song
// @Failure 400 {object} string "Некорректный запрос"
// @Failure 404 {object} string "Песня не найдена или куплет отсутствует"
// @Router /songs/{id} [get]
func (h *Handler) GetSongHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		log.Printf("Некорректный ID: %v", err)
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	var song Song
	var text, link *string // Указатели для полей, которые могут быть NULL
	query := `SELECT id, group_name, song_title, text, link FROM songs WHERE id = $1`
	err = h.DB.QueryRow(query, id).Scan(&song.ID, &song.GroupName, &song.SongTitle, &text, &link)
	if err == sql.ErrNoRows {
		log.Printf("Песня с ID %d не найдена", id)
		http.Error(w, "Песня не найдена", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Ошибка выполнения запроса: %v", err)
		http.Error(w, "Ошибка выполнения запроса", http.StatusInternalServerError)
		return
	}

	// Если поля text или link NULL, задаётся их значения по умолчанию
	if text != nil {
		song.Text = *text
	} else {
		song.Text = ""
	}
	if link != nil {
		song.Link = *link
	} else {
		song.Link = ""
	}

	// Пагинация по куплетам
	verseParam := r.URL.Query().Get("verse")
	if verseParam != "" {
		verse, err := strconv.Atoi(verseParam)
		if err != nil || verse <= 0 {
			log.Printf("Некорректный номер куплета: %v", err)
			http.Error(w, "Некорректный номер куплета", http.StatusBadRequest)
			return
		}

		// Разделение текста на куплеты
		verses := strings.Split(song.Text, "\n\n")
		if verse > len(verses) {
			log.Printf("Куплет %d отсутствует", verse)
			http.Error(w, "Куплет отсутствует", http.StatusNotFound)
			return
		}

		// Возвращение только указанного куплета
		song.Text = verses[verse-1]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(song)
}

// AddSongHandler добавляет новую песню
// @Summary Добавление новой песни
// @Description Добавляет новую песню в библиотеку. Если переменная окружения EXTERNAL_API_URL пустая, используется мок-режим для получения данных.
// @Tags Songs
// @Accept json
// @Produce json
// @Param song body AddSongRequest true "Данные песни"
// @Success 201 {object} Song
// @Failure 400 {object} string "Некорректные входные данные"
// @Failure 500 {object} string "Ошибка выполнения запроса"
// @Router /songs [post]
func (h *Handler) AddSongHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Проверка формата входных данных
	var input AddSongRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("Некорректные входные данные: %v", err)
		http.Error(w, "Некорректные входные данные", http.StatusBadRequest)
		return
	}

	// Добавление песни в базу данных с базовыми полями
	query := `INSERT INTO songs (group_name, song_title) VALUES ($1, $2) RETURNING id`
	var id int
	err := h.DB.QueryRow(query, input.Group, input.Song).Scan(&id)
	if err != nil {
		log.Printf("Ошибка выполнения запроса: %v", err)
		http.Error(w, "Ошибка выполнения запроса", http.StatusInternalServerError)
		return
	}

	// Выполнение запроса к внешнему API для обогощения записи
	details, err := fetchSongDetails(input.Group, input.Song)
	if err != nil {
		log.Printf("Ошибка при получении данных из внешнего API: %v", err)
		log.Println("Данные для песни не обогащены, завершение с базовыми данными")
	} else {
		// Обновление записи в базе данными из API
		updateQuery := `UPDATE songs SET release_date = $1, text = $2, link = $3 WHERE id = $4`
		_, updateErr := h.DB.Exec(updateQuery, details.ReleaseDate, details.Text, details.Link, id)
		if updateErr != nil {
			log.Printf("Ошибка обновления записи: %v", updateErr)
		}
	}

	// Возврат результата
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// fetchSongDetails выполняет запрос к внешнему API "EXTERNAL_API_URL" в .env
func fetchSongDetails(group, song string) (*SongDetail, error) {
	apiURL := os.Getenv("EXTERNAL_API_URL")

	// Если URL не задан, возвращаются мок-данные
	if apiURL == "" {
		log.Println("EXTERNAL_API_URL не задан, получены мок-данные")
		return &SongDetail{
			ReleaseDate: "2006-07-16",
			Text: `Ooh baby, don't you know I suffer?
Ooh baby, can you hear me moan?
You caught me under false pretenses
How long before you let me go?

Ooh
You set my soul alight
Ooh
You set my soul alight`,
			Link: "https://www.youtube.com/watch?v=Xsp3_a-PMTw",
		}, nil
	}

	// Формирование запроса
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Ошибка создания HTTP-запроса: %v", err)
	}

	// Добавление параметров запроса
	query := req.URL.Query()
	query.Add("group", group)
	query.Add("song", song)
	req.URL.RawQuery = query.Encode()

	// Выполнение запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ошибка выполнения HTTP-запроса: %v", err)
	}
	defer resp.Body.Close()

	// Проверка статуса ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Неудачная попытка запроса. Статус ответа: %d", resp.StatusCode)
	}

	// Декодирование ответа
	var details SongDetail
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("Ошибка декодирования ответа: %v", err)
	}

	// Проверяем, что ответ соответствует ожидаемому формату
	if details.ReleaseDate == "" || details.Text == "" || details.Link == "" {
		return nil, fmt.Errorf("Получены неполные или некорректные данные от API")
	}

	return &details, nil
}

// UpdateSongHandler обновляет данные о песне
// @Summary Обновление данных о песне
// @Description Обновляет данные о песне по её идентификатору
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "ID песни"
// @Param song body Song true "Обновлённые данные песни"
// @Success 200 {string} string "Песня обновлена"
// @Failure 400 {object} string "Некорректные входные данные"
// @Failure 404 {object} string "Песня не найдена"
// @Failure 500 {object} string "Ошибка выполнения запроса"
// @Router /songs/{id} [put]
func (h *Handler) UpdateSongHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	var song Song
	if err := json.NewDecoder(r.Body).Decode(&song); err != nil {
		http.Error(w, "Некорректные входные данные", http.StatusBadRequest)
		return
	}

	// Обновление записи в базе данными из запроса
	query := `UPDATE songs SET group_name=$1, song_title=$2, release_date=$3, text=$4, link=$5, updated_at=NOW()
			  WHERE id=$6`
	result, err := h.DB.Exec(query, song.GroupName, song.SongTitle, song.ReleaseDate, song.Text, song.Link, id)
	if err != nil {
		log.Printf("Ошибка при обновлении песни: %v", err)
		http.Error(w, "Ошибка выполнения запроса", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Песня не найдена", http.StatusNotFound)
		return
	}

	// Возврат результата
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Песня обновлена"))
}

// DeleteSongHandler удаляет песню
// @Summary Удаление песни
// @Description Удаляет песню по её идентификатору
// @Tags Songs
// @Produce plain
// @Param id path int true "ID песни"
// @Success 200 {string} string "Песня удалена"
// @Failure 400 {object} string "Некорректный ID"
// @Failure 404 {object} string "Песня не найдена"
// @Failure 500 {object} string "Ошибка выполнения запроса"
// @Router /songs/{id} [delete]
func (h *Handler) DeleteSongHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		http.Error(w, "Некорректный ID", http.StatusBadRequest)
		return
	}

	// Удаление записи из базы данных
	query := `DELETE FROM songs WHERE id=$1`
	result, err := h.DB.Exec(query, id)
	if err != nil {
		log.Printf("Ошибка при удалении песни: %v", err)
		http.Error(w, "Ошибка выполнения запроса", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Песня не найдена", http.StatusNotFound)
		return
	}

	// Возврат результата
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Песня удалена"))
}
