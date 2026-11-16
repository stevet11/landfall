package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Save slots: the autosave (load only), the quicksave (F5 / F9) and numbered slots chosen in the
// Saves panel (L).

const numSlots = 5

type SaveSlot struct {
	Name   string
	Path   string
	Auto   bool // the autosave: can be loaded but not written by hand
	Exists bool
	Bad    bool // there is a file but it can't be read
	Day    int
	Season string
	Cols   int
	When   time.Time
}

func slotPath(i int) string { return filepath.Join(saveDir(), fmt.Sprintf("slot%d.sav", i)) }

// slotDefs lists the slots in the order the panel shows them.
func slotDefs() []SaveSlot {
	s := []SaveSlot{
		{Name: "Autosave", Path: autoSavePath(), Auto: true},
		{Name: "Quicksave (F5)", Path: quickSavePath()},
	}
	for i := 1; i <= numSlots; i++ {
		s = append(s, SaveSlot{Name: fmt.Sprintf("Slot %d", i), Path: slotPath(i)})
	}
	return s
}

// slotInfo fills in what a slot's file holds.
func slotInfo(s SaveSlot) SaveSlot {
	st, err := os.Stat(s.Path)
	if err != nil {
		return s
	}
	s.Exists, s.When = true, st.ModTime()
	sd, err := readSave(s.Path)
	if err != nil {
		s.Bad = true
		return s
	}
	s.Day = int(sd.Time/DayLen) + 1
	s.Season = (&Game{time: sd.Time}).seasonName()
	s.Cols = len(sd.Cols)
	return s
}

func (g *Game) refreshSlots() {
	g.slots = g.slots[:0]
	for _, s := range slotDefs() {
		g.slots = append(g.slots, slotInfo(s))
	}
}

// slotText is the one-line description shown next to a slot.
func slotText(s SaveSlot) string {
	switch {
	case s.Bad:
		return "unreadable file"
	case !s.Exists:
		return "empty"
	}
	return fmt.Sprintf("%s, day %d, %d colonists  (%s)", s.Season, s.Day, s.Cols, s.When.Format("Jan 2 15:04"))
}

// saveSlot writes the game into a slot. Writing over a used slot asks first: the first click arms it.
func (g *Game) saveSlot(i int) {
	if i < 0 || i >= len(g.slots) || g.slots[i].Auto {
		return
	}
	s := g.slots[i]
	if s.Exists && g.saveArmed != i+1 {
		g.saveArmed = i + 1
		g.msg("Click Save again to overwrite %s.", s.Name)
		return
	}
	g.saveArmed = 0
	if err := g.SaveFile(s.Path); err != nil {
		g.msg("Save failed: %v", err)
		return
	}
	g.msg("Saved to %s.", s.Name)
	g.audio.play("save", 0.6)
	g.refreshSlots()
}

// loadSlot replaces the running game with the one in a slot.
func (g *Game) loadSlot(i int) {
	if i < 0 || i >= len(g.slots) || !g.slots[i].Exists || g.slots[i].Bad {
		return
	}
	s := g.slots[i]
	g.saveArmed = 0
	if err := g.LoadFile(s.Path); err != nil {
		g.msg("Could not load %s (%v)", s.Name, err)
		return
	}
	g.panel = panelNone
	g.msg("Loaded %s.", s.Name)
}
