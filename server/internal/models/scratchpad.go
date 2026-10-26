package models

import "time"

type ScratchpadPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ScratchpadStroke struct {
	Color  string            `json:"color"`
	Points []ScratchpadPoint `json:"points"`
}

type Scratchpad struct {
	Strokes   []ScratchpadStroke `json:"strokes"`
	UpdatedAt time.Time          `json:"updatedAt"`
}
