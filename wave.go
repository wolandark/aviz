package main

import (
	"math"

	"github.com/gdamore/tcell/v2"
)

type WaveVisualizer struct {
	phase float64
}

func NewWaveVisualizer() *WaveVisualizer {
	return &WaveVisualizer{}
}

func (wv *WaveVisualizer) Name() string { return "wave" }

func (wv *WaveVisualizer) Draw(screen tcell.Screen, spectrum []float64, rawSamples []float64, w, h int, scheme ColorScheme, cfg *Config) {
	drawH := h - 1
	if !cfg.Visual.ShowStatus {
		drawH = h
	}

	canvas := NewBrailleCanvas(w, drawH)

	pw := canvas.PixelWidth()
	ph := canvas.PixelHeight()
	centerY := ph / 2

	wave := resample(rawSamples, pw)

	for i := range wave {
		wave[i] *= cfg.Visual.Sensitivity * 3.0
	}

	layers := 3
	for layer := layers - 1; layer >= 0; layer-- {
		spread := float64(layer) * 1.5
		alpha := 1.0 - float64(layer)*0.3

		for x := 0; x < pw-1; x++ {
			t := float64(x) / float64(pw)
			color := scheme.At(t)

			if layer > 0 {
				r, g, b := color.RGB()
				color = tcell.NewRGBColor(
					int32(float64(r)*alpha),
					int32(float64(g)*alpha),
					int32(float64(b)*alpha),
				)
			}

			y0 := centerY - int(wave[x]*float64(ph/3)+spread)
			y1 := centerY - int(wave[x+1]*float64(ph/3)+spread)

			canvas.DrawLine(x, clampInt(y0, 0, ph-1), x+1, clampInt(y1, 0, ph-1), color)

			if layer == 0 {
				y0b := centerY + int(wave[x]*float64(ph/3))
				y1b := centerY + int(wave[x+1]*float64(ph/3))
				dimColor := dimmedColor(color, 0.4)
				canvas.DrawLine(x, clampInt(y0b, 0, ph-1), x+1, clampInt(y1b, 0, ph-1), dimColor)
			}
		}
	}

	centerCellY := drawH / 2
	dimStyle := tcell.StyleDefault.Foreground(scheme.At(0.5)).Dim(true)
	for x := 0; x < w; x++ {
		screen.SetContent(x, centerCellY, '·', nil, dimStyle)
	}

	numBands := w
	bgData := resample(spectrum, numBands)
	for x := 0; x < w; x++ {
		val := bgData[x] * 0.3
		if val > 0.02 {
			bgH := int(val * float64(drawH))
			for y := drawH - 1; y >= drawH-bgH && y >= 0; y-- {
				t := float64(drawH-y) / float64(drawH)
				color := scheme.At(t)
				r, g, b := color.RGB()
				dimFg := tcell.NewRGBColor(
					int32(float64(r)*0.15),
					int32(float64(g)*0.15),
					int32(float64(b)*0.15),
				)
				st := tcell.StyleDefault.Foreground(dimFg)
				screen.SetContent(x, y, '░', nil, st)
			}
		}
	}

	canvas.Render(screen, 0, 0)

	wv.phase += 0.02
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func dimmedColor(c tcell.Color, factor float64) tcell.Color {
	r, g, b := c.RGB()
	return tcell.NewRGBColor(
		int32(math.Max(float64(r)*factor, 0)),
		int32(math.Max(float64(g)*factor, 0)),
		int32(math.Max(float64(b)*factor, 0)),
	)
}
