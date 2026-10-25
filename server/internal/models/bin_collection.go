package models

type BinCollection struct {
	Date string `json:"date"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type BinSchedule struct {
	Collections []BinCollection `json:"collections"`
	Enabled     bool            `json:"enabled"`
	UploadedAt  string          `json:"uploaded_at"`
}
