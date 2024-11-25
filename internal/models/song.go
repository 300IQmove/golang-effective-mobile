package models

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
