package main

import (
	"fmt"
	"math"
)

// Raiders: armed bandits who attack the colony. They come with the weapons of the day (knives and
// axes early, swords, pistols and rifles later), go for the nearest colonist, steal from the
// stockpile if nobody can be reached, and run when the raid goes badly. Killed raiders drop what
// they carried. They are Animals of kind ARaider so combat reuses the wolf and hunting code.

const (
	goalAttack = iota
	goalSteal
	goalFlee
)

// raidInfo tracks the raid in progress.
type raidInfo struct {
	Start  int
	Origin pt
	Broken bool // the survivors have lost heart
}

func (a *Animal) hostile() bool { return a.Kind == AWolf || a.Kind == ARaider }

// raiderDamageScale makes raiders hit a little softer than the same weapon in a colonist's hand.
const raiderDamageScale = 0.42

func compassWord(dx, dy float64) string {
	ang := math.Atan2(-dy, dx) * 180 / math.Pi // screen y points down: flip so north is up
	names := []string{"east", "north-east", "north", "north-west", "west", "south-west", "south", "south-east"}
	return names[int(math.Mod(math.Round(ang/45)+8, 8))]
}

// raiderWeapon picks a weapon that suits how far into the game we are.
func (g *Game) raiderWeapon(day float64) Item {
	r := g.rng.Float64()
	switch {
	case day < 5:
		if r < 0.5 {
			return IKnife
		}
		return IAxe
	case day < 9:
		switch {
		case r < 0.25:
			return IKnife
		case r < 0.45:
			return IAxe
		case r < 0.75:
			return ISword
		}
		return IPistol
	}
	switch {
	case r < 0.25:
		return ISword
	case r < 0.35:
		return IAxe
	case r < 0.70:
		return IPistol
	}
	return IRifle
}

// raidSize is how many raiders turn up.
func (g *Game) raidSize() int {
	return max(2, min(7, 1+len(g.cols)/4+int(g.time/DayLen)/6))
}

func (g *Game) spawnRaid() bool {
	ox, oy, ok := g.landTile(50, 62)
	if !ok {
		return false
	}
	day := g.time / DayLen
	n := g.raidSize()
	spawned := 0
	for i := 0; i < n*12 && spawned < n; i++ {
		x, y := ox+g.rng.Intn(7)-3, oy+g.rng.Intn(7)-3
		if !g.W.Walkable(x, y) || !g.onMainland(x, y) {
			continue
		}
		hp := math.Min(72, 48+1.5*math.Floor(day))
		g.nextID++
		g.animals = append(g.animals, &Animal{ID: g.nextID, Kind: ARaider, X: float64(x) + 0.5, Y: float64(y) + 0.5,
			HP: hp, MaxHP: hp, DX: 1, Weapon: g.raiderWeapon(day), Exit: pt{ox, oy}})
		spawned++
	}
	if spawned == 0 {
		return false
	}
	g.raid = &raidInfo{Start: spawned, Origin: pt{ox, oy}}
	g.stats.RaidersSeen += spawned
	g.alert("Raiders! %d armed men are coming from the %s!", spawned, compassWord(float64(ox-g.start.x), float64(oy-g.start.y)))
	g.emit("horn", -1, -1)
	return true
}

func (g *Game) liveRaiders() []*Animal {
	var r []*Animal
	for _, a := range g.animals {
		if a.Kind == ARaider && !a.Dead {
			r = append(r, a)
		}
	}
	return r
}

// updateRaidState ends the raid when nobody is left and breaks morale when it goes badly.
func (g *Game) updateRaidState() {
	if g.raid == nil {
		return
	}
	live := g.liveRaiders()
	if len(live) == 0 {
		g.raid = nil
		g.msg("The raid is over.")
		return
	}
	if !g.raid.Broken && float64(len(live)) <= 0.4*float64(g.raid.Start) {
		g.raid.Broken = true
		g.alert("The surviving raiders are fleeing!")
	}
}

func (g *Game) raidersBroken() bool { return g.raid != nil && g.raid.Broken }

// raidStatus describes the raid for the banner.
func (g *Game) raidStatus() (msg string, ok bool) {
	live := g.liveRaiders()
	if len(live) == 0 {
		return "", false
	}
	fleeing, stealing := 0, 0
	for _, a := range live {
		switch a.Goal {
		case goalFlee:
			fleeing++
		case goalSteal:
			stealing++
		}
	}
	switch {
	case stealing > 0:
		return "RAIDERS are stealing from the stockpile! (click to look)", true
	case fleeing == len(live):
		return fmt.Sprintf("Raiders retreating (%d left)", len(live)), true
	}
	return fmt.Sprintf("RAIDERS! %d armed men attacking (click to look)", len(live)), true
}

// raiderFocus is the raider closest to the colony: what the banner points at.
func (g *Game) raiderFocus() *Animal {
	var best *Animal
	bd := 1e9
	for _, a := range g.liveRaiders() {
		if d := math.Hypot(a.X-float64(g.start.x), a.Y-float64(g.start.y)); d < bd {
			bd, best = d, a
		}
	}
	return best
}

// hitColonist applies a raider's blow or bullet.
func (g *Game) raiderHit(a *Animal, c *Colonist, dmg float64, ranged bool) {
	c.Health -= dmg * c.damageTakenMult()
	kind := "Cut"
	if ranged {
		kind = "Gunshot"
	}
	g.addWound(c, kind, dmg/5, true)
	g.emit("hit", c.X, c.Y)
	g.stats.RaiderHits++
}

// updateRaider runs one raider for one tick and returns how fast it moves.
func (g *Game) updateRaider(a *Animal, dt float64) float64 {
	x, y := int(a.X), int(a.Y)
	wd := weaponDefs[a.Weapon]
	if a.Goal != goalFlee && (a.HP < a.MaxHP*0.3 || g.raidersBroken()) {
		a.Goal, a.Path = goalFlee, nil
	}
	switch a.Goal {
	case goalFlee:
		if math.Hypot(a.X-float64(a.Exit.x)-0.5, a.Y-float64(a.Exit.y)-0.5) < 2.5 {
			a.Dead = true // got away
			if a.LootN > 0 {
				g.msg("A raider got away with %d %s.", a.LootN, itemNames[a.Loot])
				g.stats.Stolen += a.LootN
			}
			return 3.0
		}
		if a.Think <= 0 && len(a.Path) == 0 {
			a.Think = 1
			ex, ey := a.Exit.x, a.Exit.y
			if p, ok := g.W.bfs(x, y, 120, func(p, q int) bool { return p == ex && q == ey }); ok {
				a.Path = p
			} else {
				a.Dead = true
			}
		}
		return 3.2

	case goalSteal:
		tgt := g.nearestLoot(a)
		if tgt < 0 {
			a.Goal = goalFlee
			return 2.9
		}
		tx, ty := tgt%MapW, tgt/MapW
		if iabs(int(a.X)-tx) <= 1 && iabs(int(a.Y)-ty) <= 1 { // standing next to it, as the path asks
			t := &g.W.T[tgt]
			k := min(t.ItemN, 15)
			a.Loot, a.LootN = t.Item, k
			t.ItemN -= k
			if t.ItemN == 0 {
				t.Item = INone
			}
			g.recompute()
			g.alert("A raider stole %d %s from the stockpile!", k, itemNames[a.Loot])
			a.Goal, a.Path = goalFlee, nil
			return 2.9
		}
		if a.Think <= 0 {
			a.Think = 0.6
			if p, ok := g.W.bfs(x, y, 90, func(p, q int) bool { return iabs(p-tx) <= 1 && iabs(q-ty) <= 1 }); ok {
				a.Path = p
			} else {
				a.Goal = goalFlee
			}
		}
		return 2.9
	}

	// attack
	c, d := g.nearestColonist(a, 120)
	if c == nil {
		a.Goal = goalFlee
		return 2.9
	}
	ranged := !wd.Melee
	reach := 1.1
	if ranged {
		reach = wd.Range * 0.85
	}
	if d <= reach && (!ranged || g.clearShot(a.X, a.Y, c.X, c.Y)) {
		a.Path = nil
		if a.Cool <= 0 {
			a.Cool = wd.Cool
			dmg := wd.Dmg * raiderDamageScale
			if ranged {
				g.shots = append(g.shots, Shot{a.X, a.Y, c.X, c.Y, 0.12})
				g.emit(map[Item]string{IPistol: "shot", IRifle: "rifle"}[a.Weapon], a.X, a.Y)
				p := math.Max(0.1, math.Min(0.9, 0.55+wd.Acc-0.02*d))
				if g.chance(p) {
					g.raiderHit(a, c, dmg, true)
				}
			} else {
				g.emit("slash", a.X, a.Y)
				g.raiderHit(a, c, dmg, false)
			}
		}
		return 0
	}
	if a.Think <= 0 {
		a.Think = 0.5
		tx, ty := int(c.X), int(c.Y)
		if p, ok := g.W.bfs(x, y, 90, func(p, q int) bool { return p == tx && q == ty }); ok {
			a.Path, a.Stuck = p, 0
		} else {
			a.Path = nil
			a.Stuck += 0.5
			if a.Stuck > 4 { // nobody can be reached (walled in?): go for the stockpile instead
				a.Goal = goalSteal
			}
		}
	}
	return 2.9
}

// nearestLoot picks the stockpile tile a thief likes best: valuable goods that aren't too far.
func (g *Game) nearestLoot(a *Animal) int {
	best, bs := -1, 0.0
	for _, i := range g.zoneList {
		t := &g.W.T[i]
		if t.Item == INone || t.ItemN == 0 {
			continue
		}
		dist := math.Hypot(a.X-float64(i%MapW), a.Y-float64(i/MapW))
		if s := math.Max(itemValue[t.Item], 1) * float64(min(t.ItemN, 15)) / (10 + dist); s > bs {
			bs, best = s, i
		}
	}
	return best
}

// raiderDown is what happens when a colonist kills a raider: the loot and weapon drop.
func (g *Game) raiderDown(c *Colonist, a *Animal) {
	a.Dead = true
	g.stats.RaidersKilled++
	g.emit("death", a.X, a.Y)
	g.msg("%s killed a raider.", c.Name)
	x, y := int(a.X), int(a.Y)
	if a.LootN > 0 {
		g.placeItem(x, y, a.Loot, a.LootN)
	}
	if g.chance(0.6) {
		g.placeItem(x, y, a.Weapon, 1)
	}
	g.placeItem(x, y, ISilver, 5+g.rng.Intn(36))
}
