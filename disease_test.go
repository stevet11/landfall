package main

import (
	"math"
	"testing"
)

// lone returns a game with a single colonist (so nothing else interferes).
func lone(seed int64) (*Game, *Colonist) {
	g := quiet(seed)
	for _, o := range g.cols[1:] {
		o.Dead = true
	}
	g.cols = g.cols[:1]
	c := g.cols[0]
	c.Traits = nil
	c.Hunger, c.Health = 100, 100
	return g, c
}

// course runs an illness for up to maxSecs in steady conditions and reports how it ended.
func course(kind Disease, tended int, quality float64, rest int, temp, hunger float64, maxSecs int) (recovered, died bool, peak float64) {
	g, c := lone(7)
	c.Ill = []Illness{{Kind: kind, Tended: tended, Quality: quality}}
	for i := 0; i < maxSecs*10; i++ {
		c.Temp, c.Hunger = temp, hunger
		switch rest {
		case 2:
			c.Job = Job{Kind: JRecover, Stage: 0}
		case 1:
			c.Job = Job{Kind: JRecover, Stage: 1}
		default:
			c.Job = Job{}
		}
		c.Path = nil
		g.updateIllness(c, 0.1)
		if c.sick() {
			peak = math.Max(peak, c.sickSeverity())
		}
		if c.Health <= 0 || c.sickSeverity() >= 1 && c.Health < 60 {
			return false, true, peak
		}
		if !c.sick() {
			return true, false, peak
		}
	}
	return false, false, peak
}

func TestFluRunsItsCourseAndGivesImmunity(t *testing.T) {
	g, c := lone(7)
	if !g.infect(c, DisFlu) || !c.hasIll(DisFlu) {
		t.Fatal("infect")
	}
	if g.infect(c, DisFlu) {
		t.Fatal("you can't catch what you already have")
	}
	for i := 0; i < 60*400; i++ {
		c.Temp, c.Hunger = 18, 100
		g.updateIllness(c, 1.0/60)
		if !c.sick() {
			break
		}
	}
	if c.sick() {
		t.Fatalf("a flu in good conditions burns out within minutes: sev %.2f imm %.2f", c.sickSeverity(), c.Ill[0].Imm)
	}
	if c.Immune[DisFlu] <= 0 {
		t.Fatal("recovery brings immunity")
	}
	if g.infect(c, DisFlu) {
		t.Fatal("immune")
	}
	if g.stats.Recovered != 1 || g.stats.Illnesses != 1 {
		t.Fatalf("stats %+v", g.stats)
	}
	// immunity wears off
	for i := 0; i < int(diseaseDefs[DisFlu].Immunity)+5; i++ {
		g.updateIllness(c, 1)
	}
	if !g.infect(c, DisFlu) {
		t.Fatal("after a couple of days you can catch it again")
	}
}

func TestFluIsMildButCareAndComfortMatter(t *testing.T) {
	ok, died, peak := course(DisFlu, 0, 1, 0, 18, 100, 900)
	if !ok || died || peak > 0.6 {
		t.Fatalf("a healthy colonist shrugs off the flu: recovered=%v died=%v peak=%.2f", ok, died, peak)
	}
	_, _, cold := course(DisFlu, 0, 1, 0, -5, 20, 900)
	if cold <= peak+0.15 {
		t.Fatalf("cold and hunger make it much worse: %.2f vs %.2f", cold, peak)
	}
	_, _, bed := course(DisFlu, 0, 1, 2, 18, 100, 900)
	if bed >= peak {
		t.Fatalf("resting in bed helps: %.2f vs %.2f", bed, peak)
	}
}

func TestMarshFeverKillsTheNeglected(t *testing.T) {
	// working through it, untreated, in a cold camp: deadly
	_, died, _ := course(DisFever, 0, 1, 0, 3, 60, 1200)
	if !died {
		t.Fatal("untreated marsh fever without rest should kill")
	}
	// the three things that save people: a bed, a nurse, medicine
	bedOK, bedDied, _ := course(DisFever, 0, 1, 2, 18, 100, 1200)
	if !bedOK || bedDied {
		t.Fatal("a warm bed alone gets most people through")
	}
	groundOK, _, groundPeak := course(DisFever, 0, 1, 1, 18, 100, 1200)
	_, _, bedPeak := course(DisFever, 0, 1, 2, 18, 100, 1200)
	if groundPeak <= bedPeak {
		t.Fatalf("the ground is worse than a bed: %.2f vs %.2f (recovered %v)", groundPeak, bedPeak, groundOK)
	}
	medOK, medDied, medPeak := course(DisFever, 2, 1.4, 2, 18, 100, 1200)
	if !medOK || medDied || medPeak > 0.35 {
		t.Fatalf("medicine in a bed nearly stops it: peak %.2f", medPeak)
	}
	_, _, noviceMed := course(DisFever, 2, 1.0, 2, 18, 100, 1200)
	if medPeak >= noviceMed {
		t.Fatalf("a skilled medic does better: %.2f vs %.2f", medPeak, noviceMed)
	}
}

func TestCriticalIllnessDamagesHealthAndSicknessCanKill(t *testing.T) {
	g, c := lone(7)
	c.Ill = []Illness{{Kind: DisFever, Sev: 0.9}}
	h0 := c.Health
	for i := 0; i < 60*10; i++ {
		g.updateIllness(c, 1.0/60)
	}
	if c.Health >= h0 {
		t.Fatal("a critical illness hurts")
	}
	// below the critical line it does no direct damage
	g2, d := lone(7)
	d.Ill = []Illness{{Kind: DisFlu, Sev: 0.5}}
	for i := 0; i < 60*10; i++ {
		d.Temp = 18
		g2.updateIllness(d, 1.0/60)
	}
	if d.Health < 100 {
		t.Fatalf("a moderate illness does not damage health: %.1f", d.Health)
	}
	// and it can kill: the cause is named
	g3, e := lone(7)
	e.Ill = []Illness{{Kind: DisFever, Sev: 1}}
	e.Health = 0.5
	e.Job = Job{}
	g3.msgs = nil
	run(g3, 3)
	if !e.Dead {
		t.Fatal("should have died")
	}
	if g3.stats.IllDeaths != 1 {
		t.Fatalf("stats %+v", g3.stats)
	}
	named := false
	for _, m := range g3.msgs {
		if m.Alert && contains(m.Text, "died of marsh fever") {
			named = true
		}
	}
	if !named {
		t.Fatalf("the death should say what killed them: %v", g3.msgs)
	}
}

func TestSickColonistsWorkAndWalkSlower(t *testing.T) {
	well, ill := &Colonist{}, &Colonist{Ill: []Illness{{Kind: DisFlu, Sev: 0.8}}}
	if ill.illWork() >= well.illWork() || ill.illSpeed() >= well.illSpeed() {
		t.Fatal("fever slows people")
	}
	if well.illWork() != 1 {
		t.Fatal("healthy people are unaffected")
	}
	chop := func(sev float64) float64 {
		g, c := lone(7)
		if sev > 0 {
			c.Ill = []Illness{{Kind: DisFlu, Sev: sev, Tended: 1}}
		}
		tree := place(g, OTree, 3, 3)
		g.applyTool(TChop, tree.x, tree.y, tree.x, tree.y)
		for i := 0; i < 60*60; i++ {
			c.Ill = c.Ill[:min(len(c.Ill), 1)]
			if len(c.Ill) == 1 {
				c.Ill[0].Sev, c.Ill[0].Imm = sev, 0 // hold the illness steady
			}
			g.step(1.0 / 60)
			if g.W.At(tree.x, tree.y).Obj != OTree {
				return float64(i) / 60
			}
		}
		return 60
	}
	if chop(0.8) <= chop(0)*1.2 {
		t.Fatal("a sick colonist takes noticeably longer to chop a tree")
	}
}

func TestNoNaturalHealingWhileSick(t *testing.T) {
	g, c := lone(7)
	c.Health = 70
	c.Ill = []Illness{{Kind: DisFlu, Sev: 0.2, Tended: 1}}
	c.Temp = 18
	for i := 0; i < 60*20; i++ {
		c.Ill[0].Sev, c.Ill[0].Imm = 0.2, 0
		g.step(1.0 / 60)
	}
	if c.Health > 70.5 {
		t.Fatalf("you don't heal on your own while ill: %.1f", c.Health)
	}
}

func TestContagionNeedsCloseContact(t *testing.T) {
	spread := func(sx, sy, hx, hy float64, immune bool, kind Disease) bool {
		g := quiet(7)
		openField(g, -6, 30, -6, 30)
		a, b := g.cols[0], g.cols[1]
		for _, o := range g.cols[2:] {
			o.Dead = true
		}
		g.cols = g.cols[:2]
		a.X, a.Y = float64(g.start.x)+sx, float64(g.start.y)+sy+10
		b.X, b.Y = float64(g.start.x)+hx, float64(g.start.y)+hy+10
		a.Ill = []Illness{{Kind: kind, Sev: 0.3, Imm: 0.1}}
		if immune {
			b.Immune[kind] = 1000
		}
		for i := 0; i < 60*60; i++ {
			a.Ill[0].Imm = 0.1 // keep it alive for the whole test
			g.spreadDisease(0.25)
			if b.hasIll(kind) {
				return true
			}
		}
		return false
	}
	if !spread(0, 0, 1, 0, false, DisFlu) {
		t.Fatal("standing next to someone with the flu for a minute: you catch it")
	}
	if spread(0, 0, 9, 0, false, DisFlu) {
		t.Fatal("distance protects")
	}
	if spread(0, 0, 1, 0, true, DisFlu) {
		t.Fatal("the immune don't catch it")
	}
	if spread(0, 0, 1, 0, false, DisPoison) {
		t.Fatal("food poisoning is not contagious")
	}
	if !spread(0, 0, 1.5, 0, false, DisFever) {
		t.Fatal("marsh fever spreads too")
	}
}

func TestSharedRoomsSpreadItFasterAndFurther(t *testing.T) {
	// time to infection averaged over many tries, same room vs outdoors
	avg := func(indoors bool) float64 {
		total, n := 0.0, 0
		for seed := int64(1); seed <= 12; seed++ {
			g := quiet(seed)
			openField(g, -8, 30, -8, 30)
			var hx, hy int
			if indoors {
				in := buildHouse(g, true)
				hx, hy = in.x, in.y
			} else {
				hx, hy = g.start.x+10, g.start.y+10
			}
			a, b := g.cols[0], g.cols[1]
			for _, o := range g.cols[2:] {
				o.Dead = true
			}
			g.cols = g.cols[:2]
			a.X, a.Y = float64(hx)+0.5, float64(hy)+0.5
			b.X, b.Y = float64(hx)+1.5, float64(hy)+0.5
			a.Ill = []Illness{{Kind: DisFlu, Sev: 0.3, Imm: 0.1}}
			for i := 0; i < 60*400; i++ {
				a.Ill[0].Imm = 0.1
				g.spreadDisease(0.25)
				if b.hasIll(DisFlu) {
					total += float64(i) * 0.25
					n++
					break
				}
			}
		}
		if n == 0 {
			return 1e9
		}
		return total / float64(n)
	}
	in, out := avg(true), avg(false)
	if in >= out {
		t.Fatalf("crowded rooms spread it faster: %.0f s indoors vs %.0f s outdoors", in, out)
	}
}

func TestTendingTheSick(t *testing.T) {
	g := quiet(7)
	g.addToStock(IMedicine, 3)
	patient := g.cols[0]
	patient.Ill = []Illness{{Kind: DisFever, Sev: 0.3, Imm: 0.05}}
	if !g.needsTend(patient) {
		t.Fatal("a sick colonist needs a medic")
	}
	for _, c := range g.cols {
		c.Skills[SMedicine].Level = 6
	}
	for i := 0; i < 60*40 && patient.Ill[0].Tended < 2; i++ {
		patient.Ill[0].Imm = 0.05
		patient.Ill[0].Sev = 0.3
		g.step(1.0 / 60)
	}
	if patient.Ill[0].Tended != 2 || patient.Ill[0].Quality < 1.3 {
		t.Fatalf("medicine should have been used, skill-adjusted: %+v", patient.Ill[0])
	}
	g.recompute()
	if g.stock[IMedicine] != 2 {
		t.Fatalf("one medicine spent, %d left", g.stock[IMedicine])
	}
	if g.needsTend(patient) {
		t.Fatal("already on medicine")
	}
}

func TestNursingWhenThereIsNoMedicine(t *testing.T) {
	g := quiet(7)
	for g.stock[IMedicine] > 0 {
		g.takeStock(IMedicine, g.stock[IMedicine])
		g.recompute()
	}
	patient := g.cols[0]
	patient.Ill = []Illness{{Kind: DisFlu, Sev: 0.2, Imm: 0.05}}
	for i := 0; i < 60*40 && patient.Ill[0].Tended == 0; i++ {
		patient.Ill[0].Imm, patient.Ill[0].Sev = 0.05, 0.2
		g.step(1.0 / 60)
	}
	if patient.Ill[0].Tended != 1 {
		t.Fatal("without medicine you still get basic nursing")
	}
	if g.needsTend(patient) {
		t.Fatal("nothing more to do without medicine")
	}
	// and a mild early case isn't worth a medic's time
	healthy := g.cols[1]
	healthy.Ill = []Illness{{Kind: DisFlu, Sev: 0.02}}
	if g.illNeedsCare(healthy) {
		t.Fatal("a sniffle can wait")
	}
}

func TestNursesCatchWhatTheyTend(t *testing.T) {
	caught, trials := 0, 300
	for i := 0; i < trials; i++ {
		g := quiet(int64(i + 1))
		doc, p := g.cols[0], g.cols[1]
		doc.Skills[SMedicine].Level = 0
		p.Ill = []Illness{{Kind: DisFever, Sev: 0.4, Imm: 0.1}}
		g.exposeDoctor(doc, p)
		if doc.hasIll(DisFever) {
			caught++
		}
	}
	if caught < trials/8 || caught > trials/2 {
		t.Fatalf("an unskilled nurse catches it about a quarter of the time: %d of %d", caught, trials)
	}
	skilled := 0
	for i := 0; i < trials; i++ {
		g := quiet(int64(i + 1))
		doc, p := g.cols[0], g.cols[1]
		doc.Skills[SMedicine].Level = 14
		p.Ill = []Illness{{Kind: DisFever, Sev: 0.4, Imm: 0.1}}
		g.exposeDoctor(doc, p)
		if doc.hasIll(DisFever) {
			skilled++
		}
	}
	if skilled >= caught {
		t.Fatalf("skill protects: %d vs %d", skilled, caught)
	}
}

func TestSickColonistsGoToBedAndStayThere(t *testing.T) {
	g := quiet(7)
	for _, c := range g.cols[1:] {
		c.Dead = true
	}
	g.cols = g.cols[:1]
	c := g.cols[0]
	bed := place(g, OBed, 3, 3)
	_ = bed
	c.Ill = []Illness{{Kind: DisFever, Sev: 0.5, Imm: 0.1, Tended: 1}}
	for i := 0; i < 60*30; i++ {
		c.Ill[0].Imm = 0.1
		c.Ill[0].Sev = 0.5
		g.step(1.0 / 60)
		if c.Job.Kind == JRecover && len(c.Path) == 0 {
			break
		}
	}
	if c.Job.Kind != JRecover {
		t.Fatalf("a badly sick colonist should lie down (job %d, %s)", c.Job.Kind, c.Status)
	}
	if c.Job.Stage != 0 {
		t.Fatal("in the bed, not on the floor")
	}
	// they stay until well, even though their health is fine
	for i := 0; i < 60*30; i++ {
		c.Ill[0].Imm = 0.1
		g.step(1.0 / 60)
	}
	if c.Job.Kind != JRecover {
		t.Fatal("stay in bed until recovered")
	}
	c.Ill = nil
	run(g, 5)
	if c.Job.Kind == JRecover {
		t.Fatal("once better they get up")
	}
}

func TestOutbreaksStartWithAnAlertAndPatientZero(t *testing.T) {
	g := quiet(7)
	g.time = DayLen * 5
	g.nextOutbreak = g.time
	g.msgs = nil
	g.updateOutbreaks()
	sick := 0
	for _, c := range g.cols {
		if c.sick() {
			sick++
		}
	}
	if sick != 1 {
		t.Fatalf("exactly one patient zero, got %d", sick)
	}
	alert := false
	for _, m := range g.msgs {
		if m.Alert && contains(m.Text, "Outbreak!") {
			alert = true
		}
	}
	if !alert {
		t.Fatalf("no alert: %v", g.msgs)
	}
	if g.nextOutbreak < g.time+DayLen*3 || g.stats.Outbreaks != 1 {
		t.Fatalf("the next outbreak is days away: %.0f s", g.nextOutbreak-g.time)
	}
	// the immune are skipped
	h := quiet(7)
	h.nextOutbreak = h.time
	for _, c := range h.cols {
		for d := range c.Immune {
			c.Immune[d] = 1e6
		}
	}
	h.updateOutbreaks()
	for _, c := range h.cols {
		if c.sick() {
			t.Fatal("immune colonists cannot be infected")
		}
	}
}

func TestOutbreakKindsFollowTheSeasonAndGameStage(t *testing.T) {
	count := func(day float64, season string) (flu, fever int) {
		g := quiet(7)
		g.time = DayLen * day
		for i := 0; i < 400; i++ {
			switch g.pickOutbreakIn(season) {
			case DisFlu:
				flu++
			case DisFever:
				fever++
			}
		}
		return
	}
	if _, fever := count(3, "Winter"); fever != 0 {
		t.Fatal("no marsh fever in the early game")
	}
	fluW, feverW := count(9, "Winter")
	fluS, feverS := count(9, "Summer")
	if fluW <= feverW*2 || feverS <= fluS {
		t.Fatalf("winter is flu season, spring and summer bring fever: winter %d/%d summer %d/%d", fluW, feverW, fluS, feverS)
	}
}

func TestFoodPoisoningFromRottingFood(t *testing.T) {
	rate := func(it Item, roofed bool, temp float64) float64 {
		g := quiet(7)
		bad := 0
		for i := 0; i < 2000; i++ {
			c := &Colonist{}
			g.maybePoison(c, it, roofed, temp)
			if c.hasIll(DisPoison) {
				bad++
			}
		}
		return float64(bad) / 2000
	}
	open := rate(IFood, false, 20)
	if open < 0.05 || open > 0.12 {
		t.Fatalf("raw food left in the open on a warm day: %.2f", open)
	}
	if rate(IFood, true, 20) != 0 || rate(IFood, false, 5) != 0 {
		t.Fatal("roofed or cold-stored food is safe")
	}
	if m := rate(IMeal, false, 20); m >= open || m == 0 {
		t.Fatalf("cooked food is much safer but not perfectly: %.3f", m)
	}
	if rate(IWood, false, 20) != 0 {
		t.Fatal("only food")
	}
}

func TestEatingOutdoorsCanMakeYouIll(t *testing.T) {
	sick := 0
	for seed := int64(1); seed <= 60; seed++ {
		g, c := lone(seed)
		g.time = DayLen * 3.6 // a warm summer afternoon
		// the only food lies in the open
		for _, i := range g.zoneList {
			g.W.T[i].Item, g.W.T[i].ItemN = INone, 0
		}
		p := place(g, ONone, 5, 5)
		g.W.At(p.x, p.y).Item, g.W.At(p.x, p.y).ItemN = IFood, 3
		c.Hunger = 30
		for i := 0; i < 60*12; i++ {
			g.step(1.0 / 60)
		}
		if c.hasIll(DisPoison) {
			sick++
		}
	}
	if sick == 0 {
		t.Fatal("eating raw food from the open ground in summer should sometimes make you sick")
	}
}

func TestIllnessesSurviveSaves(t *testing.T) {
	g := quiet(7)
	c := g.cols[0]
	c.Ill = []Illness{{Kind: DisFever, Sev: 0.4, Imm: 0.2, Tended: 2, Quality: 1.3}}
	c.Immune[DisFlu] = 321
	g.nextOutbreak = g.time + DayLen*4
	path := t.TempDir() + "/s.sav"
	if err := g.SaveFile(path); err != nil {
		t.Fatal(err)
	}
	h := quiet(3)
	if err := h.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	d := h.cols[0]
	if len(d.Ill) != 1 || d.Ill[0] != (Illness{Kind: DisFever, Sev: 0.4, Imm: 0.2, Tended: 2, Quality: 1.3}) {
		t.Fatalf("illness lost: %+v", d.Ill)
	}
	if d.Immune[DisFlu] != 321 {
		t.Fatal("immunity lost")
	}
	if h.nextOutbreak < g.time+DayLen*2 {
		t.Fatal("the outbreak timer should be kept (and never fire right at loading)")
	}
}

func TestIllnessPanelAndLabel(t *testing.T) {
	if l := sickLine(Illness{Kind: DisFlu, Sev: 0.34, Tended: 2}); !contains(l, "Flu") || !contains(l, "34%") || !contains(l, "medicine") {
		t.Fatal(l)
	}
	if l := sickLine(Illness{Kind: DisFever, Sev: 0.9}); !contains(l, "CRITICAL") || !contains(l, "untreated") {
		t.Fatal(l)
	}
	g := quiet(7)
	g.scale, g.sw, g.sh = 1, 1280, 760
	c := g.cols[0]
	_, _, _, h0 := g.colPanelRect(c)
	c.Ill = []Illness{{Kind: DisFlu}, {Kind: DisPoison}}
	_, _, _, h1 := g.colPanelRect(c)
	if h1 != h0+32 {
		t.Fatalf("the panel grows by two lines: %.0f -> %.0f", h0, h1)
	}
}

func TestTwoIllnessesAtMost(t *testing.T) {
	g, c := lone(7)
	g.infect(c, DisFlu)
	g.infect(c, DisPoison)
	if g.infect(c, DisFever) || len(c.Ill) != 2 {
		t.Fatal("at most two illnesses at once")
	}
}

func TestMedicsTreatBleedingBeforeMildIllness(t *testing.T) {
	g := quiet(7)
	doc, flu, bleeder := g.cols[0], g.cols[1], g.cols[2]
	flu.Ill = []Illness{{Kind: DisFlu, Sev: 0.2, Imm: 0.05}}
	g.addWound(bleeder, "Bite", 8, true)
	// the sick one is standing right next to the doctor, the bleeding one far away
	doc.X, doc.Y = float64(g.start.x)+0.5, float64(g.start.y)+0.5
	flu.X, flu.Y = doc.X+1, doc.Y
	bleeder.X, bleeder.Y = doc.X+25, doc.Y
	if g.tendUrgency(bleeder) <= g.tendUrgency(flu) {
		t.Fatal("bleeding outranks a mild flu")
	}
	g.abort(doc)
	if !g.assignJob(doc) {
		t.Fatal("no job")
	}
	if doc.Job.Kind != JTend || doc.Job.Patient != bleeder.ID {
		t.Fatalf("the medic should go to the bleeding colonist first (job %d, patient %d)", doc.Job.Kind, doc.Job.Patient)
	}
	// a critical illness outranks a mild one
	g2 := quiet(7)
	a, b := g2.cols[0], g2.cols[1]
	a.Ill = []Illness{{Kind: DisFever, Sev: 0.7}}
	b.Ill = []Illness{{Kind: DisFlu, Sev: 0.2}}
	if g2.tendUrgency(a) <= g2.tendUrgency(b) {
		t.Fatal("a critical fever outranks a sniffle")
	}
}
