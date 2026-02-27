package main

import "github.com/gdamore/tcell/v2"

type Visualizer interface {
	Name() string
	Draw(screen tcell.Screen, spectrum []float64, rawSamples []float64, w, h int, scheme ColorScheme, cfg *Config)
}

var visualizerNames = []string{"bars", "wave", "spectrum", "circle", "fire"}

func GetVisualizer(name string) Visualizer {
	switch name {
	case "bars":
		return NewBarsVisualizer()
	case "wave":
		return NewWaveVisualizer()
	case "spectrum":
		return NewSpectrumVisualizer()
	case "circle":
		return NewCircleVisualizer()
	case "fire":
		return NewFireVisualizer()
	default:
		return NewBarsVisualizer()
	}
}

func NextVisualizer(current Visualizer) Visualizer {
	name := current.Name()
	for i, n := range visualizerNames {
		if n == name {
			return GetVisualizer(visualizerNames[(i+1)%len(visualizerNames)])
		}
	}
	return GetVisualizer(visualizerNames[0])
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func resample(data []float64, n int) []float64 {
	if n <= 0 {
		return nil
	}
	if len(data) == 0 {
		return make([]float64, n)
	}
	if len(data) == n {
		result := make([]float64, n)
		copy(result, data)
		return result
	}
	if n == 1 {
		return []float64{data[0]}
	}
	result := make([]float64, n)
	ratio := float64(len(data)-1) / float64(n-1)
	for i := 0; i < n; i++ {
		pos := float64(i) * ratio
		idx := int(pos)
		frac := pos - float64(idx)
		if idx >= len(data)-1 {
			result[i] = data[len(data)-1]
		} else {
			result[i] = lerp(data[idx], data[idx+1], frac)
		}
	}
	return result
}
