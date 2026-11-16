package main

import "testing"

func TestGameOverWhenAllColonistsDie(t *testing.T) {
	g := quiet(3)
	for i := 0; i < 60; i++ {
		g.step(1.0 / 60)
	}
	if g.over {
		t.Fatal("game over with living colonists")
	}
	for _, c := range g.cols[:len(g.cols)-1] {
		c.Health = -5
	}
	g.step(1.0 / 60)
	if g.over || len(g.cols) != 1 {
		t.Fatalf("one colonist left should not end the game (over=%v, cols=%d)", g.over, len(g.cols))
	}
	g.cols[0].Health = -5
	g.step(1.0 / 60)
	if !g.over || g.overDay < 1 {
		t.Fatal("the game should be over once every colonist is dead")
	}
	if g.deathSummary() == "" {
		t.Error("expected a cause of death summary")
	}
	g.restart(5)
	if g.over || len(g.cols) != 3 || g.onEvent == nil {
		t.Fatalf("restart should give a fresh colony (over=%v, cols=%d)", g.over, len(g.cols))
	}
}
