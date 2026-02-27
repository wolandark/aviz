package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type AudioSource interface {
	Read() []float64
	Close()
}

type PulseAudioCapture struct {
	cmd        *exec.Cmd
	reader     io.ReadCloser
	bufferSize int
	mu         sync.Mutex
	samples    []float64
	running    bool
}

func getMonitorSource() (string, error) {
	out, err := exec.Command("pactl", "get-default-sink").Output()
	if err != nil {
		return "", fmt.Errorf("cannot get default sink: %w", err)
	}
	sink := strings.TrimSpace(string(out))
	if sink == "" {
		return "", fmt.Errorf("no default sink found")
	}
	return sink + ".monitor", nil
}

func NewPulseAudioCapture(sampleRate, bufferSize int) (*PulseAudioCapture, error) {
	monitor, err := getMonitorSource()
	if err != nil {
		return nil, fmt.Errorf("failed to find monitor source: %w", err)
	}

	cmd := exec.Command("parec",
		"--format=float32le",
		fmt.Sprintf("--rate=%d", sampleRate),
		"--channels=1",
		fmt.Sprintf("--device=%s", monitor),
		"--latency-msec=25",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start parec (is pulseaudio/pipewire-pulse installed?): %w", err)
	}

	pac := &PulseAudioCapture{
		cmd:        cmd,
		reader:     stdout,
		bufferSize: bufferSize,
		samples:    make([]float64, bufferSize),
		running:    true,
	}

	go pac.readLoop()
	return pac, nil
}

func (pac *PulseAudioCapture) readLoop() {
	buf := make([]byte, pac.bufferSize*4)
	for pac.running {
		n, err := io.ReadFull(pac.reader, buf)
		if err != nil {
			if pac.running {
				time.Sleep(10 * time.Millisecond)
			}
			continue
		}

		numSamples := n / 4
		samples := make([]float64, numSamples)
		for i := 0; i < numSamples; i++ {
			bits := binary.LittleEndian.Uint32(buf[i*4 : i*4+4])
			samples[i] = float64(math.Float32frombits(bits))
		}

		pac.mu.Lock()
		pac.samples = samples
		pac.mu.Unlock()
	}
}

func (pac *PulseAudioCapture) Read() []float64 {
	pac.mu.Lock()
	defer pac.mu.Unlock()
	result := make([]float64, len(pac.samples))
	copy(result, pac.samples)
	return result
}

func (pac *PulseAudioCapture) Close() {
	pac.running = false
	if pac.cmd != nil && pac.cmd.Process != nil {
		_ = pac.cmd.Process.Kill()
		_ = pac.cmd.Wait()
	}
}

type DemoAudio struct {
	sampleRate float64
	bufferSize int
	phase      float64
	time       float64
	freqs      []demoOsc
}

type demoOsc struct {
	freq     float64
	amp      float64
	ampMod   float64
	ampModF  float64
	freqMod  float64
	freqModF float64
	phase    float64
}

func NewDemoAudio(sampleRate, bufferSize int) *DemoAudio {
	return &DemoAudio{
		sampleRate: float64(sampleRate),
		bufferSize: bufferSize,
		freqs: []demoOsc{
			{freq: 55, amp: 0.8, ampMod: 0.9, ampModF: 2.1, freqMod: 10, freqModF: 2.1},
			{freq: 80, amp: 0.6, ampMod: 0.8, ampModF: 1.05},
			{freq: 150, amp: 0.4, ampMod: 0.7, ampModF: 3.3},
			{freq: 220, amp: 0.35, ampMod: 0.6, ampModF: 1.7},
			{freq: 440, amp: 0.3, ampMod: 0.8, ampModF: 0.8},
			{freq: 554, amp: 0.25, ampMod: 0.7, ampModF: 1.2},
			{freq: 660, amp: 0.25, ampMod: 0.75, ampModF: 0.6},
			{freq: 880, amp: 0.2, ampMod: 0.6, ampModF: 1.5},
			{freq: 1200, amp: 0.15, ampMod: 0.5, ampModF: 2.5},
			{freq: 1800, amp: 0.1, ampMod: 0.6, ampModF: 3.0},
			{freq: 2400, amp: 0.08, ampMod: 0.5, ampModF: 1.8},
			{freq: 3600, amp: 0.06, ampMod: 0.4, ampModF: 2.2},
			{freq: 5000, amp: 0.04, ampMod: 0.3, ampModF: 4.0},
			{freq: 8000, amp: 0.03, ampMod: 0.4, ampModF: 5.5},
			{freq: 12000, amp: 0.02, ampMod: 0.3, ampModF: 3.5},
		},
	}
}

func (da *DemoAudio) Read() []float64 {
	samples := make([]float64, da.bufferSize)
	dt := 1.0 / da.sampleRate

	for i := range samples {
		t := da.time + float64(i)*dt
		sample := 0.0

		for j := range da.freqs {
			osc := &da.freqs[j]
			amp := osc.amp * (1 - osc.ampMod + osc.ampMod*math.Abs(math.Sin(2*math.Pi*osc.ampModF*t)))
			freq := osc.freq + osc.freqMod*math.Sin(2*math.Pi*osc.freqModF*t)
			sample += amp * math.Sin(2*math.Pi*freq*t+osc.phase)
		}

		sample += (rand.Float64()*2 - 1) * 0.01
		samples[i] = sample * 0.3
	}

	da.time += float64(da.bufferSize) * dt

	return samples
}

func (da *DemoAudio) Close() {}
