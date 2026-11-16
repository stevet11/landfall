package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// All art is generated in code at startup as 32x32 pixel-art sprites, so the
// game ships as a single executable with no asset files.

type col = color.NRGBA

func rgb(r, g, b uint8) col { return col{r, g, b, 255} }

type canvas struct{ img *image.NRGBA }

func newCanvas() *canvas { return &canvas{image.NewNRGBA(image.Rect(0, 0, 32, 32))} }

func (c *canvas) set(x, y int, k col) {
	if x >= 0 && y >= 0 && x < 32 && y < 32 {
		c.img.SetNRGBA(x, y, k)
	}
}

func (c *canvas) get(x, y int) col {
	if x < 0 || y < 0 || x >= 32 || y >= 32 {
		return col{}
	}
	return c.img.NRGBAAt(x, y)
}

func (c *canvas) rect(x, y, w, h int, k col) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			c.set(i, j, k)
		}
	}
}

func (c *canvas) ellipse(cx, cy, rx, ry float64, k col) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				c.set(x, y, k)
			}
		}
	}
}

func (c *canvas) disc(cx, cy, r float64, k col) { c.ellipse(cx, cy, r, r, k) }

func (c *canvas) line(x0, y0, x1, y1 float64, k col) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))*2) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		c.set(int(x0+(x1-x0)*t), int(y0+(y1-y0)*t), k)
	}
}

// outline adds a 1px border around opaque pixels.
func (c *canvas) outline(k col) {
	src := image.NewNRGBA(c.img.Rect)
	copy(src.Pix, c.img.Pix)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if src.NRGBAAt(x, y).A != 0 {
				continue
			}
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if nx >= 0 && ny >= 0 && nx < 32 && ny < 32 && src.NRGBAAt(nx, ny).A > 200 {
					c.img.SetNRGBA(x, y, k)
					break
				}
			}
		}
	}
}

// shadow paints a soft ellipse only onto empty pixels (call after outline).
func (c *canvas) shadow(cx, cy, rx, ry float64, a uint8) {
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 && c.img.NRGBAAt(x, y).A == 0 {
				c.img.SetNRGBA(x, y, col{0, 0, 0, a})
			}
		}
	}
}

// headlessArt lets tests render sprites without a graphics context.
var headlessArt bool
var headlessSprites []*image.NRGBA

func (c *canvas) image() *ebiten.Image {
	if headlessArt {
		headlessSprites = append(headlessSprites, c.img)
		return nil
	}
	return ebiten.NewImageFromImage(c.img)
}

func noise(x, y, s int) float64 { return hash2(x, y, int64(s)) }

func lerpc(a, b col, t float64) col {
	f := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return col{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

var dark = rgb(34, 28, 30)

// ---------- terrain ----------

func paintGround(g Ground, seed, frame int) *ebiten.Image {
	c := newCanvas()
	var base, lo, hi col
	switch g {
	case GGrass:
		base, lo, hi = rgb(86, 140, 70), rgb(72, 122, 60), rgb(104, 158, 82)
	case GDirt:
		base, lo, hi = rgb(128, 104, 76), rgb(112, 90, 64), rgb(146, 120, 88)
	case GSand:
		base, lo, hi = rgb(206, 190, 130), rgb(192, 174, 116), rgb(222, 208, 152)
	default:
		base, lo, hi = rgb(50, 98, 160), rgb(42, 86, 146), rgb(86, 140, 200)
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			n := noise(x, y, seed)
			k := base
			if n < 0.16 {
				k = lo
			} else if n > 0.9 {
				k = hi
			}
			if g == GWater {
				w := math.Sin(float64(x)*0.42+float64(frame)*math.Pi/2+float64(y)*0.25+float64(seed)) + math.Sin(float64(y)*0.5-float64(frame)*1.3+float64(seed)*2)
				if w > 1.55 {
					k = hi
				} else if w < -1.6 {
					k = lo
				}
			}
			c.set(x, y, k)
		}
	}
	switch g {
	case GGrass:
		for i := 0; i < 5; i++ {
			x, y := int(noise(i, 1, seed)*28)+2, int(noise(i, 2, seed)*24)+5
			c.set(x, y, lo)
			c.set(x-1, y-1, hi)
			c.set(x+1, y-1, hi)
			c.set(x, y-2, hi)
		}
	case GDirt:
		for i := 0; i < 4; i++ {
			x, y := int(noise(i, 3, seed)*27)+2, int(noise(i, 4, seed)*27)+2
			c.rect(x, y, 2, 2, rgb(154, 128, 96))
			c.set(x+1, y+1, rgb(96, 76, 54))
		}
	case GSand:
		for i := 0; i < 3; i++ {
			y := int(noise(i, 5, seed)*28) + 2
			for x := 0; x < 32; x++ {
				if int(float64(y)+2*math.Sin(float64(x)*0.3+float64(i))) >= 0 && noise(x, i, seed) < 0.75 {
					c.set(x, y+int(2*math.Sin(float64(x)*0.3+float64(i))), lo)
				}
			}
		}
	}
	return c.image()
}

func paintTree(v int) *ebiten.Image {
	c := newCanvas()
	c.rect(14, 18, 4, 8, rgb(92, 62, 36))
	var d, m, l col
	switch v {
	case 2:
		d, m, l = rgb(24, 70, 50), rgb(34, 92, 62), rgb(56, 120, 80)
	default:
		d, m, l = rgb(30, 90, 40), rgb(46, 118, 54), rgb(78, 152, 76)
	}
	c.disc(16, 14, 13+float64(v%2), d)
	c.disc(14.5, 12.5, 10.5, m)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if c.get(x, y) == m {
				n := noise(x, y, 40+v)
				if n > 0.8 && x+y < 28 {
					c.set(x, y, l)
				} else if n < 0.12 {
					c.set(x, y, d)
				}
			}
		}
	}
	if v == 2 { // conifer spikes
		for a := 0.0; a < 6.28; a += 0.52 {
			c.line(16, 14, 16+math.Cos(a)*12, 14+math.Sin(a)*12, m)
		}
		c.disc(16, 14, 3, l)
	} else {
		c.disc(11, 9, 3, l)
	}
	c.outline(dark)
	c.shadow(18, 24, 12, 5, 70)
	return c.image()
}

func paintRock(v int) *ebiten.Image {
	c := newCanvas()
	base, hi, lo := rgb(122, 124, 132), rgb(170, 172, 180), rgb(84, 86, 96)
	c.ellipse(16, 18, 12, 9, lo)
	c.ellipse(15, 16, 11, 8, base)
	if v == 1 {
		c.ellipse(21, 13, 6, 5, base)
		c.ellipse(20, 12, 4, 3, hi)
	}
	c.ellipse(12, 13, 5, 3.5, hi)
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if c.get(x, y) == base && noise(x, y, 60+v) < 0.1 {
				c.set(x, y, lo)
			}
		}
	}
	c.line(14, 15, 18, 22, lo)
	c.outline(dark)
	c.shadow(17, 25, 12, 4, 70)
	return c.image()
}

func paintBush() *ebiten.Image {
	c := newCanvas()
	c.disc(16, 17, 10, rgb(34, 96, 46))
	c.disc(13, 14, 7, rgb(50, 128, 60))
	c.disc(20, 19, 5, rgb(42, 112, 52))
	for _, p := range [][2]int{{11, 13}, {17, 11}, {21, 17}, {14, 20}, {19, 22}, {9, 18}} {
		c.set(p[0], p[1], rgb(220, 40, 70))
		c.set(p[0]+1, p[1], rgb(220, 40, 70))
		c.set(p[0], p[1]+1, rgb(170, 24, 56))
		c.set(p[0]+1, p[1]+1, rgb(170, 24, 56))
	}
	c.outline(dark)
	c.shadow(17, 25, 10, 3, 60)
	return c.image()
}

func paintHerb() *ebiten.Image {
	c := newCanvas()
	for a := 0.0; a < 6.28; a += 1.05 {
		c.ellipse(16+math.Cos(a)*6, 17+math.Sin(a)*6, 4.5, 2.5, rgb(96, 200, 150))
		c.line(16, 17, 16+math.Cos(a)*8, 17+math.Sin(a)*8, rgb(60, 150, 110))
	}
	c.disc(16, 17, 3, rgb(250, 250, 200))
	c.set(16, 17, rgb(240, 200, 80))
	c.outline(rgb(24, 70, 50))
	c.shadow(17, 24, 9, 3, 50)
	return c.image()
}

// ---------- buildings ----------

func paintWall() *ebiten.Image {
	c := newCanvas()
	c.rect(0, 0, 32, 32, rgb(84, 80, 76))
	for row := 0; row < 4; row++ {
		off := 0
		if row%2 == 1 {
			off = 8
		}
		for bx := -8 + off; bx < 32; bx += 16 {
			shade := noise(bx, row, 70)
			k := lerpc(rgb(120, 114, 108), rgb(146, 138, 130), shade)
			c.rect(bx+1, row*8+1, 14, 6, k)
			c.rect(bx+1, row*8+1, 14, 1, rgb(168, 160, 150))
			c.rect(bx+1, row*8+6, 14, 1, rgb(96, 90, 86))
		}
	}
	c.rect(0, 0, 32, 1, rgb(60, 56, 54))
	c.rect(0, 31, 32, 1, rgb(60, 56, 54))
	return c.image()
}

func paintFloor(v int) *ebiten.Image {
	c := newCanvas()
	for p := 0; p < 4; p++ {
		y := p * 8
		off := 0
		if (p+v)%2 == 1 {
			off = 14
		}
		for x := 0; x < 32; x++ {
			for yy := y; yy < y+8; yy++ {
				n := noise(x, yy, 80+p)
				k := rgb(176, 132, 84)
				if n < 0.12 {
					k = rgb(160, 118, 72)
				} else if n > 0.92 {
					k = rgb(192, 148, 98)
				}
				if yy == y+7 || (x+off)%28 == 0 {
					k = rgb(116, 82, 50)
				}
				c.set(x, yy, k)
			}
		}
	}
	return c.image()
}

func paintFarm() *ebiten.Image {
	c := newCanvas()
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			k := rgb(104, 74, 48)
			if y%8 < 2 {
				k = rgb(80, 56, 36)
			} else if y%8 == 2 {
				k = rgb(122, 90, 60)
			}
			if noise(x, y, 90) > 0.93 {
				k = rgb(130, 98, 66)
			}
			c.set(x, y, k)
		}
	}
	return c.image()
}

func paintCrop(stage int) *ebiten.Image {
	c := newCanvas()
	green, light := rgb(70, 160, 60), rgb(110, 200, 80)
	for _, p := range [][2]int{{8, 12}, {22, 12}, {8, 28}, {22, 28}} {
		x, y := p[0], p[1]
		switch stage {
		case 0:
			c.rect(x, y-2, 2, 2, green)
			c.set(x-1, y-3, light)
			c.set(x+2, y-3, light)
		case 1:
			c.rect(x, y-5, 2, 5, green)
			c.ellipse(float64(x)-2, float64(y)-4, 2.5, 1.5, light)
			c.ellipse(float64(x)+4, float64(y)-5, 2.5, 1.5, light)
		case 2:
			c.rect(x, y-8, 2, 8, green)
			c.ellipse(float64(x)-3, float64(y)-6, 3.5, 1.7, light)
			c.ellipse(float64(x)+5, float64(y)-4, 3.5, 1.7, light)
			c.ellipse(float64(x)+1, float64(y)-9, 2, 2.5, light)
		default:
			c.rect(x, y-9, 2, 9, rgb(150, 160, 60))
			c.ellipse(float64(x)-3, float64(y)-5, 3.5, 1.7, rgb(150, 180, 70))
			c.ellipse(float64(x)+1, float64(y)-11, 2.5, 4, rgb(226, 190, 70))
			c.set(x, y-12, rgb(250, 226, 120))
			c.set(x+1, y-9, rgb(180, 140, 40))
		}
	}
	c.outline(col{40, 30, 20, 140})
	return c.image()
}

func paintDoor() *ebiten.Image {
	c := newCanvas()
	c.rect(0, 8, 32, 16, rgb(80, 56, 34))
	c.rect(2, 10, 28, 12, rgb(158, 112, 66))
	for x := 2; x < 30; x += 7 {
		c.rect(x, 10, 1, 12, rgb(120, 84, 48))
	}
	c.rect(2, 10, 28, 1, rgb(184, 136, 84))
	c.disc(25, 16, 1.6, rgb(236, 206, 96))
	c.rect(0, 5, 3, 22, rgb(60, 56, 54))
	c.rect(29, 5, 3, 22, rgb(60, 56, 54))
	return c.image()
}

func paintBed() *ebiten.Image {
	c := newCanvas()
	c.rect(3, 2, 26, 28, rgb(110, 74, 42))
	c.rect(5, 4, 22, 24, rgb(200, 204, 224))
	c.rect(7, 5, 18, 7, rgb(250, 250, 255))
	c.rect(7, 5, 18, 1, rgb(220, 224, 240))
	c.rect(5, 14, 22, 14, rgb(100, 114, 188))
	for x := 5; x < 27; x += 4 {
		c.rect(x, 14, 1, 14, rgb(88, 100, 170))
	}
	c.rect(5, 14, 22, 1, rgb(140, 154, 220))
	c.outline(dark)
	return c.image()
}

func paintHeater(frame int) *ebiten.Image {
	c := newCanvas()
	c.rect(3, 3, 26, 26, rgb(64, 64, 72))
	c.rect(5, 5, 22, 22, rgb(40, 40, 46))
	for y := 8; y < 26; y += 4 {
		c.rect(7, y, 18, 1, rgb(86, 86, 96))
	}
	c.rect(3, 3, 26, 1, rgb(110, 110, 120))
	if frame > 0 {
		h := 9.0 + float64(frame)
		c.ellipse(16, 18, 7, h*0.7, rgb(240, 90, 30))
		c.ellipse(16, 19, 5, h*0.55, rgb(255, 160, 40))
		c.ellipse(16, 20, 2.5, h*0.35, rgb(255, 232, 130))
	}
	c.outline(dark)
	return c.image()
}

// ---------- items ----------

func paintItem(it Item) *ebiten.Image {
	c := newCanvas()
	switch it {
	case IWood:
		for i, y := range []int{9, 15, 21} {
			x0 := 5 + (i%2)*3
			c.rect(x0, y, 20, 6, rgb(150, 100, 56))
			c.rect(x0, y, 20, 1, rgb(184, 130, 76))
			c.rect(x0, y+5, 20, 1, rgb(112, 74, 40))
			c.disc(float64(x0)+19, float64(y)+3, 3, rgb(214, 170, 110))
			c.disc(float64(x0)+19, float64(y)+3, 1.2, rgb(150, 100, 56))
		}
	case IStone:
		c.ellipse(11, 20, 7, 5, rgb(122, 124, 134))
		c.ellipse(21, 21, 6, 5, rgb(138, 140, 150))
		c.ellipse(16, 13, 7, 5.5, rgb(150, 152, 162))
		c.ellipse(14, 11, 3, 2, rgb(190, 192, 200))
	case IFood:
		for _, p := range [][2]float64{{10, 19}, {16, 21}, {13, 14}, {20, 17}, {16, 12}} {
			c.disc(p[0], p[1], 3.4, rgb(212, 40, 70))
			c.set(int(p[0])-1, int(p[1])-1, rgb(255, 150, 160))
		}
		c.ellipse(22, 22, 4, 3, rgb(176, 82, 70))
		c.set(25, 20, rgb(240, 236, 220))
		c.set(26, 19, rgb(240, 236, 220))
	case IMedicine:
		c.rect(10, 12, 12, 14, rgb(80, 190, 140))
		c.rect(11, 13, 3, 11, rgb(150, 240, 200))
		c.rect(13, 7, 6, 5, rgb(176, 128, 74))
		c.rect(12, 18, 8, 2, rgb(250, 250, 250))
		c.rect(15, 15, 2, 8, rgb(250, 250, 250))
	}
	c.outline(dark)
	c.shadow(17, 27, 10, 2.5, 50)
	return c.image()
}

// ---------- creatures ----------

var skins = []col{rgb(244, 208, 170), rgb(222, 170, 126), rgb(176, 124, 86), rgb(126, 84, 58)}
var hairs = []col{rgb(54, 36, 26), rgb(120, 78, 38), rgb(212, 172, 84), rgb(190, 70, 40), rgb(36, 36, 44), rgb(200, 200, 206)}

type colKey struct {
	shirt      col
	hair, skin int
	dir, frame int
	parka      bool
}

// paintColonist draws a colonist from above; parka dresses them in a blue parka with a fur hood.
func paintColonist(shirt, hair, skin col, dir, frame int, parka bool) *ebiten.Image {
	c := newCanvas()
	fur := rgb(238, 238, 242)
	if parka {
		shirt = rgb(96, 132, 186)
	}
	ph := []float64{0, 2, 0, -2}[frame]
	boot := rgb(66, 52, 46)
	sh := lerpc(shirt, rgb(0, 0, 0), 0.25)
	switch dir {
	case 0, 1: // down, up
		c.rect(11, int(23+ph), 4, 4, boot)
		c.rect(17, int(23-ph), 4, 4, boot)
		c.ellipse(16, 17, 8.5, 6, shirt)
		c.ellipse(16, 20, 7, 2.5, sh)
		c.disc(7, 17+ph, 2.4, skin)
		c.disc(25, 17-ph, 2.4, skin)
		if parka {
			c.disc(16, 12, 7.6, fur)
			c.disc(16, 12, 6.6, rgb(206, 208, 218))
			c.rect(15, 18, 2, 8, rgb(66, 88, 128)) // zip
		}
		c.disc(16, 12, 5.8, skin)
		if dir == 0 {
			for y := 6; y <= 10; y++ {
				for x := 10; x <= 22; x++ {
					if c.get(x, y) == skin {
						c.set(x, y, hair)
					}
				}
			}
			c.set(14, 13, dark)
			c.set(18, 13, dark)
		} else {
			c.disc(16, 12, 5.8, hair)
			c.rect(13, 15, 6, 2, hair)
		}
	default: // left, right
		s := 1.0
		if dir == 2 {
			s = -1
		}
		c.rect(int(14+s*ph*1.5), 23, 4, 4, boot)
		c.rect(int(14-s*ph*1.5), 21, 4, 4, boot)
		c.ellipse(16, 17, 5.5, 7, shirt)
		c.ellipse(16, 17+3, 4.5, 3, sh)
		c.disc(16+s*ph*1.5, 17, 2.4, skin)
		if parka {
			c.disc(16+s*1, 11.5, 7.4, fur)
			c.disc(16+s*1.5, 11.5, 6.4, rgb(206, 208, 218))
		}
		c.disc(16+s*1, 11.5, 5.6, skin)
		for y := 5; y <= 17; y++ {
			for x := 9; x <= 23; x++ {
				if c.get(x, y) == skin && ((s > 0 && float64(x) < 16-s*0.5) || (s < 0 && float64(x) > 16-s*0.5) || y < 8) && y < 15 {
					c.set(x, y, hair)
				}
			}
		}
		c.set(int(16+s*4), 12, dark)
	}
	c.outline(dark)
	c.shadow(16, 25, 9, 3, 60)
	return c.image()
}

func paintLying(shirt, hair, skin col) *ebiten.Image {
	c := newCanvas()
	c.ellipse(19, 16, 11, 6, lerpc(shirt, rgb(255, 255, 255), 0.1))
	c.rect(10, 12, 18, 2, lerpc(shirt, rgb(255, 255, 255), 0.35))
	c.disc(8, 16, 5.5, skin)
	c.ellipse(7, 14, 5.5, 3.5, hair)
	c.set(10, 17, dark)
	c.outline(dark)
	return c.image()
}

func paintDeer(frame int) *ebiten.Image {
	c := newCanvas()
	body, belly, leg := rgb(176, 124, 72), rgb(214, 176, 124), rgb(110, 78, 46)
	o := []int{-1, 1}[frame]
	for i, x := range []int{8, 11, 18, 21} {
		d := o
		if i%2 == 1 {
			d = -o
		}
		c.rect(x+d, 19, 2, 8, leg)
	}
	c.ellipse(14, 17, 10, 5.5, body)
	c.ellipse(13, 20, 7, 2, belly)
	c.line(21, 15, 25, 10, body)
	c.line(22, 16, 26, 11, body)
	c.ellipse(27, 10, 3.5, 2.6, body)
	c.rect(29, 10, 3, 2, rgb(70, 50, 36))
	c.set(27, 9, dark)
	c.line(26, 8, 24, 3, rgb(120, 90, 60))
	c.line(25, 5, 22, 4, rgb(120, 90, 60))
	c.line(27, 8, 29, 3, rgb(120, 90, 60))
	c.disc(4.5, 15, 2.2, rgb(250, 244, 232))
	for _, p := range [][2]int{{10, 14}, {14, 13}, {17, 16}, {12, 17}} {
		c.set(p[0], p[1], rgb(236, 210, 170))
	}
	c.outline(dark)
	c.shadow(14, 26, 11, 3, 55)
	return c.image()
}

func paintWolf(frame int) *ebiten.Image {
	c := newCanvas()
	body, back, leg := rgb(104, 108, 122), rgb(70, 74, 88), rgb(70, 74, 86)
	o := []int{-2, 2}[frame]
	for i, x := range []int{8, 11, 18, 21} {
		d := o
		if i%2 == 1 {
			d = -o
		}
		c.rect(x+d, 19, 2, 7, leg)
	}
	c.ellipse(14, 17, 10, 5, body)
	c.ellipse(14, 14, 9, 2.5, back)
	c.ellipse(4, 15, 5, 2.6, back)
	c.ellipse(2, 16, 2.5, 2, rgb(50, 52, 62))
	c.ellipse(24, 14, 4.5, 4, body)
	c.rect(27, 14, 5, 3, rgb(80, 84, 98))
	c.set(31, 14, dark)
	c.disc(22.5, 10, 1.8, back)
	c.disc(26, 10, 1.8, back)
	c.set(26, 13, rgb(255, 70, 50))
	c.set(27, 13, rgb(255, 200, 120))
	c.rect(28, 17, 1, 1, rgb(240, 240, 240))
	c.outline(dark)
	c.shadow(14, 25, 11, 3, 55)
	return c.image()
}

// ---------- atlas ----------

type Art struct {
	ground                  [4][4]*ebiten.Image
	water                   [4][4]*ebiten.Image
	tree                    [3]*ebiten.Image
	rock                    [2]*ebiten.Image
	bush                    *ebiten.Image
	herb                    *ebiten.Image
	wall                    [4]*ebiten.Image
	ore                     *ebiten.Image
	furnace                 [3]*ebiten.Image
	flame                   [3]*ebiten.Image
	floor                   [numFloors][2]*ebiten.Image
	farm                    *ebiten.Image
	crop                    [numCrops][4]*ebiten.Image
	stove                   [3]*ebiten.Image
	horse, checker, shrine  *ebiten.Image
	door                    [4]*ebiten.Image
	bed                     *ebiten.Image
	heater                  [3]*ebiten.Image
	item                    [numItems]*ebiten.Image
	deer                    [2]*ebiten.Image
	wolf                    [2]*ebiten.Image
	colonst                 map[colKey]*ebiten.Image
	bench, fab, solar, desk *ebiten.Image
	gen                     [3]*ebiten.Image
	lamp, eheater           [2]*ebiten.Image
	glow                    *ebiten.Image
	trader                  map[colKey]*ebiten.Image
	donkey, camel           [2]*ebiten.Image
	raider                  map[colKey]*ebiten.Image
	lying                   map[colKey]*ebiten.Image
}

var art *Art

func initArt() {
	a := &Art{colonst: map[colKey]*ebiten.Image{}, lying: map[colKey]*ebiten.Image{}}
	for g := GGrass; g <= GSand; g++ {
		for v := 0; v < 4; v++ {
			a.ground[g][v] = paintGround(g, 10+int(g)*7+v, 0)
		}
	}
	for v := 0; v < 4; v++ {
		for f := 0; f < 4; f++ {
			a.water[v][f] = paintGround(GWater, 3+v, f)
		}
	}
	for v := range a.tree {
		a.tree[v] = paintTree(v)
	}
	for v := range a.rock {
		a.rock[v] = paintRock(v)
	}
	a.bush, a.herb = paintBush(), paintHerb()
	for _, m := range []Material{MatWood, MatStone, MatIron} {
		a.wall[m], a.door[m] = paintWallMat(m), paintDoorMat(m)
	}
	a.wall[MatNone], a.door[MatNone] = a.wall[MatStone], a.door[MatWood]
	a.ore = paintOre()
	for f := range a.furnace {
		a.furnace[f], a.flame[f] = paintFurnace(f), paintFlame(f)
	}
	a.floor[FloorWood][0], a.floor[FloorWood][1] = paintFloor(0), paintFloor(1)
	for v := 0; v < 2; v++ {
		a.floor[FloorStone][v], a.floor[FloorTile][v], a.floor[FloorChecker][v] = paintStoneFloor(v), paintTileFloor(v), paintCheckerFloor(v)
	}
	a.farm, a.bed = paintFarm(), paintBed()
	for k := CropKind(0); k < numCrops; k++ {
		for s := 0; s < 4; s++ {
			a.crop[k][s] = paintCropKind(k, s)
		}
	}
	for f := range a.stove {
		a.stove[f] = paintStove(f)
	}
	a.horse, a.checker, a.shrine = paintHorsePit(), paintCheckers(), paintShrine()
	for f := range a.heater {
		a.heater[f] = paintHeater(f)
	}
	for it := IWood; it < numItems; it++ {
		if it >= IPickaxe {
			a.item[it] = paintFarmGoods(it)
		} else if it >= IKnife {
			a.item[it] = paintArms(it)
		} else if it >= IOre {
			a.item[it] = paintMetalGoods(it)
		} else if it >= ISilver {
			a.item[it] = paintGoods(it)
		} else {
			a.item[it] = paintItem(it)
		}
	}
	a.bench, a.fab, a.solar, a.desk = paintBench(), paintFab(), paintSolar(), paintDesk()
	for f := range a.gen {
		a.gen[f] = paintGen(f)
	}
	a.lamp[0], a.lamp[1] = paintLamp(false), paintLamp(true)
	a.eheater[0], a.eheater[1] = paintEHeater(false), paintEHeater(true)
	a.glow = paintGlow()
	a.trader = map[colKey]*ebiten.Image{}
	a.raider = map[colKey]*ebiten.Image{}
	for f := 0; f < 2; f++ {
		a.donkey[f], a.camel[f] = paintDonkey(f), paintCamel(f)
	}
	for f := 0; f < 2; f++ {
		a.deer[f], a.wolf[f] = paintDeer(f), paintWolf(f)
	}
	art = a
}

func (a *Art) colonist(c *Colonist, dir, frame int) *ebiten.Image {
	k := colKey{c.Col, c.ID % len(hairs), (c.ID / 2) % len(skins), dir, frame, c.Wear == IParka}
	if img, ok := a.colonst[k]; ok {
		return img
	}
	img := paintColonist(c.Col, hairs[k.hair], skins[k.skin], dir, frame, k.parka)
	a.colonst[k] = img
	return img
}

func (a *Art) lyingSprite(c *Colonist) *ebiten.Image {
	k := colKey{c.Col, c.ID % len(hairs), (c.ID / 2) % len(skins), 9, 0, false}
	if img, ok := a.lying[k]; ok {
		return img
	}
	img := paintLying(c.Col, hairs[k.hair], skins[k.skin])
	a.lying[k] = img
	return img
}

func (a *Art) traderSprite(dir, frame int) *ebiten.Image {
	k := colKey{rgb(150, 106, 56), 2, 1, dir, frame, false}
	if img, ok := a.trader[k]; ok {
		return img
	}
	img := paintColonist(k.shirt, hairs[2], skins[1], dir, frame, false)
	a.trader[k] = img
	return img
}

func (a *Art) raiderSprite(dir, frame int) *ebiten.Image {
	k := colKey{rgb(96, 38, 40), 4, 0, dir, frame, false}
	if img, ok := a.raider[k]; ok {
		return img
	}
	img := paintColonist(k.shirt, rgb(30, 28, 34), skins[1], dir, frame, false)
	a.raider[k] = img
	return img
}
