package main

import "testing"

func TestParkasAreVisibleOnColonists(t *testing.T) {
	headlessArt = true
	defer func() { headlessArt = false; headlessSprites = nil }()
	fur := rgb(238, 238, 242)
	count := func(dir int, parka bool) (n int) {
		before := len(headlessSprites)
		paintColonist(colors[1], hairs[2], skins[1], dir, 0, parka)
		img := headlessSprites[before]
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				if img.NRGBAAt(x, y) == fur {
					n++
				}
			}
		}
		return
	}
	for dir := 0; dir < 4; dir++ {
		if count(dir, false) != 0 {
			t.Errorf("dir %d: nobody has a fur hood without a parka", dir)
		}
		if count(dir, true) < 8 {
			t.Errorf("dir %d: a parka should show a fur hood (%d fur pixels)", dir, count(dir, true))
		}
	}
	// and the sprite cache keeps worn and bare versions apart
	initArt()
	defer func() { art = nil }()
	c := &Colonist{ID: 3, Col: colors[2]}
	art.colonist(c, 0, 0)
	n := len(art.colonst)
	c.Wear = IParka
	art.colonist(c, 0, 0)
	if len(art.colonst) != n+1 {
		t.Fatal("a colonist in a parka is a different sprite from the same colonist without one")
	}
	art.colonist(c, 0, 0)
	if len(art.colonst) != n+1 {
		t.Fatal("and it is cached")
	}
}

func TestSparkParkasAreWornNotHoarded(t *testing.T) {
	// regression: colonists in the stockpile's reach put on a spare parka
	g := quiet(7)
	for _, c := range g.cols {
		c.Wear = INone
	}
	g.addToStock(IParka, 3)
	run(g, 60)
	worn := 0
	for _, c := range g.cols {
		if c.Wear == IParka {
			worn++
		}
	}
	if worn != 3 {
		t.Fatalf("all three should be worn, %d are", worn)
	}
}
