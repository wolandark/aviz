# aviz

terminal audio visualizer. captures system audio and draws stuff.

## install

need go and pulseaudio/pipewire-pulse (parec).

```
go build -o aviz .
```

## usage

```
./aviz              # captures system audio
./aviz --demo       # fake audio, no setup needed
./aviz --style fire --colors neon
```

## keys

```
1-5       visualization style (bars, wave, spectrum, circle, fire)
n         next style
c / C     cycle color scheme
+ / -     sensitivity
m         mirror
p         peaks
s         smoothing
[ / ]     bar width
?         help
q / esc   quit
```

## config

optional. put it at `~/.config/aviz/config.yaml` or don't.

```yaml
style: bars
color_scheme: rainbow
audio:
  sample_rate: 44100
  buffer_size: 4096
visual:
  fps: 60
  bar_width: 2
  bar_gap: 1
  smoothing: 0.65
  sensitivity: 1.0
  peak_fall_speed: 0.03
  show_peaks: true
  mirror: false
  show_status: true
```

cli flags override config.

## flags

```
--style        bars|wave|spectrum|circle|fire
--colors       rainbow|fire|ocean|neon|pastel|matrix|sunset|aurora
--sensitivity  float
--fps          int
--demo         no audio needed
--config       path to config file
--list         show available styles/schemes
```
