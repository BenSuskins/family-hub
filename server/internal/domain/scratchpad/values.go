package scratchpad

import "time"

type Point struct{ X, Y float64 }
type Stroke struct {
	Color  string
	Points []Point
}
type Snapshot struct {
	Strokes   []Stroke
	UpdatedAt time.Time
}
type Drawing struct{ strokes []Stroke }

func NewDrawing(strokes []Stroke) (Drawing, error) {
	if err := Validate(strokes); err != nil {
		return Drawing{}, err
	}
	return Drawing{strokes: CopyStrokes(strokes)}, nil
}
func (drawing Drawing) Strokes() []Stroke { return CopyStrokes(drawing.strokes) }
func CopyStrokes(strokes []Stroke) []Stroke {
	if strokes == nil {
		return nil
	}
	result := make([]Stroke, len(strokes))
	for index, stroke := range strokes {
		result[index] = stroke
		if stroke.Points != nil {
			result[index].Points = append([]Point{}, stroke.Points...)
		}
	}
	return result
}
