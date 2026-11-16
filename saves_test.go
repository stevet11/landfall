package main

import (
	"os"
	"testing"
)

func slotGame(t *testing.T) *Game {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return quiet(7)
}

func TestSaveSlotsListEmptyThenFilled(t *testing.T) {
	g := slotGame(t)
	g.refreshSlots()
	if len(g.slots) != numSlots+2 || !g.slots[0].Auto || g.slots[0].Exists {
		t.Fatalf("slots: %+v", g.slots)
	}
	if slotText(g.slots[2]) != "empty" {
		t.Fatalf("%q", slotText(g.slots[2]))
	}
	g.time = DayLen*3 + 5
	g.saveSlot(2)
	s := g.slots[2]
	if !s.Exists || s.Bad || s.Day != 4 || s.Cols != len(g.cols) || s.Season == "" {
		t.Fatalf("after saving: %+v", s)
	}
	if g.slots[3].Exists {
		t.Fatal("other slots stay empty")
	}
}

func TestSlotsLoadWhatWasSavedAndKeepSeparateGames(t *testing.T) {
	g := slotGame(t)
	g.refreshSlots()
	g.addToStock(IWood, 7)
	wood := g.stock[IWood]
	g.saveSlot(2)
	g.addToStock(IWood, 50)
	g.saveSlot(3)
	g.addToStock(IWood, 50)
	g.loadSlot(2)
	if g.stock[IWood] != wood {
		t.Fatalf("slot 1 held %d wood, loaded %d", wood, g.stock[IWood])
	}
	g.loadSlot(3)
	if g.stock[IWood] != wood+50 {
		t.Fatalf("slot 2 held %d wood, loaded %d", wood+50, g.stock[IWood])
	}
}

func TestOverwritingASlotNeedsASecondClick(t *testing.T) {
	g := slotGame(t)
	g.refreshSlots()
	g.saveSlot(2)
	g.addToStock(IWood, 40)
	g.saveSlot(2) // arms
	if g.saveArmed != 3 {
		t.Fatalf("armed %d", g.saveArmed)
	}
	before, _ := os.ReadFile(g.slots[2].Path)
	g.saveSlot(3) // a different slot (empty) saves at once and disarms
	if g.saveArmed != 0 || !g.slots[3].Exists {
		t.Fatal("saving an empty slot needs no confirmation")
	}
	g.saveSlot(2)
	g.saveSlot(2)
	after, _ := os.ReadFile(g.slots[2].Path)
	if len(before) == 0 || string(before) == string(after) {
		t.Fatal("the second click should overwrite the slot")
	}
}

func TestAutosaveSlotCannotBeWrittenByHand(t *testing.T) {
	g := slotGame(t)
	g.refreshSlots()
	g.saveSlot(0)
	if _, err := os.Stat(autoSavePath()); err == nil {
		t.Fatal("the autosave is only written by the game")
	}
	g.loadSlot(0) // nothing there: harmless
}

func TestBadSlotFileIsReportedNotLoaded(t *testing.T) {
	g := slotGame(t)
	os.WriteFile(slotPath(1), []byte("junk"), 0o644)
	g.refreshSlots()
	if !g.slots[2].Bad || slotText(g.slots[2]) != "unreadable file" {
		t.Fatalf("%+v", g.slots[2])
	}
	n := len(g.cols)
	g.loadSlot(2)
	if len(g.cols) != n {
		t.Fatal("a bad file must not change the game")
	}
}

func TestSavesPanelButtons(t *testing.T) {
	g := slotGame(t)
	g.sw, g.sh, g.scale = 1280, 760, 1
	g.togglePanel(panelSaves)
	if g.panel != panelSaves || len(g.slots) == 0 {
		t.Fatal("panel opens and lists slots")
	}
	saves, loads := 0, 0
	for _, b := range g.buttons() {
		switch b.label {
		case "Save":
			saves++
		case "Load":
			loads++
			if !b.disabled {
				t.Fatal("nothing saved yet: Load is disabled")
			}
		}
	}
	if saves != numSlots+1 || loads != numSlots+2 {
		t.Fatalf("saves %d loads %d", saves, loads)
	}
	if x, y, w, h := g.modalRect(); w == 0 || h == 0 || !g.overPanels(x+5, y+5) {
		t.Fatal("the panel has a rect that blocks clicks to the map")
	}
	g.togglePanel(panelSaves)
	if g.panel != panelNone {
		t.Fatal("L closes it")
	}
}
