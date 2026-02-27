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
	if drawH < 2 || w < 2 {
		return
	}

	fv.initHeatmap(w, drawH)

	data := resample(spectrum, w)

	for x := 0; x < w; x++ {
		val := clamp(data[x], 0, 1) * cfg.Visual.Sensitivity

		fv.heatmap[drawH-1][x] = val + rand.Float64()*0.2*val
		if drawH > 1 {
			fv.heatmap[drawH-2][x] = val*0.85 + rand.Float64()*0.15*val
		}
		if drawH > 2 {
			fv.heatmap[drawH-3][x] = val*0.5 + rand.Float64()*0.1*val
		}
	}

	for y := 0; y < drawH-2; y++ {
		for x := 0; x < w; x++ {
			sum := 0.0
			count := 0

			if y+1 < drawH {
				sum += fv.heatmap[y+1][x]
				count++
			}
			if y+2 < drawH {
				sum += fv.heatmap[y+2][x]
				count++
			}
			if y+1 < drawH && x > 0 {
				sum += fv.heatmap[y+1][x-1]
				count++
			}
			if y+1 < drawH && x < w-1 {
				sum += fv.heatmap[y+1][x+1]
				count++
			}

			if count > 0 {
				decay := 0.85 + rand.Float64()*0.07
				fv.heatmap[y][x] = (sum / float64(count)) * decay
			}

			if rand.Float64() < 0.1 {
				fv.heatmap[y][x] += (rand.Float64() - 0.5) * 0.05
			}

			fv.heatmap[y][x] = clamp(fv.heatmap[y][x], 0, 1)
		}
	}

	for cy := 0; cy < drawH; cy++ {
		for x := 0; x < w; x++ {
			heat := fv.heatmap[cy][x]
			if heat < 0.02 {
				continue
			}

			color := heatToColor(heat, scheme)

			var ch rune
			switch {
			case heat > 0.8:
				ch = '█'
			case heat > 0.6:
				ch = '▓'
			case heat > 0.4:
				ch = '▒'
			case heat > 0.2:
				ch = '░'
			case heat > 0.1:
				tips := []rune{'·', '∗', '᛫', '⁘', '⁖'}
				ch = tips[rand.Intn(len(tips))]
			default:
				ch = '·'
			}

			st := tcell.StyleDefault.Foreground(color)
			screen.SetContent(x, cy, ch, nil, st)
		}
	}

	for i := 0; i < 10; i++ {
		x := rand.Intn(w)
		maxH := 0
		for y := 0; y < drawH; y++ {
			if fv.heatmap[y][x] > 0.3 {
				maxH = drawH - y
				break
			}
		}
		if maxH > 3 {
			emberY := drawH - maxH - rand.Intn(3) - 1
			if emberY >= 0 && emberY < drawH {
				emberColor := scheme.At(0.7 + rand.Float64()*0.3)
				st := tcell.StyleDefault.Foreground(emberColor)
				embers := []rune{'·', '∘', '°', '•', '*'}
				screen.SetContent(x, emberY, embers[rand.Intn(len(embers))], nil, st)
			}
		}
	}
}

func heatToColor(heat float64, scheme ColorScheme) tcell.Color {
	if heat < 0.01 {
		return tcell.ColorDefault
	}

	t := math.Pow(heat, 0.6)
	color := scheme.At(t)

	if heat < 0.15 {
		r, g, b := color.RGB()
		factor := heat / 0.15
		return tcell.NewRGBColor(
			int32(float64(r)*factor),
			int32(float64(g)*factor),
			int32(float64(b)*factor),
		)
	}

	return color
}
