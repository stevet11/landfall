package main

import "fmt"

// selfTest runs the simulation without a window to sanity-check colonist AI.
func selfSetup(g *Game) {
	s := g.start
	g.applyTool(TChop, s.x-20, s.y-20, s.x+20, s.y+20)
	g.applyTool(TMine, s.x-20, s.y-20, s.x+20, s.y+20)
	g.applyTool(THarvest, s.x-20, s.y-20, s.x+20, s.y+20)
	g.applyTool(TZone, s.x-3, s.y-5, s.x+3, s.y-3)
	// a 7x5 room with a door, bed and heater
	rx, ry := s.x-12, s.y+3
	for y := ry - 1; y <= ry+5; y++ {
		for x := rx - 1; x <= rx+7; x++ {
			g.W.At(x, y).Obj, g.W.At(x, y).Ground = ONone, GGrass
		}
	}
	g.applyTool(TDoor, rx+3, ry+4, rx+3, ry+4)
	g.applyTool(TWall, rx, ry, rx+6, ry)
	g.applyTool(TWall, rx, ry+4, rx+6, ry+4)
	g.applyTool(TWall, rx, ry+1, rx, ry+3)
	g.applyTool(TWall, rx+6, ry+1, rx+6, ry+3)
	g.applyTool(TBed, rx+1, ry+1, rx+2, ry+1)
	g.applyTool(THeater, rx+5, ry+3, rx+5, ry+3)
	g.applyTool(TBed, rx+1, ry+3, rx+2, ry+3)
	g.applyTool(TFloor, rx+1, ry+1, rx+5, ry+3)
	g.applyTool(TRoof, rx+1, ry+1, rx+5, ry+3)
	g.applyTool(TFarm, s.x-8, s.y-8, s.x-3, s.y-6)
	g.applyTool(TFarm, rx+4, ry+1, rx+4, ry+2)
	// hunt every deer
	g.applyTool(THunt, 0, 0, MapW, MapH)
}

func selfTest(g *Game) {
	selfSetup(g)
	fmt.Println("start stock:", g.stock, "deer:", len(g.animals))
	for i := 0; i < int(60*DayLen*9); i++ {
		g.step(1.0 / 60)
		if i%(60*300) == 0 {
			fmt.Printf("t=%4ds %s %.0fC stock=%v cols=%d animals=%d\n", i/60, g.seasonName(), g.outT, g.stock, len(g.cols), len(g.animals))
			for _, c := range g.cols {
				fmt.Printf("   %-5s food=%3.0f rest=%3.0f hp=%3.0f T=%3.0f %s\n", c.Name, c.Hunger, c.Rest, c.Health, c.Temp, c.Status)
			}
		}
	}
	counts := map[Obj]int{}
	crops := 0
	for i := range g.W.T {
		t := &g.W.T[i]
		counts[t.Obj]++
		if t.Crop {
			crops++
		}
	}
	fmt.Println("walls", counts[OWall], "doors", counts[ODoor], "beds", counts[OBed], "heaters", counts[OHeater], "crops", crops, "rooms", len(g.rooms))
	for _, m := range g.msgs {
		fmt.Println("msg:", m.Text)
	}
}

// setupDemo builds a showcase colony for screenshots: ./landfall -new -demo prod|trade|build|night
func (g *Game) setupDemo(kind string) {
	s := g.start
	put := func(o Obj, dx, dy int) {
		t := g.W.At(s.x+dx, s.y+dy)
		t.Obj, t.Ground, t.Item, t.ItemN = o, GGrass, INone, 0
	}
	put(OBench, -4, -3)
	put(OBench, -3, -3)
	put(OFab, -2, -3)
	put(ODesk, -5, -3)
	put(OGen, 0, -4)
	put(OSolar, 1, -4)
	put(OSolar, 2, -4)
	put(OLamp, -1, 3)
	put(OLamp, 3, 3)
	put(OEHeater, -6, -1)
	put(OBed, 4, 2)
	g.addToStock(ICarving, 6)
	g.addToStock(ISculpture, 3)
	g.addToStock(IComponent, 7)
	g.addToStock(IGadget, 1)
	g.addToStock(IBook, 1)
	g.addToStock(ISilver, 135)
	g.addToStock(IMedicine, 12)
	g.prod[0].On, g.prod[2].On, g.prod[3].On = true, true, true
	g.buildSel = BGen
	g.slow(1)
	g.selected = g.cols[1].ID
	g.cols[1].Skills[SMedicine] = SkillState{Level: 6, XP: 120, Passion: true}
	switch kind {
	case "prod":
		g.panel = panelProd
		g.prod[1].On, g.prod[1].Urgent = true, true
	case "urgent": // orders marked urgent
		for dx := 0; dx < 4; dx++ {
			t := g.W.At(s.x-4+dx, s.y+4)
			t.Ground, t.Obj = GGrass, ONone
			t.Blue = BBed
			g.W.At(s.x-4+dx, s.y+6).Desig = DRoof
		}
		g.applyTool(TPriority, s.x-4, s.y+4, s.x-3, s.y+4)
		g.applyTool(TPriority, s.x-4, s.y+6, s.x-2, s.y+6)
	case "trade":
		g.nextTrader = 0
		g.spawnTrader()
		t := g.trader
		t.X, t.Y = float64(t.Spot.x)+0.5, float64(t.Spot.y)+0.5
		t.Arrived, t.Stay = true, 100
		g.panel = panelTrade
	case "banner": // trader waiting in the colony, no panel open
		g.nextTrader = 0
		g.spawnTrader()
		t := g.trader
		t.Beast = BeastCamel
		t.X, t.Y = float64(t.Spot.x)+0.5, float64(t.Spot.y)+0.5
		t.BX, t.BY = t.X+1.2, t.Y+0.3
		t.Arrived, t.Stay = true, 150
		g.alert("The trader has arrived! Press T (or click the banner) to trade.")
	case "arrow": // trader still far away
		g.nextTrader = 0
		g.spawnTrader()
		g.alert("A trader is approaching the colony! Click the banner or press T to find them.")
	case "walls": // wall and door materials side by side, with the material picker open
		for i, m := range []Material{MatWood, MatStone, MatIron} {
			for k := 0; k < 4; k++ {
				t := g.W.At(s.x-2+k, s.y+4+i*2)
				t.Obj, t.Mat, t.Ground = OWall, m, GGrass
			}
			d := g.W.At(s.x+3, s.y+4+i*2)
			d.Obj, d.Mat, d.Ground = ODoor, m, GGrass
		}
		g.W.At(s.x+6, s.y+5).Obj = OOre
		g.W.At(s.x+7, s.y+5).Obj = ORock
		put(OFurnace, 6, 3)
		g.tool, g.panel = TWall, panelMat
	case "fire": // a wooden shack on fire next to a heater
		for k := 0; k < 6; k++ {
			for _, dy := range []int{4, 8} {
				t := g.W.At(s.x-3+k, s.y+dy)
				t.Obj, t.Mat, t.Ground = OWall, MatWood, GGrass
			}
		}
		for dy := 5; dy <= 7; dy++ {
			for _, dx := range []int{-3, 2} {
				t := g.W.At(s.x+dx, s.y+dy)
				t.Obj, t.Mat, t.Ground = OWall, MatWood, GGrass
			}
		}
		put(OHeater, -2, 5)
		g.W.At(s.x-2, s.y+5).Fuel = 1e6
		put(OBed, 0, 6)
		g.W.At(s.x-3, s.y+4).Mat = MatStone
		g.W.At(s.x-3, s.y+8).Mat = MatIron
		g.ignite((s.y+4)*MapW+s.x-1, "")
		g.ignite((s.y+8)*MapW+s.x+2, "")
		g.W.At(s.x-1, s.y+4).Fire = 12
		g.W.At(s.x-2, s.y+4).Fire = 6
		g.W.At(s.x+2, s.y+8).Fire = 15
	case "roof": // a roofed house, an open stockpile with rotting food, and the Roof tool selected
		for dx := -3; dx <= 3; dx++ {
			for dy := 4; dy <= 8; dy++ {
				t := g.W.At(s.x+dx, s.y+dy)
				t.Ground, t.Obj = GGrass, ONone
				if dx == -3 || dx == 3 || dy == 4 || dy == 8 {
					t.Obj, t.Mat = OWall, MatStone
				} else if dx < 1 {
					t.Roof = true
				}
			}
		}
		g.W.At(s.x, s.y+8).Obj, g.W.At(s.x, s.y+8).Mat = ODoor, MatWood
		put(OHeater, -2, 5)
		g.W.At(s.x-2, s.y+5).Fuel = 1e6
		for i := 0; i < 4; i++ { // an open, unroofed stockpile
			t := g.W.At(s.x-8+i, s.y-2)
			t.Zone, t.Roof = true, false
			t.Item, t.ItemN = IFood, 30+i*5
		}
		g.W.At(s.x-8, s.y-2).Roof = true
		g.tool = TRoof
		g.W.At(s.x+1, s.y+5).Desig = DRoof
		g.W.At(s.x+1, s.y+6).Desig = DRoof
		g.W.At(s.x+2, s.y+5).Desig = DRoof
	case "weapons": // armed colonists, a weapon stockpile and wolves closing in
		for _, it := range []Item{IKnife, IAxe, ISword, IPistol, IRifle} {
			g.addToStock(it, 1)
		}
		g.addToStock(IAmmo, 60)
		g.addToStock(IIron, 14)
		g.cols[0].Weapon, g.cols[1].Weapon, g.cols[2].Weapon = IRifle, ISword, IPistol
		g.cols[0].Skills[SCombat].Level = 6
		g.nextRaid = 0
		g.time = DayLen * 0.5
		for i := 0; i < 3; i++ {
			g.spawnAnimal(AWolf, 9, 13)
		}
		g.selected = g.cols[0].ID
	case "crops": // every crop at every growth stage, a stove, and the crop picker
		for k := CropKind(0); k < numCrops; k++ {
			for st := 0; st < 4; st++ {
				t := g.W.At(s.x-8+st, s.y+3+int(k))
				t.Ground, t.Obj, t.Farm, t.CropKind, t.Crop = GGrass, ONone, true, k, true
				t.Grow = []float32{0.1, 0.45, 0.8, 1}[st]
			}
		}
		put(OStove, 0, 3)
		g.addToStock(IBread, 6)
		g.addToStock(IMeal, 4)
		g.addToStock(ICloth, 3)
		g.addToStock(IParka, 1)
		g.addToStock(IPickaxe, 1)
		g.addToStock(ITobacco, 5)
		g.cols[0].Tool, g.cols[0].Wear = IPickaxe, IParka
		g.cols[0].Skills[SCooking] = SkillState{Level: 5, XP: 40, Passion: true}
		g.selected = g.cols[0].ID
		g.tool, g.panel = TFarm, panelCrop
	case "raid": // armed raiders closing in on the colony
		g.cols[0].Weapon, g.cols[1].Weapon = ISword, IPistol
		g.addToStock(IAmmo, 40)
		g.time = DayLen * 6
		g.animals = nil
		g.nextRaiders = 0
		g.spawnRaid()
		g.selected = g.cols[0].ID
	case "move": // buildings waiting to be moved, one picked with the Move tool
		for dx := -6; dx <= 6; dx++ {
			for dy := 3; dy <= 8; dy++ {
				t := g.W.At(s.x+dx, s.y+dy)
				t.Ground, t.Obj = GGrass, ONone
			}
		}
		put(OFurnace, -4, 4)
		put(ODesk, -4, 6)
		put(OBed, 4, 4)
		g.orderMove(s.x-4, s.y+6, s.x+3, s.y+7) // the desk is already on its way
		g.tool = TMove
		g.moveSrc = (s.y+4)*MapW + s.x - 4 // the furnace is picked up with the tool
	case "sick": // an outbreak: flu, a critical fever and food poisoning
		g.cols[0].Ill = []Illness{{Kind: DisFlu, Sev: 0.34, Imm: 0.3, Tended: 2, Quality: 1.2}, {Kind: DisPoison, Sev: 0.2, Imm: 0.1}}
		g.cols[1].Ill = []Illness{{Kind: DisFever, Sev: 0.88, Imm: 0.3}}
		g.cols[2].Immune[DisFlu] = 500
		g.addToStock(IMedicine, 6)
		g.selected = g.cols[0].ID
		g.alert("Outbreak! Ada has come down with the flu. It spreads between people who stay close: put the sick to bed and tend them.")
	case "floors": // every floor kind side by side, with the picker open
		for k := FloorKind(0); k < numFloors; k++ {
			for dx := 0; dx < 4; dx++ {
				for dy := 0; dy < 3; dy++ {
					t := g.W.At(s.x-9+int(k)*5+dx, s.y+3+dy)
					t.Ground, t.Obj, t.Floor, t.FloorKind = GGrass, ONone, true, k
				}
			}
		}
		g.addToStock(ITile, 8)
		g.floorSel = FloorChecker
		g.tool, g.panel = TFloor, panelFloor
	case "joy": // games and a shrine, with the colonists in need of cheering up
		for dx := -2; dx <= 8; dx++ {
			for dy := 3; dy <= 6; dy++ {
				t := g.W.At(s.x+dx, s.y+dy)
				t.Ground, t.Obj = GGrass, ONone
			}
		}
		put(OHorse, 0, 4)
		put(OChecker, 3, 4)
		put(OShrine, 6, 4)
		g.cols[0].Joy, g.cols[1].Joy, g.cols[2].Joy = 18, 12, 30
		g.selected = g.cols[0].ID
	case "rain", "snow": // a spell of weather
		if kind == "snow" {
			g.time = DayLen * 7.6
		} else {
			g.time = DayLen * 1.4
		}
		g.wet, g.precip, g.weatherLeft = true, 1, 1e9
		g.slow(1)
	case "saves":
		g.togglePanel(panelSaves)
	case "build":
		g.tool, g.panel = TBuild, panelBuild
	case "nopower": // like a real player's mistake: lamps but no generator, and an unbuildable solar panel
		g.W.At(g.start.x+1, g.start.y-4).Obj = ONone
		g.W.At(g.start.x, g.start.y-4).Obj = ONone
		g.W.At(g.start.x+2, g.start.y-4).Obj = ONone
		g.W.At(g.start.x-2, g.start.y-3).Obj = ONone
		g.W.At(g.start.x+1, g.start.y-4).Blue = BSolar
		g.slow(1)
	case "night":
		g.time = DayLen * (22.0 / 24)
		g.slow(1)
	}
	g.showHelp = false
}
