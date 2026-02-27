package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

func main() {
	configFile := flag.String("config", "", "Path to config file (default: ~/.config/audiovis/config.yaml)")
	style := flag.String("style", "", "Visualization style: bars, wave, spectrum, circle, fire")
	colorScheme := flag.String("colors", "", "Color scheme: rainbow, fire, ocean, neon, pastel, matrix, sunset, aurora")
	sensitivity := flag.Float64("sensitivity", 0, "Audio sensitivity multiplier (default: 1.0)")
	demo := flag.Bool("demo", false, "Demo mode with synthetic audio (no audio input needed)")
	listStyles := flag.Bool("list", false, "List available styles and color schemes")
	fps := flag.Int("fps", 0, "Target frames per second (default: 60)")
	flag.Parse()

	if *listStyles {
		fmt.Println("╔══════════════════════════════════════════╗")
		fmt.Println("║          AUDIOVIS - Terminal Audio       ║")
		fmt.Println("║              Visualizer                  ║")
		fmt.Println("╠══════════════════════════════════════════╣")
		fmt.Println("║  Visualization Styles:                   ║")
		fmt.Println("║    bars     - Classic frequency bars     ║")
		fmt.Println("║    wave     - Oscilloscope waveform      ║")
		fmt.Println("║    spectrum - Smooth spectrum curve      ║")
		fmt.Println("║    circle   - Radial visualizer          ║")
		fmt.Println("║    fire     - Flame effect               ║")
		fmt.Println("║                                          ║")
		fmt.Println("║  Color Schemes:                          ║")
		for _, name := range AllSchemeNames() {
			fmt.Printf("║    %-38s║\n", name)
		}
		fmt.Println("╚══════════════════════════════════════════╝")
		return
	}

	cfg := DefaultConfig()
	cfg.TryLoadDefault()

	if *configFile != "" {
		if err := cfg.LoadFromFile(*configFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	if *style != "" {
		cfg.Style = *style
	}
	if *colorScheme != "" {
		cfg.ColorScheme = *colorScheme
	}
	if *sensitivity > 0 {
		cfg.Visual.Sensitivity = *sensitivity
	}
	if *fps > 0 {
		cfg.Visual.FPS = *fps
	}
	cfg.DemoMode = *demo

	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating screen: %v\n", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	screen.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite))
	screen.EnableMouse()
	screen.Clear()

	var audio AudioSource
	audioErr := ""

	if cfg.DemoMode {
		audio = NewDemoAudio(cfg.Audio.SampleRate, cfg.Audio.BufferSize)
	} else {
		pa, err := NewPulseAudioCapture(cfg.Audio.SampleRate, cfg.Audio.BufferSize)
		if err != nil {
			audioErr = err.Error()
			audio = NewDemoAudio(cfg.Audio.SampleRate, cfg.Audio.BufferSize)
			cfg.DemoMode = true
		} else {
			audio = pa
		}
	}
	defer audio.Close()

	processor := NewProcessor(cfg)

	vis := GetVisualizer(cfg.Style)
	colors := GetColorScheme(cfg.ColorScheme)

	ticker := time.NewTicker(time.Second / time.Duration(cfg.Visual.FPS))
	defer ticker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	eventCh := make(chan tcell.Event, 32)
	quitEventLoop := make(chan struct{})
	go func() {
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case eventCh <- ev:
			case <-quitEventLoop:
				return
			}
		}
	}()

	showHelp := false
	showAudioErr := audioErr != ""
	audioErrTimeout := time.Now().Add(5 * time.Second)
	running := true
	frameCount := 0

	for running {
		select {
		case <-sigCh:
			running = false

		case ev := <-eventCh:
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape:
					if showHelp {
						showHelp = false
					} else {
						running = false
					}
				case tcell.KeyCtrlC:
					running = false
				case tcell.KeyRune:
					switch ev.Rune() {
					case 'q', 'Q':
						running = false
					case '1':
						vis = GetVisualizer("bars")
					case '2':
						vis = GetVisualizer("wave")
					case '3':
						vis = GetVisualizer("spectrum")
					case '4':
						vis = GetVisualizer("circle")
					case '5':
						vis = GetVisualizer("fire")
					case 'n', 'N':
						vis = NextVisualizer(vis)
					case 'c':
						colors = NextColorScheme(colors)
					case 'C':
						colors = PrevColorScheme(colors)
					case '+', '=':
						cfg.Visual.Sensitivity *= 1.15
						if cfg.Visual.Sensitivity > 5 {
							cfg.Visual.Sensitivity = 5
						}
					case '-', '_':
						cfg.Visual.Sensitivity /= 1.15
						if cfg.Visual.Sensitivity < 0.01 {
							cfg.Visual.Sensitivity = 0.01
						}
					case 'm', 'M':
						cfg.Visual.Mirror = !cfg.Visual.Mirror
					case 'p', 'P':
						cfg.Visual.ShowPeaks = !cfg.Visual.ShowPeaks
					case 's', 'S':
						cfg.Visual.Smoothing += 0.1
						if cfg.Visual.Smoothing > 0.95 {
							cfg.Visual.Smoothing = 0.1
						}
					case '?', 'h', 'H':
						showHelp = !showHelp
					case '[':
						cfg.Visual.BarWidth--
						if cfg.Visual.BarWidth < 1 {
							cfg.Visual.BarWidth = 1
						}
					case ']':
						cfg.Visual.BarWidth++
						if cfg.Visual.BarWidth > 10 {
							cfg.Visual.BarWidth = 10
						}
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}

		case <-ticker.C:
			samples := audio.Read()

			w, h := screen.Size()
			if w < 2 || h < 2 {
				continue
			}

			numBands := w
			if vis.Name() == "bars" {
				numBands = (w + cfg.Visual.BarGap) / (cfg.Visual.BarWidth + cfg.Visual.BarGap)
			} else if vis.Name() == "circle" {
				numBands = 128
			}
			if numBands < 16 {
				numBands = 16
			}

			spectrum := processor.Process(samples, numBands)

			screen.Clear()
			vis.Draw(screen, spectrum, samples, w, h, colors, cfg)

			if cfg.Visual.ShowStatus {
				drawStatusBar(screen, w, h, vis.Name(), colors.Name, cfg)
			}

			if showHelp {
				drawHelpOverlay(screen, w, h)
			}

			if showAudioErr && time.Now().Before(audioErrTimeout) {
				drawNotification(screen, w, h, "Audio: "+audioErr+" (using demo mode)", tcell.ColorYellow)
			} else {
				showAudioErr = false
			}

			screen.Show()
			frameCount++
		}
	}

	close(quitEventLoop)
}

func drawStatusBar(screen tcell.Screen, w, h int, styleName, colorName string, cfg *Config) {
	y := h - 1

	barStyle := tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(120, 120, 140))

	mode := "♪ LIVE"
	if cfg.DemoMode {
		mode = "♪ DEMO"
	}

	mirror := ""
	if cfg.Visual.Mirror {
		mirror = " │ mirror"
	}

	peaks := ""
	if cfg.Visual.ShowPeaks {
		peaks = " │ peaks"
	}

	status := fmt.Sprintf(" %s │ %s │ %s │ sens:%.1fx │ smooth:%.0f%%%s%s │ ?:help ",
		mode,
		strings.ToUpper(styleName),
		colorName,
		cfg.Visual.Sensitivity,
		cfg.Visual.Smoothing*100,
		mirror,
		peaks,
	)

	accentStyle := barStyle.Foreground(tcell.NewRGBColor(100, 200, 255))
	dimStyle := barStyle.Foreground(tcell.NewRGBColor(80, 80, 100))

	x := 0
	for _, ch := range status {
		if x >= w {
			break
		}
		s := barStyle
		if ch == '│' {
			s = dimStyle
		} else if ch == '♪' {
			s = accentStyle
		}
		screen.SetContent(x, y, ch, nil, s)
		x++
	}
}

func drawHelpOverlay(screen tcell.Screen, w, h int) {
	lines := []string{
		"╔══════════════════════════════════════════════╗",
		"║           AUDIOVIS  ─  CONTROLS              ║",
		"╠══════════════════════════════════════════════╣",
		"║                                              ║",
		"║   1-5     Switch visualization style         ║",
		"║   n       Next visualization                 ║",
		"║   c / C   Next / Previous color scheme       ║",
		"║   + / -   Adjust sensitivity                 ║",
		"║   m       Toggle mirror mode                 ║",
		"║   p       Toggle peak indicators             ║",
		"║   s       Cycle smoothing level              ║",
		"║   [ / ]   Adjust bar width                   ║",
		"║                                              ║",
		"║   ?/h     Toggle this help                   ║",
		"║   q/ESC   Quit                               ║",
		"║                                              ║",
		"║   Styles: bars wave spectrum circle fire     ║",
		"║                                              ║",
		"╚══════════════════════════════════════════════╝",
	}

	boxW := 48
	boxH := len(lines)
	startX := (w - boxW) / 2
	startY := (h - boxH) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	bgStyle := tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(200, 200, 220))

	borderStyle := tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(80, 140, 220))

	titleStyle := tcell.StyleDefault.
		Foreground(tcell.NewRGBColor(120, 200, 255)).
		Bold(true)

	for i, line := range lines {
		y := startY + i
		if y >= h {
			break
		}
		x := startX
		for _, ch := range line {
			if x >= w {
				break
			}
			s := bgStyle
			if ch == '╔' || ch == '╗' || ch == '╚' || ch == '╝' || ch == '═' || ch == '║' || ch == '╠' || ch == '╣' {
				s = borderStyle
			}
			if i == 1 {
				s = titleStyle
			}
			screen.SetContent(x, y, ch, nil, s)
			x++
		}
	}
}

func drawNotification(screen tcell.Screen, w, h int, msg string, color tcell.Color) {
	y := 1
	x := (w - len(msg) - 4) / 2
	if x < 0 {
		x = 0
	}

	style := tcell.StyleDefault.
		Foreground(color)

	text := "  " + msg + "  "
	for i, ch := range text {
		if x+i < w {
			screen.SetContent(x+i, y, ch, nil, style)
		}
	}
}
