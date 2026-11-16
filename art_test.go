package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"
)

// TestArtSheet renders every sprite to a PNG so the art can be eyeballed.
// Set LANDFALL_SHEET=path to write it.
func TestArtSheet(t *testing.T) {
	headlessArt = true
	defer func() { headlessArt = false }()
	initArt()
	c := &Colonist{ID: 3, Col: colors[1]}
	for dir := 0; dir < 4; dir++ {
		for f := 0; f < 4; f++ {
			paintColonist(colors[dir], hairs[dir], skins[f%4], dir, f, f%2 == 1)
		}
	}
	paintLying(c.Col, hairs[2], skins[1])
	if len(headlessSprites) < 60 {
		t.Fatalf("only %d sprites", len(headlessSprites))
	}
	path := os.Getenv("LANDFALL_SHEET")
	if path == "" {
		return
	}
	cols, scale := 16, 3
	rows := (len(headlessSprites) + cols - 1) / cols
	sheet := image.NewNRGBA(image.Rect(0, 0, cols*32*scale, rows*32*scale))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.NRGBA{96, 150, 80, 255}}, image.Point{}, draw.Src)
	for i, s := range headlessSprites {
		ox, oy := (i%cols)*32*scale, (i/cols)*32*scale
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				px := s.NRGBAAt(x, y)
				if px.A == 0 {
					continue
				}
				bg := sheet.NRGBAAt(ox+x*scale, oy+y*scale)
				a := float64(px.A) / 255
				m := color.NRGBA{uint8(float64(px.R)*a + float64(bg.R)*(1-a)), uint8(float64(px.G)*a + float64(bg.G)*(1-a)), uint8(float64(px.B)*a + float64(bg.B)*(1-a)), 255}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						sheet.SetNRGBA(ox+x*scale+dx, oy+y*scale+dy, m)
					}
				}
			}
		}
	}
	f, _ := os.Create(path)
	defer f.Close()
	png.Encode(f, sheet)
}
