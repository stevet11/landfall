package main

import "github.com/hajimehoshi/ebiten/v2"

// Sprites for the new crops, the stove, tools, clothing and food.

// plant draws one plant of a crop at (x, y), growth stage 0-3.
func plant(c *canvas, kind CropKind, x, y, stage int) {
	fx, fy := float64(x), float64(y)
	green, light := rgb(70, 160, 60), rgb(116, 204, 84)
	switch kind {
	case CropWheat:
		switch stage {
		case 0:
			c.rect(x, y-2, 1, 2, green)
			c.rect(x+2, y-2, 1, 2, green)
		case 1:
			for i := 0; i < 3; i++ {
				c.rect(x+i*2-1, y-5, 1, 5, green)
			}
		case 2:
			for i := 0; i < 3; i++ {
				c.rect(x+i*2-1, y-8, 1, 8, light)
				c.rect(x+i*2-1, y-9, 1, 3, rgb(150, 190, 70))
			}
		default:
			for i := 0; i < 3; i++ {
				c.rect(x+i*2-1, y-9, 1, 9, rgb(206, 176, 80))
				c.ellipse(fx+float64(i*2), fy-11, 1.3, 3, rgb(240, 206, 90))
				c.set(x+i*2-1, y-12, rgb(255, 232, 140))
			}
		}
	case CropCorn:
		h := []int{3, 6, 10, 12}[stage]
		c.rect(x, y-h, 2, h, green)
		if stage >= 1 {
			c.ellipse(fx-3, fy-float64(h)*0.55, 3.4, 1.4, light)
			c.ellipse(fx+5, fy-float64(h)*0.7, 3.4, 1.4, light)
		}
		if stage == 3 {
			c.ellipse(fx+3.5, fy-7, 1.6, 3.2, rgb(238, 206, 70)) // cobs
			c.set(x+3, y-10, rgb(250, 235, 140))
		}
	case CropCotton:
		h := []int{3, 5, 8, 9}[stage]
		c.rect(x, y-h, 2, h, rgb(96, 130, 60))
		if stage >= 1 {
			c.ellipse(fx-2, fy-float64(h)*0.6, 2.6, 1.5, green)
			c.ellipse(fx+4, fy-float64(h)*0.5, 2.6, 1.5, green)
		}
		if stage == 3 {
			for _, o := range [][2]float64{{-2, -9}, {3, -10}, {0, -6}, {4, -6}} {
				c.disc(fx+o[0], fy+o[1], 1.9, rgb(246, 246, 250))
				c.set(int(fx+o[0]), int(fy+o[1]), rgb(214, 218, 230))
			}
		}
	case CropAloe:
		n := []int{1, 2, 4, 5}[stage]
		for i := 0; i < n; i++ {
			a := float64(i)/float64(n)*3.0 - 1.5
			c.line(fx+1, fy, fx+1+a*3.4, fy-float64(stage+2)*1.5-float64(i%2)*2, rgb(72, 160, 140))
			c.line(fx+2, fy, fx+2+a*3.4, fy-float64(stage+2)*1.5-float64(i%2)*2, rgb(104, 196, 170))
		}
		if stage == 3 {
			c.rect(x+1, y-13, 1, 6, rgb(90, 150, 80))
			c.disc(fx+1.5, fy-14, 1.6, rgb(240, 110, 70))
		}
	case CropTobacco:
		h := []int{3, 6, 9, 11}[stage]
		c.rect(x, y-h, 2, h, rgb(90, 130, 56))
		if stage >= 1 {
			c.ellipse(fx-3, fy-float64(h)*0.35, 3.4, 1.9, rgb(96, 150, 62))
			c.ellipse(fx+5, fy-float64(h)*0.5, 3.4, 1.9, rgb(112, 168, 70))
		}
		if stage == 3 {
			c.disc(fx+1.5, fy-12, 1.7, rgb(244, 150, 190)) // pink flowers
			c.set(int(fx+1), int(fy-12), rgb(255, 210, 230))
		}
	default:
		_ = light
	}
}

func paintCropKind(kind CropKind, stage int) *ebiten.Image {
	if kind == CropVeg {
		return paintCrop(stage)
	}
	c := newCanvas()
	for _, p := range [][2]int{{8, 14}, {22, 14}, {8, 29}, {22, 29}} {
		plant(c, kind, p[0], p[1], stage)
	}
	c.outline(col{40, 30, 20, 140})
	return c.image()
}

func paintStove(frame int) *ebiten.Image {
	c := newCanvas()
	c.rect(3, 6, 26, 22, rgb(56, 56, 62))
	c.rect(5, 8, 22, 18, rgb(86, 88, 98))
	c.rect(5, 8, 22, 2, rgb(130, 134, 148))
	c.rect(10, 2, 5, 6, rgb(48, 48, 54)) // flue
	// hob with a pot
	c.disc(11, 15, 4.5, rgb(40, 40, 46))
	c.disc(11, 15, 3.5, rgb(150, 154, 166))
	c.disc(11, 14.5, 2.5, rgb(190, 194, 206))
	c.disc(22, 15, 4, rgb(40, 40, 46))
	// door with a window and fire glow
	c.rect(8, 20, 16, 6, rgb(36, 36, 40))
	if frame > 0 {
		c.ellipse(16, 24, 5, 2.4+float64(frame)*0.5, rgb(240, 100, 30))
		c.ellipse(16, 24.5, 3, 1.6+float64(frame)*0.4, rgb(255, 200, 70))
	}
	c.rect(26, 12, 2, 3, rgb(210, 190, 90)) // knob
	c.outline(dark)
	c.shadow(16, 29, 14, 2.5, 55)
	return c.image()
}

func paintFarmGoods(it Item) *ebiten.Image {
	c := newCanvas()
	switch it {
	case IPickaxe:
		c.line(8, 27, 23, 9, rgb(120, 82, 46)) // haft
		c.line(9, 27, 24, 9, rgb(160, 114, 66))
		c.line(7, 27, 22, 9, rgb(90, 60, 34))
		c.line(8, 7, 26, 9, rgb(186, 194, 208)) // pick head
		c.line(8, 8, 26, 10, rgb(150, 158, 174))
		c.line(26, 9, 29, 13, rgb(186, 194, 208))
		c.line(8, 7, 6, 12, rgb(186, 194, 208))
		c.rect(21, 7, 4, 5, rgb(110, 118, 134))
	case IParka:
		c.ellipse(16, 18, 10, 9, rgb(120, 150, 190)) // body
		c.rect(5, 12, 5, 13, rgb(100, 130, 170))     // sleeves
		c.rect(22, 12, 5, 13, rgb(100, 130, 170))
		c.ellipse(16, 10, 6.5, 4.5, rgb(236, 236, 240)) // fur hood
		c.ellipse(16, 11, 3.5, 2.6, rgb(70, 90, 120))
		c.rect(15, 14, 2, 12, rgb(70, 90, 120)) // zip
		c.rect(11, 22, 10, 2, rgb(90, 118, 156))
	case IWheat:
		for i := 0; i < 5; i++ {
			x := 7 + i*4
			c.line(float64(x), 28, float64(x+i-2), 11, rgb(206, 176, 80))
			c.ellipse(float64(x+i-2), 9, 1.8, 4, rgb(240, 206, 90))
		}
		c.rect(6, 21, 20, 2, rgb(150, 100, 60)) // tie
	case ICorn:
		c.ellipse(16, 16, 5, 10, rgb(238, 206, 70))
		for y := 8; y < 24; y += 3 {
			for x := 12; x < 20; x += 3 {
				c.set(x, y, rgb(214, 170, 40))
			}
		}
		c.ellipse(11, 20, 3, 8, rgb(96, 160, 64)) // husk
		c.ellipse(21, 20, 3, 8, rgb(110, 176, 72))
	case ICotton:
		for _, o := range [][2]float64{{11, 20}, {21, 20}, {16, 12}, {16, 21}} {
			c.disc(o[0], o[1], 5, rgb(246, 246, 250))
			c.disc(o[0]-1, o[1]-1, 2, rgb(255, 255, 255))
		}
		c.rect(13, 24, 6, 3, rgb(96, 130, 60))
	case ITobacco:
		for i := 0; i < 3; i++ {
			c.ellipse(16, 14+float64(i)*5, 9, 3.4, rgb(120+uint8(i*14), 150, 62))
			c.line(8, 14+float64(i)*5, 24, 14+float64(i)*5, rgb(86, 110, 44))
		}
		c.rect(14, 24, 4, 3, rgb(120, 84, 46))
	case ICloth:
		c.rect(5, 10, 22, 6, rgb(206, 216, 236))
		c.rect(5, 16, 22, 6, rgb(186, 198, 224))
		c.rect(5, 22, 22, 5, rgb(166, 180, 212))
		c.rect(5, 10, 22, 1, rgb(240, 244, 252))
		c.rect(5, 16, 22, 1, rgb(228, 234, 248))
	case IBread:
		c.ellipse(16, 18, 11, 7, rgb(196, 132, 62)) // loaf
		c.ellipse(16, 16.5, 10, 5.5, rgb(224, 160, 82))
		for _, x := range []int{10, 15, 20} {
			c.line(float64(x), 13, float64(x+2), 17, rgb(176, 108, 48)) // scoring
		}
	case IMeal:
		c.ellipse(16, 20, 11, 6, rgb(170, 160, 150))   // bowl
		c.ellipse(16, 18, 9.5, 4.5, rgb(210, 150, 80)) // stew
		for _, o := range [][2]float64{{12, 17}, {17, 18}, {20, 17}} {
			c.disc(o[0], o[1], 1.6, rgb(120, 70, 44))
		}
		c.disc(14, 17, 1.4, rgb(240, 200, 70))
		c.rect(12, 8, 1, 5, col{230, 230, 235, 150}) // steam
		c.rect(17, 6, 1, 6, col{230, 230, 235, 150})
		c.rect(21, 9, 1, 4, col{230, 230, 235, 150})
	}
	c.outline(dark)
	c.shadow(17, 28, 10, 2.5, 45)
	return c.image()
}

// ---------- floors ----------

func paintStoneFloor(v int) *ebiten.Image {
	c := newCanvas()
	mortar := rgb(82, 84, 94)
	c.rect(0, 0, 32, 32, mortar)
	// irregular flagstones: three rows with staggered joints
	rows := [][]int{{0, 11, 22, 32}, {0, 8, 19, 32}, {0, 13, 24, 32}}
	if v == 1 {
		rows = [][]int{{0, 9, 21, 32}, {0, 14, 25, 32}, {0, 7, 18, 32}}
	}
	ys := []int{0, 11, 21, 32}
	for r, cuts := range rows {
		for i := 0; i+1 < len(cuts); i++ {
			x0, x1, y0, y1 := cuts[i], cuts[i+1], ys[r], ys[r+1]
			shade := noise(r*7+i, v, 130)
			k := lerpc(rgb(122, 124, 134), rgb(156, 158, 168), shade)
			c.rect(x0+1, y0+1, x1-x0-1, y1-y0-1, k)
			c.rect(x0+1, y0+1, x1-x0-1, 1, lerpc(k, rgb(220, 222, 230), 0.35))
			c.rect(x0+1, y1-1, x1-x0-1, 1, lerpc(k, rgb(60, 62, 72), 0.3))
			for n := 0; n < 3; n++ { // speckle
				c.set(x0+2+int(noise(n, r*5+i, 131)*float64(x1-x0-3)), y0+2+int(noise(n+9, r*5+i, 132)*float64(y1-y0-3)), lerpc(k, rgb(90, 92, 100), 0.5))
			}
		}
	}
	return c.image()
}

func paintTileFloor(v int) *ebiten.Image {
	c := newCanvas()
	grout := rgb(170, 178, 192)
	c.rect(0, 0, 32, 32, grout)
	for ty := 0; ty < 4; ty++ {
		for tx := 0; tx < 4; tx++ {
			x, y := tx*8, ty*8
			base := rgb(232, 236, 240)
			if (tx+ty+v)%2 == 1 {
				base = rgb(222, 230, 238)
			}
			c.rect(x+1, y+1, 7, 7, base)
			c.rect(x+1, y+1, 7, 1, rgb(250, 252, 255))
			c.rect(x+1, y+7, 7, 1, rgb(196, 204, 216))
			if (tx*3+ty*5+v)%4 == 0 { // a little blue diamond now and then
				c.set(x+4, y+3, rgb(96, 140, 200))
				c.set(x+3, y+4, rgb(96, 140, 200))
				c.set(x+5, y+4, rgb(96, 140, 200))
				c.set(x+4, y+5, rgb(96, 140, 200))
			}
		}
	}
	return c.image()
}

func paintCheckerFloor(v int) *ebiten.Image {
	c := newCanvas()
	for ty := 0; ty < 4; ty++ {
		for tx := 0; tx < 4; tx++ {
			x, y := tx*8, ty*8
			base, hi, lo := rgb(236, 238, 242), rgb(255, 255, 255), rgb(190, 196, 206)
			if (tx+ty+v)%2 == 1 {
				base, hi, lo = rgb(44, 46, 54), rgb(78, 82, 94), rgb(26, 28, 34)
			}
			c.rect(x, y, 8, 8, base)
			c.rect(x, y, 8, 1, hi)
			c.rect(x, y+7, 8, 1, lo)
		}
	}
	return c.image()
}

// ---------- entertainment ----------

func paintHorsePit() *ebiten.Image {
	c := newCanvas()
	// a rectangle of raked sand with an iron stake at each end
	c.rect(4, 8, 24, 18, rgb(196, 168, 112))
	for y := 9; y < 26; y += 3 {
		c.rect(5, y, 22, 1, rgb(176, 148, 94))
	}
	c.rect(4, 8, 24, 1, rgb(110, 84, 50))
	c.rect(4, 25, 24, 1, rgb(110, 84, 50))
	c.rect(4, 8, 1, 18, rgb(110, 84, 50))
	c.rect(27, 8, 1, 18, rgb(110, 84, 50))
	for _, x := range []int{8, 23} { // stakes
		c.rect(x, 14, 2, 6, rgb(90, 94, 106))
		c.rect(x, 13, 2, 1, rgb(190, 196, 210))
	}
	// a couple of horseshoes lying about
	for _, p := range [][2]int{{13, 20}, {17, 11}} {
		c.set(p[0], p[1], rgb(150, 156, 170))
		c.set(p[0]+1, p[1], rgb(150, 156, 170))
		c.set(p[0]+2, p[1], rgb(150, 156, 170))
		c.set(p[0], p[1]+1, rgb(150, 156, 170))
		c.set(p[0]+2, p[1]+1, rgb(150, 156, 170))
	}
	c.outline(dark)
	return c.image()
}

func paintCheckers() *ebiten.Image {
	c := newCanvas()
	c.rect(2, 6, 28, 20, rgb(112, 76, 44)) // table
	c.rect(3, 7, 26, 18, rgb(166, 120, 72))
	c.rect(3, 7, 26, 1, rgb(196, 150, 98))
	// the board: 6x4 squares of 4px
	for bx := 0; bx < 6; bx++ {
		for by := 0; by < 4; by++ {
			k := rgb(236, 218, 176)
			if (bx+by)%2 == 1 {
				k = rgb(110, 60, 44)
			}
			c.rect(4+bx*4, 8+by*4, 4, 4, k)
		}
	}
	// a few pieces
	for _, p := range [][3]int{{5, 9, 0}, {13, 9, 0}, {9, 13, 1}, {17, 17, 1}, {21, 13, 0}} {
		k := rgb(210, 60, 56)
		if p[2] == 1 {
			k = rgb(40, 40, 46)
		}
		c.disc(float64(p[0])+0.5, float64(p[1])+0.5, 1.4, k)
	}
	c.outline(dark)
	c.shadow(16, 27, 14, 2.5, 55)
	return c.image()
}

func paintShrine() *ebiten.Image {
	c := newCanvas()
	c.rect(7, 20, 18, 7, rgb(120, 124, 136)) // stone base
	c.rect(7, 20, 18, 1, rgb(178, 182, 194))
	c.rect(10, 15, 12, 5, rgb(140, 144, 156))
	c.ellipse(16, 11, 4, 5, rgb(206, 208, 218)) // a small statue
	c.disc(16, 5.5, 3, rgb(222, 224, 232))
	c.set(15, 5, dark)
	c.set(17, 5, dark)
	for _, x := range []int{9, 22} { // two candles
		c.rect(x, 17, 2, 3, rgb(240, 236, 220))
		c.rect(x, 15, 2, 2, rgb(255, 200, 80))
		c.set(x, 14, rgb(255, 240, 160))
	}
	c.outline(dark)
	c.shadow(16, 28, 11, 2.5, 55)
	return c.image()
}
