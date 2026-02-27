package main

import (
	"math"

	"github.com/gdamore/tcell/v2"
)

type SpectrumVisualizer struct {
	prevData []float64
}

func NewSpectrumVisualizer() *SpectrumVisualizer {
	return &SpectrumVisualizer{}
}

func (sv *SpectrumVisualizer) Name() string { return "spectrum" }

func (sv *SpectrumVisualizer) Draw(screen tcell.Screen, spectrum []float64, rawSamples []float64, w, h int, scheme ColorScheme, cfg *Config) {
	drawH := h - 1
	if !cfg.Visual.ShowStatus {
		drawH = h
	}
	if drawH < 2 {
		return
	}

	numPoints := w
	data := resample(spectrum, numPoints)

	smoothed := make([]float64, numPoints)
	copy(smoothed, data)
	for pass := 0; pass < 3; pass++ {
		for i := 1; i < numPoints-1; i++ {
			smoothed[i] = smoothed[i]*0.5 + (smoothed[i-1]+smoothed[i+1])*0.25
		}
	}

	if len(sv.prevData) != numPoints {
		sv.prevData = make([]float64, numPoints)
	}
	for i := range smoothed {
		sv.prevData[i] = sv.prevData[i]*0.3 + smoothed[i]*0.7
		smoothed[i] = sv.prevData[i]
	}

	for x := 0; x < w; x++ {
		val := clamp(smoothed[x], 0, 1)
		barH := int(val * float64(drawH-1))

		for y := 0; y < barH; y++ {
			cellY := drawH - 1 - y
			t := float64(y) / float64(drawH)
			color := scheme.At(t)

			distFromTop := float64(barH-y) / float64(barH)
			brightness := 0.3 + 0.7*math.Pow(1-distFromTop, 0.5)

			r, g, b := color.RGB()
			finalColor := tcell.NewRGBColor(
				int32(float64(r)*brightness),
				int32(float64(g)*brightness),
				int32(float64(b)*brightness),
			)

			st := tcell.StyleDefault.Foreground(finalColor)
			if y == barH-1 {
				subPos := val*float64(drawH-1) - float64(barH-1)
				if subPos > 0.5 {
					screen.SetContent(x, cellY, '▄', nil, st)
				} else {
					screen.SetContent(x, cellY, '▂', nil, st)
				}
			} else {
				screen.SetContent(x, cellY, '█', nil, st)
			}
		}
	}

	canvas := NewBrailleCanvas(w, drawH)
	pw := canvas.PixelWidth()
	ph := canvas.PixelHeight()

	curveData := resample(smoothed, pw)
	for x := 0; x < pw-1; x++ {
		y0 := ph - 1 - int(clamp(curveData[x], 0, 1)*float64(ph-1))
		y1 := ph - 1 - int(clamp(curveData[x+1], 0, 1)*float64(ph-1))

		t := float64(x) / float64(pw)
		outlineColor := scheme.At(t)
		r, g, b := outlineColor.RGB()
		brightColor := tcell.NewRGBColor(
			int32(math.Min(float64(r)*1.8+50, 255)),
			int32(math.Min(float64(g)*1.8+50, 255)),
			int32(math.Min(float64(b)*1.8+50, 255)),
		)

		canvas.DrawLine(x, y0, x+1, y1, brightColor)
		if y0 > 0 {
			canvas.Set(x, y0-1, brightColor)
		}
	}

	canvas.Render(screen, 0, 0)
}
