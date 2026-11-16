package main

import "testing"

func rainy(g *Game) {
	g.wet, g.precip, g.weatherLeft = true, 1, 1e9
	g.time = DayLen * 0.3 // a mild morning
}

func TestWeatherComesAndGoes(t *testing.T) {
	g := newGame(7)
	g.animals = nil
	wets, dry := 0, 0
	for i := 0; i < int(DayLen*6); i++ {
		g.weatherStep(1)
		if g.precipitating() {
			wets++
		} else {
			dry++
		}
	}
	if wets == 0 || dry == 0 || wets > dry {
		t.Fatalf("expected mostly dry with some rain: wet %d dry %d", wets, dry)
	}
}

func TestRainOrSnowByTemperature(t *testing.T) {
	g := quiet(7)
	rainy(g)
	g.outT = 12
	if !g.raining() || g.snowing() || g.weatherName() != "Rain" {
		t.Fatal("warm precipitation is rain")
	}
	g.outT = -5
	if g.raining() || !g.snowing() || g.weatherName() != "Snow" {
		t.Fatal("cold precipitation is snow")
	}
	g.wet, g.precip = false, 0
	if g.weatherName() != "" {
		t.Fatal("dry")
	}
}

func TestRainWaterCropsAndSlowsWalkingOutdoorsOnly(t *testing.T) {
	g := quiet(7)
	grow := func(roof bool) float32 {
		x, y := g.start.x+3, g.start.y+3
		tl := g.W.At(x, y)
		tl.Farm, tl.Crop, tl.CropKind, tl.Grow, tl.Roof = true, true, CropVeg, 0, roof
		g.outT = 14
		g.wet, g.precip = true, 1
		g.slow(10)
		return tl.Grow
	}
	rainy(g)
	open := grow(false)
	if open <= 0 {
		t.Fatal("crops should grow")
	}
	g2 := quiet(7)
	rainy(g2)
	g2.wet, g2.precip = false, 0
	x, y := g2.start.x+3, g2.start.y+3
	tl := g2.W.At(x, y)
	tl.Farm, tl.Crop, tl.CropKind, tl.Grow = true, true, CropVeg, 0
	g2.outT = 14
	g2.slow(10)
	if open < tl.Grow*1.4 {
		t.Fatalf("rain should speed outdoor growth: %.4f vs dry %.4f", open, tl.Grow)
	}
	if g.weatherWalk(x, y) >= 1 {
		t.Fatal("rain slows walking")
	}
	g.W.At(x, y).Roof = true
	if g.weatherWalk(x, y) != 1 || g.wetChill(x, y) != 0 {
		t.Fatal("under a roof nobody gets wet")
	}
}

func TestRainChillsAndPutsOutFiresAndDimsSolar(t *testing.T) {
	g := quiet(7)
	g.noFireFight = true
	rainy(g)
	g.outT = 12
	x, y := g.start.x+4, g.start.y+4
	if g.wetChill(x, y) != rainChill {
		t.Fatal("rain chills")
	}
	tl := g.W.At(x, y)
	tl.Obj, tl.Mat = OWall, MatWood
	g.ignite(y*MapW+x, "")
	for i := 0; i < 400 && tl.Fire > 0; i++ {
		g.updateFire(0.5)
	}
	if tl.Fire != 0 {
		t.Fatal("rain should put out an exposed fire")
	}
	place(g, OSolar, 8, 8)
	g.time = DayLen * 0.5 // noon
	g.wet, g.precip = false, 0
	g.updatePower(1)
	dry := g.powerSupply
	rainy(g)
	g.time = DayLen * 0.5
	g.updatePower(1)
	if dry <= 0 || g.powerSupply > dry*0.5 {
		t.Fatalf("clouds cut solar: %.0f vs %.0f", g.powerSupply, dry)
	}
}

func TestWeatherBalanceStillSurvivable(t *testing.T) {
	// precipitation is a mild penalty, not a death sentence: a colonist in the rain does not freeze in summer
	g := quiet(7)
	rainy(g)
	g.outT = 20
	x, y := g.start.x, g.start.y
	if g.tempAt(x, y)-g.wetChill(x, y) < 8 {
		t.Fatal("summer rain is chilly, not freezing")
	}
}
