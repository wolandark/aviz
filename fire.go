package main

import (
	"math"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

type FireVisualizer struct {
	heatmap [][]float64
	prevW   int
	prevH   int
}

func NewFireVisualizer() *FireVisualizer {
	return &FireVisualizer{}
}

func (fv *FireVisualizer) Name() string { return "fire" }

func (fv *FireVisualizer) initHeatmap(w, h int) {
	if fv.prevW == w && fv.prevH == h && fv.heatmap != nil {
		return
	}
	fv.heatmap = make([][]float64, h)
	for y := range fv.heatmap {
		fv.heatmap[y] = make([]float64, w)
	}
	fv.prevW = w
	fv.prevH = h
}

func (fv *FireVisualizer) Draw(screen tcell.Screen, spectrum []float64, rawSamples []float64, w, h int, scheme ColorScheme, cfg *Config) {
	drawH := h - 1
	if !cfg.Visual.ShowStatus {
		drawH = h
	}
	if drawH < 3 || w < 2 {
		return
	}

	fv.initHeatmap(w, drawH)

	data := resample(spectrum, w)

	for x := 0; x < w; x++ {
		val := clamp(data[x], 0, 1) * cfg.Visual.Sensitivity
		fv.heatmap[drawH-1][x] = val
		if drawH > 1 {
			fv.heatmap[drawH-2][x] = val * (0.8 + rand.Float64()*0.2)
		}
	}

	for y := drawH - 3; y >= 0; y-- {
		for x := 0; x < w; x++ {
			below := fv.heatmap[y+1][x]
			belowBelow := fv.heatmap[y+2][x]

			left := below
			right := below
			if x > 0 {
				left = fv.heatmap[y+1][x-1]
			}
			if x < w-1 {
				right = fv.heatmap[y+1][x+1]
			}

			avg := (below*3 + belowBelow + left + right) / 6.0

			decay := 0.82 + rand.Float64()*0.08
			fv.heatmap[y][x] = avg * decay

			if fv.heatmap[y][x] < 0.01 {
				fv.heatmap[y][x] = 0
			}
		}
	}

	for x := 1; x < w-1; x++ {
		for y := 0; y < drawH; y++ {
			fv.heatmap[y][x] = fv.heatmap[y][x]*0.6 +
				(fv.heatmap[y][x-1]+fv.heatmap[y][x+1])*0.2
		}
	}

	for cy := 0; cy < drawH; cy++ {
		for x := 0; x < w; x++ {
			heat := fv.heatmap[cy][x]
			if heat < 0.03 {
				continue
			}

			color := fireColor(heat, scheme)

			var ch rune
			if heat > 0.6 {
				ch = '█'
			} else if heat > 0.3 {
				ch = '▓'
			} else if heat > 0.12 {
				ch = '▒'
			} else {
				ch = '░'
			}

			st := tcell.StyleDefault.Foreground(color)
			screen.SetContent(x, cy, ch, nil, st)
		}
	}
}

func fireColor(heat float64, scheme ColorScheme) tcell.Color {
	t := math.Pow(clamp(heat, 0, 1), 0.5)
	color := scheme.At(t)

	if heat < 0.2 {
		r, g, b := color.RGB()
		f := heat / 0.2
		return tcell.NewRGBColor(
			int32(float64(r)*f),
			int32(float64(g)*f),
			int32(float64(b)*f),
		)
	}

	return color
}
