package main

import "github.com/hajimehoshi/ebiten/v2"

// Pack animals that travel with the trader.

func paintDonkey(frame int) *ebiten.Image {
	c := newCanvas()
	body, belly, leg := rgb(138, 128, 118), rgb(200, 190, 176), rgb(88, 80, 72)
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
	c.ellipse(27, 10, 4, 3, body)
	c.rect(29, 10, 3, 3, belly) // muzzle
	c.set(27, 9, dark)
	c.rect(24, 4, 2, 5, body) // long ears
	c.rect(27, 4, 2, 5, body)
	c.rect(24, 4, 2, 1, rgb(60, 54, 50))
	c.rect(27, 4, 2, 1, rgb(60, 54, 50))
	c.line(5, 14, 3, 19, rgb(70, 64, 58)) // tail
	// the load: two crates and a red bundle strapped on
	c.rect(8, 9, 6, 6, rgb(170, 122, 70))
	c.rect(8, 9, 6, 1, rgb(206, 160, 100))
	c.rect(15, 10, 5, 5, rgb(190, 60, 60))
	c.rect(15, 10, 5, 1, rgb(230, 100, 90))
	c.rect(7, 13, 14, 1, rgb(60, 44, 34))
	c.outline(dark)
	c.shadow(14, 27, 11, 3, 55)
	return c.image()
}

func paintCamel(frame int) *ebiten.Image {
	c := newCanvas()
	body, belly, leg := rgb(196, 158, 100), rgb(228, 202, 150), rgb(130, 100, 62)
	o := []int{-1, 1}[frame]
	for i, x := range []int{7, 10, 18, 21} {
		d := o
		if i%2 == 1 {
			d = -o
		}
		c.rect(x+d, 18, 2, 9, leg)
	}
	c.ellipse(14, 17, 10, 5, body)
	c.ellipse(13, 20, 7, 1.8, belly)
	c.disc(10, 11, 3.6, body) // humps
	c.disc(17, 11, 3.6, body)
	c.line(22, 15, 26, 7, body)
	c.line(23, 16, 27, 8, body)
	c.ellipse(28, 7, 3.6, 2.6, body)
	c.rect(30, 7, 2, 3, belly)
	c.set(28, 6, dark)
	c.line(4, 15, 3, 20, leg)
	// blanket and loads on the humps
	c.rect(8, 8, 11, 3, rgb(200, 60, 80))
	c.rect(8, 8, 11, 1, rgb(240, 110, 120))
	c.rect(7, 3, 5, 5, rgb(170, 122, 70))
	c.rect(7, 3, 5, 1, rgb(206, 160, 100))
	c.rect(14, 4, 5, 4, rgb(60, 120, 170))
	c.rect(14, 4, 5, 1, rgb(100, 160, 210))
	c.outline(dark)
	c.shadow(14, 27, 11, 3, 55)
	return c.image()
}
