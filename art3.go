package main

import "github.com/hajimehoshi/ebiten/v2"

// Sprites for wall/door materials, ore, the furnace, iron and fire.

func paintWallMat(m Material) *ebiten.Image {
	switch m {
	case MatWood:
		c := newCanvas()
		c.rect(0, 0, 32, 32, rgb(84, 56, 32))
		for i := 0; i < 4; i++ {
			x := i * 8
			shade := noise(i, 3, 71)
			k := lerpc(rgb(150, 104, 62), rgb(178, 130, 80), shade)
			c.rect(x+1, 1, 6, 30, k)
			c.rect(x+1, 1, 1, 30, rgb(190, 142, 92))
			c.rect(x+6, 1, 1, 30, rgb(116, 78, 46))
			for y := 4; y < 30; y += 9 {
				c.set(x+3, y+int(noise(i, y, 5)*3), rgb(100, 68, 40))
			}
			c.set(x+3, 3, rgb(60, 44, 30))
			c.set(x+3, 28, rgb(60, 44, 30))
		}
		c.rect(0, 0, 32, 1, rgb(60, 40, 24))
		c.rect(0, 31, 32, 1, rgb(60, 40, 24))
		return c.image()
	case MatIron:
		c := newCanvas()
		c.rect(0, 0, 32, 32, rgb(52, 58, 70))
		for gy := 0; gy < 2; gy++ {
			for gx := 0; gx < 2; gx++ {
				x, y := gx*16, gy*16
				c.rect(x+1, y+1, 14, 14, rgb(112, 122, 140))
				c.rect(x+1, y+1, 14, 1, rgb(160, 172, 190))
				c.rect(x+1, y+14, 14, 1, rgb(80, 88, 104))
				for _, p := range [][2]int{{3, 3}, {12, 3}, {3, 12}, {12, 12}} {
					c.set(x+p[0], y+p[1], rgb(190, 198, 212))
					c.set(x+p[0]+1, y+p[1]+1, rgb(60, 66, 80))
				}
				c.line(float64(x+5), float64(y+8), float64(x+10), float64(y+8), rgb(96, 104, 122))
			}
		}
		c.rect(0, 0, 32, 1, rgb(34, 38, 48))
		c.rect(0, 31, 32, 1, rgb(34, 38, 48))
		return c.image()
	}
	return paintWall()
}

func paintDoorMat(m Material) *ebiten.Image {
	c := newCanvas()
	switch m {
	case MatStone:
		c.rect(0, 5, 3, 22, rgb(60, 56, 54))
		c.rect(29, 5, 3, 22, rgb(60, 56, 54))
		c.rect(0, 8, 32, 16, rgb(84, 80, 76))
		c.rect(2, 10, 28, 12, rgb(142, 138, 132))
		c.rect(2, 10, 28, 1, rgb(172, 168, 160))
		c.rect(2, 21, 28, 1, rgb(104, 100, 96))
		c.line(8, 12, 12, 19, rgb(110, 106, 100))
		c.disc(25, 16, 1.6, rgb(60, 56, 54))
	case MatIron:
		c.rect(0, 5, 3, 22, rgb(40, 44, 54))
		c.rect(29, 5, 3, 22, rgb(40, 44, 54))
		c.rect(0, 8, 32, 16, rgb(52, 58, 70))
		c.rect(2, 10, 28, 12, rgb(120, 130, 148))
		c.rect(2, 10, 28, 1, rgb(170, 182, 200))
		c.rect(2, 21, 28, 1, rgb(84, 92, 108))
		for x := 5; x < 28; x += 6 {
			c.set(x, 12, rgb(200, 208, 222))
			c.set(x, 19, rgb(200, 208, 222))
		}
		c.rect(23, 14, 5, 3, rgb(226, 190, 80))
	default:
		return paintDoor()
	}
	return c.image()
}

func paintOre() *ebiten.Image {
	c := newCanvas()
	base, hi, lo := rgb(104, 100, 106), rgb(150, 146, 152), rgb(70, 66, 74)
	c.ellipse(16, 18, 12, 9, lo)
	c.ellipse(15, 16, 11, 8, base)
	c.ellipse(12, 13, 5, 3.5, hi)
	for _, p := range [][3]int{{10, 17, 1}, {17, 12, 1}, {20, 18, 1}, {14, 20, 1}, {22, 14, 0}, {9, 12, 0}} {
		k := rgb(214, 130, 60)
		if p[2] == 0 {
			k = rgb(236, 180, 90)
		}
		c.rect(p[0], p[1], 2, 2, k)
		c.set(p[0], p[1], rgb(255, 214, 140))
	}
	c.outline(dark)
	c.shadow(17, 25, 12, 4, 70)
	return c.image()
}

func paintFurnace(frame int) *ebiten.Image {
	c := newCanvas()
	c.rect(3, 4, 26, 25, rgb(86, 82, 80))
	c.rect(5, 6, 22, 21, rgb(132, 126, 120))
	c.rect(5, 6, 22, 2, rgb(164, 158, 150))
	for _, p := range [][2]int{{7, 9}, {16, 9}, {22, 12}, {8, 22}, {20, 23}} {
		c.rect(p[0], p[1], 5, 3, rgb(112, 106, 100))
	}
	c.rect(9, 14, 14, 10, rgb(30, 28, 28)) // mouth
	c.rect(10, 3, 5, 5, rgb(70, 66, 64))   // chimney
	if frame > 0 {
		c.ellipse(16, 21, 5.5, 3.5+float64(frame)*0.6, rgb(240, 90, 30))
		c.ellipse(16, 22, 3.5, 2.4+float64(frame)*0.5, rgb(255, 170, 50))
		c.ellipse(16, 22.5, 1.8, 1.4, rgb(255, 236, 150))
	} else {
		c.rect(10, 20, 12, 3, rgb(60, 54, 52))
	}
	c.outline(dark)
	c.shadow(16, 29, 14, 2.5, 55)
	return c.image()
}

func paintFlame(frame int) *ebiten.Image {
	c := newCanvas()
	for i, p := range [][3]float64{{9, 22, 5}, {16, 18, 7}, {23, 22, 5}, {13, 12, 4}, {20, 13, 4}} {
		h := p[2] + float64((frame+i)%3)*1.5
		c.ellipse(p[0], p[1]-h*0.4, 3.2, h, col{240, 80, 20, 230})
		c.ellipse(p[0], p[1]-h*0.25, 2.2, h*0.7, col{255, 160, 40, 240})
		c.ellipse(p[0], p[1]-h*0.1, 1.2, h*0.4, col{255, 236, 150, 255})
	}
	return c.image()
}

func paintMetalGoods(it Item) *ebiten.Image {
	c := newCanvas()
	switch it {
	case IOre:
		c.ellipse(11, 20, 7, 5, rgb(104, 100, 106))
		c.ellipse(21, 21, 6, 5, rgb(120, 116, 122))
		c.ellipse(16, 13, 7, 5.5, rgb(134, 130, 136))
		for _, p := range [][2]int{{9, 19}, {20, 20}, {14, 11}, {18, 14}} {
			c.rect(p[0], p[1], 2, 2, rgb(220, 134, 60))
		}
	case IIron:
		for i, y := range []int{9, 15, 21} {
			x := 6 + (i%2)*3
			c.rect(x, y, 18, 5, rgb(132, 142, 160))
			c.rect(x, y, 18, 1, rgb(190, 200, 216))
			c.rect(x, y+4, 18, 1, rgb(84, 92, 108))
		}
	}
	c.outline(dark)
	c.shadow(17, 28, 10, 2.5, 50)
	return c.image()
}

func paintArms(it Item) *ebiten.Image {
	c := newCanvas()
	steel, steelHi, steelLo := rgb(190, 198, 212), rgb(232, 238, 248), rgb(120, 128, 144)
	wood, woodHi := rgb(120, 82, 46), rgb(160, 114, 66)
	switch it {
	case IKnife:
		c.line(6, 25, 17, 14, rgb(96, 64, 38)) // handle
		c.line(7, 26, 18, 15, wood)
		for i := 0; i < 6; i++ { // blade
			c.line(17+float64(i)*0.2, 14-float64(i)*0.1, 26, 5+float64(i)*0.2, steel)
		}
		c.line(18, 13, 27, 5, steelHi)
		c.rect(15, 15, 3, 3, rgb(80, 60, 40))
	case IAxe:
		c.line(8, 27, 22, 8, wood) // haft
		c.line(9, 27, 23, 8, woodHi)
		c.line(7, 27, 21, 8, rgb(90, 60, 34))
		c.ellipse(21, 9, 5.5, 4.5, steelLo) // head
		c.ellipse(20.5, 8.5, 4.5, 3.6, steel)
		c.ellipse(19.5, 7.5, 2.5, 1.8, steelHi)
		c.rect(24, 8, 4, 5, steel)
	case ISword:
		c.line(6, 26, 11, 21, wood)
		c.line(6, 27, 11, 22, rgb(90, 60, 34))
		c.rect(8, 19, 6, 3, rgb(226, 190, 80)) // crossguard
		for i := 0; i < 3; i++ {
			c.line(12+float64(i)*0.5, 20-float64(i)*0.3, 27, 4+float64(i)*0.5, steel)
		}
		c.line(13, 19, 27, 4, steelHi)
		c.disc(5, 27, 2, rgb(226, 190, 80))
	case IPistol:
		c.rect(6, 11, 19, 6, rgb(60, 64, 76)) // slide
		c.rect(6, 11, 19, 1, rgb(110, 116, 132))
		c.rect(24, 12, 4, 3, rgb(40, 44, 54)) // muzzle
		c.rect(8, 17, 6, 9, rgb(100, 70, 44)) // grip
		c.rect(8, 17, 6, 1, rgb(150, 110, 70))
		c.rect(14, 17, 6, 2, rgb(50, 54, 64)) // trigger guard
		c.rect(10, 9, 3, 2, rgb(40, 44, 54))  // sight
	case IRifle:
		c.rect(3, 15, 10, 5, rgb(124, 84, 48)) // stock
		c.rect(3, 15, 10, 1, rgb(166, 120, 72))
		c.rect(12, 14, 8, 4, rgb(60, 64, 76))  // receiver
		c.rect(19, 15, 11, 2, rgb(48, 52, 62)) // barrel
		c.rect(11, 18, 2, 5, rgb(40, 44, 54))  // magazine
		c.rect(14, 11, 6, 3, rgb(70, 76, 92))  // scope
		c.rect(14, 11, 6, 1, rgb(130, 140, 160))
	case IAmmo:
		for i, x := range []int{8, 14, 20} {
			c.rect(x, 13+i%2*2, 4, 10, rgb(220, 170, 70)) // brass case
			c.rect(x, 13+i%2*2, 1, 10, rgb(250, 214, 120))
			c.rect(x, 9+i%2*2, 4, 4, rgb(150, 100, 60)) // bullet
			c.rect(x+1, 7+i%2*2, 2, 2, rgb(150, 100, 60))
		}
	}
	c.outline(dark)
	c.shadow(17, 28, 10, 2.5, 45)
	return c.image()
}
