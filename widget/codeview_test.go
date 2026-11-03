package widget

import (
	"reflect"
	"testing"
)

// sampleLines is this JSON, pretty printed, as lines (index: text):
// 0 {  1   "a": 1,  2   "items": [  3     {  4       "id": 2,
// 5       "o": {  6         "x": 3  7       }  8     },  9     {
// 10      "id": 4  11    }  12  ],  13  "z": 5  14 }
func sampleLines() []Line {
	texts := []struct {
		text     string
		foldable bool
	}{
		{"{", true}, {`  "a": 1,`, false}, {`  "items": [`, true}, {"    {", true},
		{`      "id": 2,`, false}, {`      "o": {`, true}, {`        "x": 3`, false},
		{"      }", false}, {"    },", false}, {"    {", true}, {`      "id": 4`, false},
		{"    }", false}, {"  ],", false}, {`  "z": 5`, false}, {"}", false},
	}
	lines := make([]Line, len(texts))
	for i, t := range texts {
		lines[i] = Line{Spans: []Span{{Text: t.text}}, Foldable: t.foldable}
	}
	return lines
}

func TestCodeViewVisibleFolds(t *testing.T) {
	all := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
	tests := []struct {
		name string
		fold []int
		want []int
	}{
		{"none", nil, all},
		{"fold items", []int{2}, []int{0, 1, 2, 13, 14}},
		{"fold first element", []int{3}, []int{0, 1, 2, 3, 9, 10, 11, 12, 13, 14}},
		{"fold root", []int{0}, []int{0}},
		{"fold inside folded still hidden", []int{2, 5}, []int{0, 1, 2, 13, 14}},
		{"fold on non-foldable ignored", []int{1}, all},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v CodeView
			v.SetLines(sampleLines())
			for _, i := range tt.fold {
				v.ToggleFold(i)
			}
			if got := v.Visible(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodeViewToggleTwiceUnfolds(t *testing.T) {
	var v CodeView
	v.SetLines(sampleLines())
	v.ToggleFold(2)
	v.ToggleFold(2)
	if len(v.Visible()) != 15 {
		t.Fatal("second toggle must unfold")
	}
	v.ToggleFold(-1)
	v.ToggleFold(99) // out of range must not panic
}

func TestCodeViewScrollClamps(t *testing.T) {
	tests := []struct {
		name                   string
		start, dy, content, vh int
		want                   int
	}{
		{"down", 0, 50, 1000, 400, 50},
		{"past end", 500, 500, 1000, 400, 600},
		{"above top", 10, -50, 1000, 400, 0},
		{"content shorter than view", 0, 80, 100, 400, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := CodeView{ScrollY: tt.start}
			v.Scroll(tt.dy, tt.content, tt.vh)
			if v.ScrollY != tt.want {
				t.Fatalf("ScrollY = %d, want %d", v.ScrollY, tt.want)
			}
		})
	}
}
