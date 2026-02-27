package main

import (
	"math"

	"github.com/gdamore/tcell/v2"
)

type BarsVisualizer struct {
	peaks   []float64
	peakVel []float64
}

func NewBarsVisualizer() *BarsVisualizer {
	return &BarsVisualizer{}
}

func (bv *BarsVisualizer) Name() string { return "bars" }

func (bv *BarsVisualizer) Draw(screen tcell.Screen, spectrum []float64, rawSamples []float64, w, h int, scheme ColorScheme, cfg *Config) {
	barW := cfg.Visual.BarWidth
	gap := cfg.Visual.BarGap
	showPeaks := cfg.Visual.ShowPeaks
	mirror := cfg.Visual.Mirror

	drawH := h - 1
	if !cfg.Visual.ShowStatus {
		drawH = h
	}

	visH := drawH
	if mirror {
		visH = drawH / 2
	}
	if visH < 1 {
		visH = 1
	}

	numBars := (w + gap) / (barW + gap)
	if numBars < 1 {
		numBars = 1
	}

	data := resample(spectrum, numBars)

	if len(bv.peaks) != numBars {
		bv.peaks = make([]float64, numBars)
		bv.peakVel = make([]float64, numBars)
	}

	bottomY := drawH - 1
	if mirror {
		bottomY = drawH/2 + visH/2
	}

	for i := 0; i < numBars; i++ {
		val := clamp(data[i], 0, 1)
		x0 := i * (barW + gap)

		if val >= bv.peaks[i] {
			bv.peaks[i] = val
			bv.peakVel[i] = 0
		} else {
			bv.peakVel[i] += cfg.Visual.PeakFallSpeed
			bv.peaks[i] -= bv.peakVel[i]
			if bv.peaks[i] < 0 {
				bv.peaks[i] = 0
				bv.peakVel[i] = 0
			}
		}

		subHeight := int(val * float64(visH) * 8)

		for bx := 0; bx < barW; bx++ {
			cx := x0 + bx
			if cx >= w {
				break
			}

			fullCells := subHeight / 8
			remainder := subHeight % 8

			for cy := 0; cy < fullCells && cy < visH; cy++ {
				y := bottomY - cy
				t := float64(cy) / float64(visH)
				color := scheme.At(t)
				st := tcell.StyleDefault.Foreground(color)
				screen.SetContent(cx, y, '█', nil, st)
			}

			if remainder > 0 && fullCells < visH {
				y := bottomY - fullCells
				t := float64(fullCells) / float64(visH)
				color := scheme.At(t)
				st := tcell.StyleDefault.Foreground(color)
				screen.SetContent(cx, y, blockChars[remainder], nil, st)
			}

			if mirror {
				for cy := 0; cy < fullCells && cy < visH; cy++ {
					y := bottomY + 1 + cy
					if y >= drawH {
						break
					}
					t := float64(cy) / float64(visH)
					color := scheme.At(t)
					r, g, b := colorToRGB(color)
					dimColor := tcell.NewRGBColor(
						int32(float64(r)*0.3),
						int32(float64(g)*0.3),
						int32(float64(b)*0.3),
					)
					st := tcell.StyleDefault.Foreground(dimColor)
					screen.SetContent(cx, y, '█', nil, st)
				}

				if remainder > 0 && fullCells < visH {
					y := bottomY + 1 + fullCells
					if y < drawH {
						t := float64(fullCells) / float64(visH)
						color := scheme.At(t)
						r, g, b := colorToRGB(color)
						dimColor := tcell.NewRGBColor(
							int32(float64(r)*0.3),
							int32(float64(g)*0.3),
							int32(float64(b)*0.3),
						)
						st := tcell.StyleDefault.Foreground(dimColor)
						mirrorIdx := 8 - remainder
						if mirrorIdx > 0 && mirrorIdx < len(blockChars) {
							screen.SetContent(cx, y, blockChars[mirrorIdx], nil, st)
						}
					}
				}
			}

			if showPeaks && bv.peaks[i] > 0.01 {
				peakY := bottomY - int(bv.peaks[i]*float64(visH))
				if peakY >= 0 && peakY < drawH {
					peakT := bv.peaks[i]
					peakColor := scheme.At(peakT)
					pr, pg, pb := colorToRGB(peakColor)
					brightPeak := tcell.NewRGBColor(
						int32(math.Min(float64(pr)*1.5, 255)),
						int32(math.Min(float64(pg)*1.5, 255)),
						int32(math.Min(float64(pb)*1.5, 255)),
					)
					st := tcell.StyleDefault.Foreground(brightPeak)
					screen.SetContent(cx, peakY, '▔', nil, st)
				}
			}
		}
	}
}

func colorToRGB(c tcell.Color) (int32, int32, int32) {
	r, g, b := c.RGB()
	return r, g, b
}
