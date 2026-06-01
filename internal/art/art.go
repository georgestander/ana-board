package art

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

const PixelSymbol = "█"

type paletteColor struct {
	name    string
	r, g, b float64
}

var blockPalette = []paletteColor{
	{name: "white", r: 243, g: 234, b: 208},
	{name: "green", r: 148, g: 213, b: 155},
	{name: "amber", r: 228, g: 191, b: 114},
	{name: "red", r: 233, g: 139, b: 125},
	{name: "blue", r: 143, g: 185, b: 232},
	{name: "violet", r: 199, g: 165, b: 236},
}

var spritePatterns = map[string][]string{
	"check": {
		"......G",
		".....GG",
		"G...GG.",
		"GG.GG..",
		".GGG...",
		"..G....",
	},
	"x": {
		"RR...RR",
		".RR.RR.",
		"..RRR..",
		"..RRR..",
		".RR.RR.",
		"RR...RR",
	},
	"warning": {
		"...A...",
		"..AAA..",
		".AAAAA.",
		"AAA.AAA",
		"AAAAAAA",
		"...A...",
	},
	"trophy": {
		"..AAAAA..",
		".AAAAAAA.",
		".A.AAA.A.",
		"..AAAAA..",
		"...AAA...",
		"..AAAAA..",
	},
	"rocket": {
		"....W....",
		"...WWW...",
		"..WWBWW..",
		"..WWBWW..",
		".RWWBWWR.",
		"RRR...RRR",
	},
	"heart": {
		"..RR.RR..",
		".RRRRRRR.",
		"RRRRRRRRR",
		".RRRRRRR.",
		"..RRRRR..",
		"...RRR...",
	},
	"face": {
		".AAAAA.",
		"AAAAAAA",
		"AA.A.AA",
		"AAAAAAA",
		"AA...AA",
		".AAAAA.",
	},
	"wrench": {
		"BB....B",
		".BB..B.",
		"..BBBB.",
		"...BB..",
		"..BBBB.",
		".BB..BB",
	},
}

var patternColors = map[rune]string{
	'W': "white",
	'G': "green",
	'A': "amber",
	'R': "red",
	'B': "blue",
	'V': "violet",
}

func ListSprites() []string {
	names := make([]string, 0, len(spritePatterns))
	for name := range spritePatterns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func SpriteFrame(name string) (messages.FrameInput, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	pattern, ok := spritePatterns[name]
	if !ok {
		return messages.FrameInput{}, fmt.Errorf("unknown sprite %q", name)
	}

	return frameFromPattern(pattern)
}

func ImageFrame(reader io.Reader) (messages.FrameInput, error) {
	img, _, err := image.Decode(reader)
	if err != nil {
		return messages.FrameInput{}, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Empty() {
		return messages.FrameInput{}, fmt.Errorf("image is empty")
	}

	frame := blankFrame()
	for row := 0; row < board.DefaultRows; row++ {
		for col := 0; col < board.DefaultCols; col++ {
			r, g, b, alpha, ok := sampleCell(img, bounds, row, col)
			if !ok || alpha < 64 {
				continue
			}

			frame.Cells[row][col] = PixelSymbol
			frame.Colors[row][col] = nearestPaletteColor(r, g, b)
		}
	}

	return frame, nil
}

func frameFromPattern(pattern []string) (messages.FrameInput, error) {
	if len(pattern) == 0 {
		return messages.FrameInput{}, fmt.Errorf("sprite pattern is empty")
	}
	if len(pattern) > board.DefaultRows {
		return messages.FrameInput{}, fmt.Errorf("sprite pattern has %d rows, max is %d", len(pattern), board.DefaultRows)
	}

	width := 0
	for _, row := range pattern {
		if len(row) > width {
			width = len(row)
		}
	}
	if width > board.DefaultCols {
		return messages.FrameInput{}, fmt.Errorf("sprite pattern has %d columns, max is %d", width, board.DefaultCols)
	}

	frame := blankFrame()
	rowOffset := (board.DefaultRows - len(pattern)) / 2
	colOffset := (board.DefaultCols - width) / 2

	for row, line := range pattern {
		for col, marker := range line {
			colorName, ok := patternColors[marker]
			if !ok {
				continue
			}
			frame.Cells[rowOffset+row][colOffset+col] = PixelSymbol
			frame.Colors[rowOffset+row][colOffset+col] = colorName
		}
	}

	return frame, nil
}

func blankFrame() messages.FrameInput {
	cells := make([][]string, board.DefaultRows)
	colors := make([][]string, board.DefaultRows)
	for row := 0; row < board.DefaultRows; row++ {
		cells[row] = make([]string, board.DefaultCols)
		colors[row] = make([]string, board.DefaultCols)
		for col := 0; col < board.DefaultCols; col++ {
			cells[row][col] = " "
			colors[row][col] = board.DefaultColor
		}
	}

	return messages.FrameInput{Cells: cells, Colors: colors}
}

func sampleCell(img image.Image, bounds image.Rectangle, row, col int) (float64, float64, float64, float64, bool) {
	x0 := bounds.Min.X + col*bounds.Dx()/board.DefaultCols
	x1 := bounds.Min.X + (col+1)*bounds.Dx()/board.DefaultCols
	y0 := bounds.Min.Y + row*bounds.Dy()/board.DefaultRows
	y1 := bounds.Min.Y + (row+1)*bounds.Dy()/board.DefaultRows
	if x1 <= x0 {
		x1 = x0 + 1
	}
	if y1 <= y0 {
		y1 = y0 + 1
	}
	if x1 > bounds.Max.X {
		x1 = bounds.Max.X
	}
	if y1 > bounds.Max.Y {
		y1 = bounds.Max.Y
	}

	var redSum, greenSum, blueSum, alphaSum float64
	var samples float64
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			pixel := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			samples++
			alpha := float64(pixel.A)
			alphaSum += alpha
			redSum += float64(pixel.R) * alpha
			greenSum += float64(pixel.G) * alpha
			blueSum += float64(pixel.B) * alpha
		}
	}

	if samples == 0 || alphaSum == 0 {
		return 0, 0, 0, 0, false
	}

	return redSum / alphaSum, greenSum / alphaSum, blueSum / alphaSum, alphaSum / samples, true
}

func nearestPaletteColor(red, green, blue float64) string {
	best := blockPalette[0]
	bestDistance := math.MaxFloat64
	for _, candidate := range blockPalette {
		distance := squaredDistance(red, green, blue, candidate.r, candidate.g, candidate.b)
		if distance < bestDistance {
			best = candidate
			bestDistance = distance
		}
	}

	return best.name
}

func squaredDistance(r1, g1, b1, r2, g2, b2 float64) float64 {
	red := r1 - r2
	green := g1 - g2
	blue := b1 - b2
	return red*red + green*green + blue*blue
}
