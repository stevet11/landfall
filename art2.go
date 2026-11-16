package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Sprites for crafting, electricity, trading and the new goods.

func paintBench() *ebiten.Image {
	c := newCanvas()
	c.rect(2, 7, 28, 18, rgb(112, 76, 44))
	c.rect(3, 8, 26, 16, rgb(172, 126, 78))
	for x := 3; x < 29; x += 6 {
		c.rect(x, 8, 1, 16, rgb(140, 100, 60))
	}
	c.rect(3, 8, 26, 1, rgb(196, 152, 100))
	// saw, hammer and a plank on top
	c.rect(6, 11, 11, 2, rgb(170, 176, 186))
	c.rect(6, 13, 2, 3, rgb(96, 64, 38))
	c.rect(20, 10, 3, 9, rgb(96, 64, 38))
	c.rect(18, 10, 7, 3, rgb(120, 124, 134))
	c.rect(10, 17, 9, 4, rgb(214, 180, 128))
	c.outline(dark)
	c.shadow(16, 27, 14, 3, 55)
	return c.image()
}

func paintFab() *ebiten.Image {
	c := newCanvas()
	c.rect(2, 4, 28, 24, rgb(70, 78, 92))
	c.rect(4, 6, 24, 20, rgb(96, 108, 126))
	c.rect(6, 8, 14, 9, rgb(20, 40, 44)) // screen
	c.rect(7, 9, 12, 1, rgb(80, 230, 170))
	c.rect(7, 12, 8, 1, rgb(60, 180, 140))
	c.rect(7, 14, 10, 1, rgb(60, 180, 140))
	c.disc(24, 12, 3, rgb(60, 66, 80))
	c.disc(24, 12, 1.4, rgb(210, 150, 50))
	for x := 7; x < 28; x += 4 {
		c.rect(x, 21, 2, 3, rgb(60, 66, 80))
	}
	c.rect(21, 18, 5, 2, rgb(230, 70, 60))
	c.outline(dark)
	c.shadow(16, 28, 14, 2.5, 55)
	return c.image()
}

func paintGen(frame int) *ebiten.Image {
	c := newCanvas()
	c.rect(3, 8, 26, 20, rgb(86, 90, 76))
	c.rect(5, 10, 22, 16, rgb(120, 126, 104))
	c.rect(5, 10, 22, 2, rgb(150, 156, 130))
	c.rect(21, 2, 5, 8, rgb(70, 70, 76)) // chimney
	c.rect(20, 1, 7, 2, rgb(50, 50, 56))
	c.rect(7, 14, 10, 8, rgb(40, 44, 40))
	if frame > 0 {
		c.ellipse(12, 19, 3.5, 3+float64(frame), rgb(250, 140, 40))
		c.ellipse(12, 20, 2, 2+float64(frame)*0.6, rgb(255, 220, 120))
		c.disc(23, 0, 2, col{200, 200, 200, 120})
	} else {
		c.rect(8, 15, 8, 6, rgb(30, 32, 30))
	}
	c.rect(19, 15, 6, 8, rgb(200, 170, 50))
	c.rect(20, 17, 4, 1, rgb(60, 50, 10))
	c.rect(20, 20, 4, 1, rgb(60, 50, 10))
	c.outline(dark)
	c.shadow(16, 29, 14, 2.5, 55)
	return c.image()
}

func paintSolar() *ebiten.Image {
	c := newCanvas()
	c.rect(2, 4, 28, 24, rgb(60, 66, 80))
	for gy := 0; gy < 3; gy++ {
		for gx := 0; gx < 3; gx++ {
			x, y := 4+gx*8, 6+gy*7
			c.rect(x, y, 7, 6, rgb(38, 74, 150))
			c.rect(x, y, 7, 1, rgb(110, 160, 235))
			c.set(x+1, y+2, rgb(70, 120, 210))
			c.set(x+4, y+3, rgb(70, 120, 210))
		}
	}
	c.outline(dark)
	c.shadow(16, 28, 14, 2.5, 50)
	return c.image()
}

func paintLamp(on bool) *ebiten.Image {
	c := newCanvas()
	c.disc(16, 17, 5, rgb(70, 70, 78))
	c.disc(16, 17, 3.2, rgb(40, 40, 46))
	if on {
		c.disc(16, 16, 4.5, rgb(255, 236, 140))
		c.disc(16, 16, 2.4, rgb(255, 255, 220))
	} else {
		c.disc(16, 16, 3.5, rgb(120, 116, 90))
	}
	c.outline(dark)
	c.shadow(17, 24, 7, 2.5, 55)
	return c.image()
}

func paintEHeater(on bool) *ebiten.Image {
	c := newCanvas()
	c.rect(3, 6, 26, 20, rgb(70, 74, 84))
	c.rect(5, 8, 22, 16, rgb(44, 46, 54))
	for i := 0; i < 4; i++ {
		y := 10 + i*4
		k := rgb(90, 60, 50)
		if on {
			k = rgb(255, 120+uint8(i*20), 50)
		}
		c.rect(7, y, 18, 2, k)
	}
	c.rect(24, 3, 4, 3, rgb(90, 200, 230))
	c.outline(dark)
	c.shadow(16, 27, 13, 2.5, 55)
	return c.image()
}

func paintDesk() *ebiten.Image {
	c := newCanvas()
	c.rect(3, 6, 26, 20, rgb(104, 72, 44))
	c.rect(4, 7, 24, 18, rgb(150, 108, 66))
	c.rect(4, 7, 24, 1, rgb(176, 132, 86))
	// open book and a pencil
	c.rect(8, 10, 8, 9, rgb(240, 236, 220))
	c.rect(16, 10, 8, 9, rgb(250, 246, 232))
	c.rect(15, 10, 1, 9, rgb(170, 160, 140))
	for y := 12; y < 18; y += 2 {
		c.rect(9, y, 6, 1, rgb(150, 150, 160))
		c.rect(17, y, 6, 1, rgb(150, 150, 160))
	}
	c.rect(24, 12, 1, 8, rgb(230, 190, 60))
	c.outline(dark)
	c.shadow(16, 27, 13, 2.5, 55)
	return c.image()
}

func paintGoods(it Item) *ebiten.Image {
	c := newCanvas()
	switch it {
	case ISilver:
		for _, p := range [][3]float64{{11, 21, 6}, {21, 20, 6}, {16, 14, 6.5}} {
			c.ellipse(p[0], p[1], p[2], p[2]*0.6, rgb(168, 172, 184))
			c.ellipse(p[0], p[1]-1, p[2], p[2]*0.6, rgb(226, 230, 240))
			c.ellipse(p[0], p[1]-1, p[2]*0.55, p[2]*0.33, rgb(190, 194, 206))
		}
	case ICarving:
		c.ellipse(16, 20, 6, 6, rgb(176, 122, 70)) // a little carved bird
		c.disc(21, 13, 3.5, rgb(190, 136, 82))
		c.rect(24, 13, 4, 2, rgb(220, 170, 80))
		c.set(22, 12, dark)
		c.ellipse(11, 22, 4, 2, rgb(150, 100, 56))
		c.rect(12, 25, 8, 2, rgb(120, 80, 46))
	case ISculpture:
		c.rect(10, 22, 12, 5, rgb(120, 122, 132)) // plinth
		c.ellipse(16, 17, 5, 6, rgb(190, 192, 202))
		c.disc(16, 9, 4.5, rgb(204, 206, 216))
		c.set(14, 8, dark)
		c.set(18, 8, dark)
		c.rect(14, 12, 4, 1, rgb(150, 152, 162))
	case IComponent:
		c.rect(5, 8, 22, 17, rgb(40, 130, 84))
		c.rect(5, 8, 22, 1, rgb(80, 180, 120))
		c.rect(8, 11, 7, 6, rgb(40, 40, 48))
		c.rect(18, 12, 6, 4, rgb(40, 40, 48))
		for i := 0; i < 4; i++ {
			c.set(8+i*2, 18, rgb(210, 190, 90))
			c.set(8+i*2, 17, rgb(210, 190, 90))
		}
		c.line(16, 21, 24, 21, rgb(210, 190, 90))
		c.disc(24, 20, 1.5, rgb(230, 70, 60))
	case IGadget:
		c.rect(6, 6, 20, 21, rgb(70, 76, 94))
		c.rect(8, 8, 16, 10, rgb(30, 50, 60))
		c.rect(9, 9, 14, 1, rgb(90, 240, 190))
		c.rect(9, 12, 9, 1, rgb(70, 200, 160))
		c.rect(9, 15, 11, 1, rgb(70, 200, 160))
		c.disc(12, 22, 2, rgb(230, 70, 60))
		c.disc(19, 22, 2, rgb(90, 200, 230))
		c.rect(23, 3, 1, 4, rgb(200, 200, 210))
	case IBook:
		c.rect(7, 7, 18, 20, rgb(150, 40, 50))
		c.rect(9, 8, 15, 18, rgb(240, 232, 210))
		c.rect(7, 7, 3, 20, rgb(110, 26, 36))
		c.rect(12, 12, 9, 2, rgb(150, 150, 160))
		c.rect(12, 16, 9, 1, rgb(190, 190, 200))
		c.rect(12, 19, 6, 1, rgb(190, 190, 200))
	}
	c.outline(dark)
	c.shadow(17, 28, 10, 2.5, 50)
	return c.image()
}

// lampRadius is how far (in tiles) a working lamp lights: used for both the glow and the
// "lit" work-speed bonus, so what you see is what you get.
const lampRadius = 8.0

// glowAlpha is the light's brightness at distance d (0 = centre, 1 = edge of the lit radius):
// nearly full strength across most of the circle, then a short, crisp falloff at the edge.
func glowAlpha(d float64) float64 {
	if d >= 1 {
		return 0
	}
	x := math.Min(1, math.Max(0, (d-0.62)/0.38)) // 0 inside 62% of the radius, 1 at the edge
	edge := 1 - x*x*(3-2*x)                      // smoothstep down
	return edge * (0.72 + 0.28*(1-d))
}

// paintGlow is the lamp light sprite (drawn additively).
func paintGlow() *ebiten.Image {
	const n = 128
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)-n/2+0.5, float64(y)-n/2+0.5) / (n / 2)
			if a := glowAlpha(d); a > 0 {
				img.SetNRGBA(x, y, color.NRGBA{255, 218, 130, uint8(a * 255)})
			}
		}
	}
	if headlessArt {
		return nil
	}
	return ebiten.NewImageFromImage(img)
}
