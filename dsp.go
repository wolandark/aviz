package main

import (
	"math"
	"math/cmplx"
)

type Processor struct {
	cfg        *Config
	window     []float64
	prevBands  []float64
	numBands   int
	sampleRate float64
}

func NewProcessor(cfg *Config) *Processor {
	bufSize := cfg.Audio.BufferSize
	window := make([]float64, bufSize)
	for i := range window {
		window[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(bufSize-1)))
	}

	return &Processor{
		cfg:        cfg,
		window:     window,
		numBands:   0,
		sampleRate: float64(cfg.Audio.SampleRate),
	}
}

func (p *Processor) Process(samples []float64, numBands int) []float64 {
	n := nextPow2(len(samples))
	if n < 64 {
		n = 64
	}

	windowed := make([]complex128, n)
	for i := 0; i < n && i < len(samples); i++ {
		w := 1.0
		if i < len(p.window) {
			w = p.window[i]
		}
		windowed[i] = complex(samples[i]*w, 0)
	}

	spectrum := fft(windowed)

	halfN := n / 2
	magnitudes := make([]float64, halfN)
	for i := 0; i < halfN; i++ {
		magnitudes[i] = cmplx.Abs(spectrum[i]) / float64(n)
	}

	if numBands <= 0 {
		numBands = 64
	}
	bands := p.groupIntoBands(magnitudes, numBands)

	for i := range bands {
		if bands[i] > 0 {
			bands[i] = math.Log10(1 + bands[i]*9)
		}
	}

	if p.prevBands == nil || len(p.prevBands) != len(bands) {
		p.prevBands = make([]float64, len(bands))
	}

	smoothing := p.cfg.Visual.Smoothing
	for i := range bands {
		if bands[i] >= p.prevBands[i] {
			p.prevBands[i] = bands[i]*0.7 + p.prevBands[i]*0.3
		} else {
			p.prevBands[i] = bands[i]*(1-smoothing) + p.prevBands[i]*smoothing
		}
	}

	result := make([]float64, len(bands))
	maxVal := 0.001
	for _, v := range p.prevBands {
		if v > maxVal {
			maxVal = v
		}
	}

	for i := range p.prevBands {
		result[i] = math.Min(p.prevBands[i]/maxVal, 1.0)
		result[i] = math.Pow(result[i], 0.7)
	}

	sens := p.cfg.Visual.Sensitivity
	for i := range result {
		result[i] = math.Min(result[i]*sens, 1.0)
	}

	return result
}

func (p *Processor) groupIntoBands(magnitudes []float64, numBands int) []float64 {
	bands := make([]float64, numBands)
	halfN := len(magnitudes)
	freqRes := p.sampleRate / float64(halfN*2)

	lowFreq := 30.0
	highFreq := math.Min(p.sampleRate/2, 18000.0)

	for i := 0; i < numBands; i++ {
		f0 := lowFreq * math.Pow(highFreq/lowFreq, float64(i)/float64(numBands))
		f1 := lowFreq * math.Pow(highFreq/lowFreq, float64(i+1)/float64(numBands))

		bin0 := int(f0 / freqRes)
		bin1 := int(f1 / freqRes)

		if bin0 >= halfN {
			bin0 = halfN - 1
		}
		if bin1 >= halfN {
			bin1 = halfN - 1
		}
		if bin1 < bin0 {
			bin1 = bin0
		}

		sum := 0.0
		count := 0
		for j := bin0; j <= bin1; j++ {
			if j < halfN {
				sum += magnitudes[j]
				count++
			}
		}
		if count > 0 {
			bands[i] = sum / float64(count)
		}

		weight := 1.0 + float64(i)/float64(numBands)*2.0
		bands[i] *= weight
	}

	return bands
}

func fft(data []complex128) []complex128 {
	n := len(data)
	if n <= 1 {
		result := make([]complex128, len(data))
		copy(result, data)
		return result
	}

	bits := 0
	for tmp := n; tmp > 1; tmp >>= 1 {
		bits++
	}

	result := make([]complex128, n)
	for i := 0; i < n; i++ {
		j := bitReverse(i, bits)
		result[j] = data[i]
	}

	for size := 2; size <= n; size *= 2 {
		halfSize := size / 2
		wBase := -2.0 * math.Pi / float64(size)

		for start := 0; start < n; start += size {
			for i := 0; i < halfSize; i++ {
				angle := wBase * float64(i)
				w := cmplx.Rect(1, angle)
				t := w * result[start+i+halfSize]
				result[start+i+halfSize] = result[start+i] - t
				result[start+i] = result[start+i] + t
			}
		}
	}

	return result
}

func bitReverse(x, bits int) int {
	result := 0
	for i := 0; i < bits; i++ {
		result = (result << 1) | (x & 1)
		x >>= 1
	}
	return result
}

func nextPow2(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}
