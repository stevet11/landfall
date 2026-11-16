package main

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
)

const saveVersion = 6

// legacySkills is how many skills saves up to version 4 knew about (cooking came later).
const legacySkills = 7

// Saved forms contain only exported, pointer-free fields so gob can encode
// them. Jobs and paths are not saved: colonists simply pick new work on load.
type savedColonist struct {
	ID                   int
	Name                 string
	X, Y                 float64
	Hunger, Rest, Health float64
	Col                  color.NRGBA
	Carry                Item
	CarryN               int
	Injuries             []Injury
	Skills               [legacySkills]SkillState
	SkillList            []SkillState // all skills (saves from version 5 on)
	Traits               []Trait
	Focus                Skill
	Weapon               Item
	Tool, Wear           Item
	Joy, Calm, Sulk      float64
	Ill                  []Illness
	Immune               []float64
}

type savedAnimal struct {
	ID     int
	Kind   AKind
	X, Y   float64
	HP     float64
	MaxHP  float64
	Hunt   bool
	Weapon Item
	Goal   int
	Loot   Item
	LootN  int
	Exit   [2]int
}

type saveData struct {
	Version          int
	Tiles            []Tile
	Cols             []savedColonist
	Animals          []savedAnimal
	NextID           int
	Time             float64
	NextRaid         float64
	ImmT             float64
	DeerT            float64
	RegrowT          float64
	Start            [2]int
	CamX             float64
	CamY             float64
	Zoom             float64
	Speed            int
	Prod             []ProdSetting
	NextTrader       float64
	NextRaiders      float64
	NextOutbreak     float64
	WallMat, DoorMat Material
	CropSel          CropKind
	FloorSel         FloorKind
}

func saveDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	d = filepath.Join(d, "landfall")
	_ = os.MkdirAll(d, 0o755)
	return d
}

func quickSavePath() string { return filepath.Join(saveDir(), "quicksave.sav") }
func autoSavePath() string  { return filepath.Join(saveDir(), "autosave.sav") }

func (g *Game) snapshot() *saveData {
	sd := &saveData{
		Version: saveVersion, NextID: g.nextID, Time: g.time, NextRaid: g.nextRaid,
		ImmT: g.immT, DeerT: g.deerT, RegrowT: g.regrowT, Start: [2]int{g.start.x, g.start.y},
		CamX: g.camX, CamY: g.camY, Zoom: g.zoom, Speed: g.speed,
		Prod: append([]ProdSetting(nil), g.prod...), NextTrader: g.nextTrader, NextRaiders: g.nextRaiders, NextOutbreak: g.nextOutbreak, WallMat: g.wallMat, DoorMat: g.doorMat, CropSel: g.cropSel, FloorSel: g.floorSel,
	}
	sd.Tiles = make([]Tile, len(g.W.T))
	copy(sd.Tiles, g.W.T)
	for i := range sd.Tiles { // reservations belong to jobs, which aren't saved
		sd.Tiles[i].ResT, sd.Tiles[i].ResD, sd.Tiles[i].ResI = 0, 0, 0
	}
	for _, c := range g.cols { // jobs aren't saved, so a building in someone's arms is set down again
		if cg := c.Cargo; cg != nil {
			if d := &sd.Tiles[cg.Dst]; int(d.MoveFrom) == cg.Src+1 {
				d.Blue, d.Mat, d.MoveFrom = BNone, MatNone, 0
			}
			home := cg.Src
			if s := &sd.Tiles[home]; s.Obj != ONone || s.Blue != BNone {
				home = cg.Dst
			}
			sd.Tiles[home].Obj, sd.Tiles[home].Mat, sd.Tiles[home].Fuel = cg.Obj, cg.Mat, cg.Fuel
		}
	}
	for _, c := range g.cols {
		sd.Cols = append(sd.Cols, savedColonist{c.ID, c.Name, c.X, c.Y, c.Hunger, c.Rest, c.Health, c.Col, c.Carry, c.CarryN, append([]Injury(nil), c.Injuries...), legacySkillArray(c), append([]SkillState(nil), c.Skills[:]...), append([]Trait(nil), c.Traits...), c.Focus, c.Weapon, c.Tool, c.Wear, c.Joy, c.Calm, c.Sulk, append([]Illness(nil), c.Ill...), append([]float64(nil), c.Immune[:]...)})
	}
	for _, a := range g.animals {
		sd.Animals = append(sd.Animals, savedAnimal{a.ID, a.Kind, a.X, a.Y, a.HP, a.MaxHP, a.Hunt, a.Weapon, a.Goal, a.Loot, a.LootN, [2]int{a.Exit.x, a.Exit.y}})
	}
	return sd
}

func (g *Game) SaveFile(path string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(f)
	err = gob.NewEncoder(zw).Encode(g.snapshot())
	if e := zw.Close(); err == nil {
		err = e
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// readSave reads and checks a save file without touching any game.
func readSave(path string) (*saveData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var sd saveData
	if err := gob.NewDecoder(zr).Decode(&sd); err != nil {
		return nil, err
	}
	if sd.Version < 1 || sd.Version > saveVersion {
		return nil, fmt.Errorf("unsupported save version %d", sd.Version)
	}
	if len(sd.Tiles) != MapW*MapH {
		return nil, fmt.Errorf("save has wrong map size")
	}
	return &sd, nil
}

func (g *Game) LoadFile(path string) error {
	sd, err := readSave(path)
	if err != nil {
		return err
	}
	g.apply(sd)
	return nil
}

func (g *Game) apply(sd *saveData) {
	g.W.T = sd.Tiles
	g.nextID, g.time, g.nextRaid = sd.NextID, sd.Time, sd.NextRaid
	g.immT, g.deerT, g.regrowT = sd.ImmT, sd.DeerT, sd.RegrowT
	g.start = pt{sd.Start[0], sd.Start[1]}
	g.land = nil
	g.cols, g.animals = nil, nil
	g.over = false
	for _, s := range sd.Cols {
		nc := &Colonist{
			ID: s.ID, Name: s.Name, X: s.X, Y: s.Y, Hunger: s.Hunger, Rest: s.Rest,
			Health: s.Health, Col: s.Col, Carry: s.Carry, CarryN: s.CarryN, Injuries: s.Injuries, Traits: s.Traits, Focus: s.Focus, Weapon: s.Weapon, Tool: s.Tool, Wear: s.Wear, Joy: s.Joy, Calm: s.Calm, Sulk: s.Sulk, Ill: s.Ill, Status: "Idle",
		}
		g.loadSkills(nc, s)
		copy(nc.Immune[:], s.Immune)
		g.cols = append(g.cols, nc)
	}
	for _, s := range sd.Animals {
		g.animals = append(g.animals, &Animal{ID: s.ID, Kind: s.Kind, X: s.X, Y: s.Y, HP: s.HP, MaxHP: math.Max(s.MaxHP, s.HP), Hunt: s.Hunt, DX: 1,
			Weapon: s.Weapon, Goal: s.Goal, Loot: s.Loot, LootN: s.LootN, Exit: pt{s.Exit[0], s.Exit[1]}})
	}
	g.camX, g.camY, g.zoom = sd.CamX, sd.CamY, sd.Zoom
	g.prod = defaultProd()
	for i := range g.prod {
		if i < len(sd.Prod) {
			g.prod[i] = sd.Prod[i]
		}
	}
	g.wallMat, g.doorMat = MatStone, MatWood
	if sd.WallMat != MatNone {
		g.wallMat = sd.WallMat
	}
	if sd.DoorMat != MatNone {
		g.doorMat = sd.DoorMat
	}
	g.cropSel, g.floorSel = sd.CropSel, sd.FloorSel
	g.nextRaiders = math.Max(sd.NextRaiders, sd.Time+DayLen*2)
	g.nextOutbreak = math.Max(sd.NextOutbreak, sd.Time+DayLen*2)
	g.raid = nil
	if live := g.liveRaiders(); len(live) > 0 { // a raid was in progress when the game was saved
		g.raid = &raidInfo{Start: len(live), Origin: live[0].Exit}
	}
	g.trader = nil
	g.nextTrader = math.Max(sd.NextTrader, sd.Time+DayLen*0.5)
	if sd.Speed >= 1 && sd.Speed <= 3 {
		g.speed = sd.Speed
	}
	g.selected, g.dragging, g.moveSrc = 0, false, -1
	if sd.Version < 4 { // older saves predate roofs: roof every enclosed room so heating keeps working
		g.computeRooms()
		for i := range g.W.T {
			if r := g.roomOf[i]; r >= 0 && g.rooms[r].Indoor {
				g.W.T[i].Roof = true
			}
		}
		g.msg("Roofs are new: unroofed food spoils. Use the Roof tool (R) on your stockpile.")
	}
	if sd.Version < 6 { // saves from before joy existed: everyone starts in good spirits
		for _, c := range g.cols {
			c.Joy = 70
		}
	}
	if sd.Version < 3 { // older saves predate skills: give everyone some
		for _, c := range g.cols {
			g.rollColonist(c, 0, 0)
		}
	}
	g.rescueItems()
	g.computeRooms()
	g.outT = g.outdoorTemp()
	g.slow(0)
	for _, c := range g.cols {
		g.dropCarry(c)
	}
}

func (g *Game) quickSave() {
	if err := g.SaveFile(quickSavePath()); err != nil {
		g.msg("Save failed: %v", err)
		return
	}
	g.msg("Game saved.")
	g.audio.play("save", 0.6)
}

func (g *Game) quickLoad(path, what string) {
	if err := g.LoadFile(path); err != nil {
		g.msg("No %s to load (%v)", what, err)
		return
	}
	g.msg("Loaded %s.", what)
}

func (g *Game) autoSave() {
	if len(g.cols) == 0 {
		return // never overwrite a good save with a dead colony
	}
	_ = g.SaveFile(autoSavePath())
}

func legacySkillArray(c *Colonist) (a [legacySkills]SkillState) {
	copy(a[:], c.Skills[:])
	return
}

// loadSkills restores a colonist's skills. Saves from before cooking existed only know the first
// seven; the new skill gets a small random start so old colonists aren't all hopeless cooks.
func (g *Game) loadSkills(c *Colonist, s savedColonist) {
	if len(s.SkillList) == int(numSkills) {
		copy(c.Skills[:], s.SkillList)
		return
	}
	copy(c.Skills[:], s.Skills[:])
	c.Skills[SCooking] = SkillState{Level: g.rng.Intn(4), Passion: g.rng.Float64() < 0.2}
}
