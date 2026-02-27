package main

import (
	"math"

	"github.com/gdamore/tcell/v2"
)

type CircleVisualizer struct {
	rotation float64
	peaks    []float64
}

func NewCircleVisualizer() *CircleVisualizer {
	return &CircleVisualizer{}
}

func (cv *CircleVisualizer) Name() string { return "circle" }

func (cv *CircleVisualizer) Draw(screen tcell.Screen, spectrum []float64, rawSamples []float64, w, h int, scheme ColorScheme, cfg *Config) {
	drawH := h - 1
	if !cfg.Visual.ShowStatus {
		drawH = h
	}
	if drawH < 4 || w < 4 {
		return
	}

	canvas := NewBrailleCanvas(w, drawH)
	pw := canvas.PixelWidth()
	ph := canvas.PixelHeight()

	centerX := pw / 2
	centerY := ph / 2

	aspectX := 1.0
	aspectY := 2.0

	maxRadius := math.Min(float64(pw)/2*0.85/aspectX, float64(ph)/2*0.85/aspectY)
	innerRadius := maxRadius * 0.25

	numBars := 128
	if len(spectrum) < numBars {
		numBars = len(spectrum)
	}
	if numBars < 16 {
		numBars = 16
	}

	data := resample(spectrum, numBars)

	if len(cv.peaks) != numBars {
		cv.peaks = make([]float64, numBars)
	}

	for i := range data {
		if data[i] > cv.peaks[i] {
			cv.peaks[i] = data[i]
		} else {
			cv.peaks[i] *= 0.97
		}
	}

	for i := 0; i < numBars; i++ {
		angle := cv.rotation + float64(i)*2*math.Pi/float64(numBars)
		val := clamp(data[i], 0, 1)

		barLength := val * (maxRadius - innerRadius)
		if barLength < 1 {
			barLength = 1
		}

		t := float64(i) / float64(numBars)

		steps := int(barLength)
		for s := 0; s <= steps; s++ {
			r := innerRadius + float64(s)
			x := centerX + int(math.Cos(angle)*r*aspectX)
			y := centerY + int(math.Sin(angle)*r*aspectY)

			dt := float64(s) / (maxRadius - innerRadius)
			radColor := scheme.At(t + dt*0.3)
			rr, gg, bb := radColor.RGB()
			brightness := 0.5 + 0.5*dt
			finalColor := tcell.NewRGBColor(
				int32(math.Min(float64(rr)*brightness, 255)),
				int32(math.Min(float64(gg)*brightness, 255)),
				int32(math.Min(float64(bb)*brightness, 255)),
			)

			canvas.Set(x, y, finalColor)
		}

		if cv.peaks[i] > 0.05 {
			peakR := innerRadius + cv.peaks[i]*(maxRadius-innerRadius)
			px := centerX + int(math.Cos(angle)*peakR*aspectX)
			py := centerY + int(math.Sin(angle)*peakR*aspectY)
			peakColor := scheme.At(t)
			pr, pg, pb := peakColor.RGB()
			brightPeak := tcell.NewRGBColor(
				int32(math.Min(float64(pr)*1.5+80, 255)),
				int32(math.Min(float64(pg)*1.5+80, 255)),
				int32(math.Min(float64(pb)*1.5+80, 255)),
			)
			canvas.Set(px, py, brightPeak)
			if px+1 < pw {
				canvas.Set(px+1, py, brightPeak)
			}
			if py+1 < ph {
				canvas.Set(px, py+1, brightPeak)
			}
		}
	}

	innerSteps := int(innerRadius * 2 * math.Pi)
	for i := 0; i < innerSteps; i++ {
		angle := float64(i) * 2 * math.Pi / float64(innerSteps)
		x := centerX + int(math.Cos(angle)*innerRadius*aspectX)
		y := centerY + int(math.Sin(angle)*innerRadius*aspectY)
		t := float64(i) / float64(innerSteps)
		canvas.Set(x, y, scheme.At(t))
	}

	canvas.Render(screen, 0, 0)

	cv.rotation += 0.005

	energy := 0.0
	for _, v := range data {
		energy += v
	}
	energy /= float64(len(data))

	glowRadius := int(energy * 3)
	cx := w / 2
	cy := drawH / 2
	for dy := -glowRadius; dy <= glowRadius; dy++ {
		for dx := -glowRadius * 2; dx <= glowRadius*2; dx++ {
			dist := math.Sqrt(float64(dx*dx)/4 + float64(dy*dy))
			if dist <= float64(glowRadius) {
				sx := cx + dx
				sy := cy + dy
				if sx >= 0 && sx < w && sy >= 0 && sy < drawH {
					t := 1 - dist/float64(glowRadius+1)
					color := scheme.At(0.5)
					r, g, b := color.RGB()
					gc := tcell.NewRGBColor(
						int32(float64(r)*t*0.6),
						int32(float64(g)*t*0.6),
						int32(float64(b)*t*0.6),
					)
					ch, _, _, _ := screen.GetContent(sx, sy)
					if ch == 0 || ch == ' ' {
						var glowChar rune
						switch {
						case t > 0.7:
							glowChar = '░'
						case t > 0.4:
							glowChar = '·'
						default:
							glowChar = '·'
						}
						st := tcell.StyleDefault.Foreground(gc)
						screen.SetContent(sx, sy, glowChar, nil, st)
					}
				}
			}
		}
	}
}
