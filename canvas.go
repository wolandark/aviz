package main

import "github.com/gdamore/tcell/v2"

type BrailleCanvas struct {
	width  int
	height int
	dots   [][]bool
	colors [][]tcell.Color
}

func NewBrailleCanvas(w, h int) *BrailleCanvas {
	ph := h * 4
	pw := w * 2
	dots := make([][]bool, ph)
	colors := make([][]tcell.Color, ph)
	for i := range dots {
		dots[i] = make([]bool, pw)
		colors[i] = make([]tcell.Color, pw)
	}
	return &BrailleCanvas{
		width:  w,
		height: h,
		dots:   dots,
		colors: colors,
	}
}

func (bc *BrailleCanvas) PixelWidth() int  { return bc.width * 2 }
func (bc *BrailleCanvas) PixelHeight() int { return bc.height * 4 }

func (bc *BrailleCanvas) Clear() {
	for y := range bc.dots {
		for x := range bc.dots[y] {
			bc.dots[y][x] = false
			bc.colors[y][x] = tcell.ColorDefault
		}
	}
}

func (bc *BrailleCanvas) Set(x, y int, color tcell.Color) {
	if x >= 0 && x < bc.PixelWidth() && y >= 0 && y < bc.PixelHeight() {
		bc.dots[y][x] = true
		bc.colors[y][x] = color
	}
}

func (bc *BrailleCanvas) DrawLine(x0, y0, x1, y1 int, color tcell.Color) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx := 1
	sy := 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		bc.Set(x0, y0, color)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func (bc *BrailleCanvas) Render(screen tcell.Screen, offsetX, offsetY int) {
	for cy := 0; cy < bc.height; cy++ {
		for cx := 0; cx < bc.width; cx++ {
			var braille rune = 0x2800
			var dominantColor tcell.Color = tcell.ColorDefault
			maxPriority := -1

			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					py := cy*4 + dy
					px := cx*2 + dx
					if py < len(bc.dots) && px < len(bc.dots[py]) && bc.dots[py][px] {
						braille |= brailleBit(dx, dy)
						priority := (3-dy)*2 + (1 - dx)
						if priority > maxPriority {
							maxPriority = priority
							dominantColor = bc.colors[py][px]
						}
					}
				}
			}

			if braille != 0x2800 {
				st := tcell.StyleDefault.Foreground(dominantColor)
				screen.SetContent(offsetX+cx, offsetY+cy, braille, nil, st)
			}
		}
	}
}

func brailleBit(x, y int) rune {
	offsets := [2][4]rune{
		{0x01, 0x02, 0x04, 0x40},
		{0x08, 0x10, 0x20, 0x80},
	}
	return offsets[x][y]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

var blockChars = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func DrawBlockColumn(screen tcell.Screen, x, bottomY, maxH int, subHeight int, scheme ColorScheme) {
	if subHeight <= 0 {
		return
	}

	fullCells := subHeight / 8
	remainder := subHeight % 8

	for i := 0; i < fullCells && i < maxH; i++ {
		y := bottomY - i
		t := float64(i) / float64(maxH)
		color := scheme.At(t)
		st := tcell.StyleDefault.Foreground(color)
		screen.SetContent(x, y, '█', nil, st)
	}

	if remainder > 0 && fullCells < maxH {
		y := bottomY - fullCells
		t := float64(fullCells) / float64(maxH)
		color := scheme.At(t)
		st := tcell.StyleDefault.Foreground(color)
		screen.SetContent(x, y, blockChars[remainder], nil, st)
	}
}
