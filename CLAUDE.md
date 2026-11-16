# Craft

Top-down 2D colony builder (RimWorld-style) in Go + Ebitengine v2. Runs on Linux and Windows.
Everything (art, sound) is generated in code at startup: there are no asset files. Single `main` package.

## Working on it
- Read `NEXT.md` first: it is the owner's to-do list. "Do the next item" means the top unchecked item under **Now**.
- Build both targets: `make` (Linux `./craft`, Windows `craft.exe`; no cgo or extra tools needed for Windows).
- Tests: `go test .` (~2 min now; use `-run Name` for the area you changed, the whole suite in the background). `go vet` prints "unkeyed fields" warnings for the `col` alias in art.go; ignore those.
- After touching balance-relevant numbers (hunger, wolves, cold, yields, costs, immigration):
  `CRAFT_BALANCE=1 go test -run Balance -v -timeout 40m .` (about 1 minute). A "competent" scripted player should survive winter
  with 0-2 deaths; a "lazy" one (no shelter) should die in winter. Tuning constants live in sim.go, animals.go, trader.go.
- Always add or update a test for new behaviour; the sim is headless so most things can be tested without a window.
- Keep the README and NEXT.md in step with what you build.

## Layout
| File | What is in it |
|---|---|
| world.go | Tile/World types, items, objects, blueprints, costs, map generation, BFS pathfinding |
| sim.go | Game state, colonist AI and jobs (`assignJob`, `doJob`), climate, rooms, power hook, injuries, designations (`applyTool`) |
| skills.go | Skills, XP, traits and their effects |
| econ.go | Item values, recipes/production, crafting and study jobs, electricity, build-blocker explanations |
| trader.go | Trader visits, prices, buying and selling, banner status text |
| move.go | Moving buildings: orders, carrying (Cargo), tidying, put-back on interruption |
| weather.go | Rain and snow spells (`weatherStep`, `precip`), what they do to crops, fire, solar, spoilage, walking and warmth |
| fire.go | Fire: ignition from burners, spreading, burning down, colonists fleeing flames and putting them out |
| weapons.go | Weapon stats, attack profile, shooting, line of sight, auto-equip |
| raiders.go | Raiders: raid size and weapons, AI (attack, steal, flee), loot, banner text |
| joy.go | Joy need, games, meditation, sulking |
| disease.go | Illness, contagion, outbreaks, food poisoning, nursing |
| animals.go | Deer, wolves, spawning on reachable land (`landmass`) |
| saves.go | Save slots (autosave, quicksave, five numbered): slot info, `saveSlot`/`loadSlot`; the Saves panel (L) lives in ui2.go |
| persist.go | Save/load (gob + gzip), `saveVersion` |
| art.go, art2.go, art3.go, art5.go | Procedural pixel-art sprites (32x32 canvases) |
| sound.go | Synthesised sound effects, ambience, music |
| draw.go | World rendering, particles, event-to-sound/particle mapping, tooltips |
| ui2.go | Panels: colonist skills, production, trade, build palette, trader banner, buttons |
| main.go | Window, input, camera, toolbar, top bar, help text |
| selftest.go | `-selftest` headless run and the `-demo` showcase setup |

## Conventions and gotchas
- The simulation (`step`, jobs, AI) must stay free of Ebitengine calls so tests and the balance harness run headless.
  Sim code announces sounds and effects through `g.emit(name, x, y)`; x < 0 means global. Messages: `g.msg` (9 s) and
  `g.alert` (25 s, gold, for things the player must not miss).
- Resources only count when they lie in a **Stockpile** (Zone) tile (`g.stock`). Loose items are hauled there.
  Building and crafting take materials straight from the stockpile.
- Colonists pick jobs in the order in `assignJob`: eat, tend wounds, recover, sleep, warm up, build, haul, reap, sow, hunt,
  craft, gather (chop/mine/harvest/salvage), study. Reservations (`ResT`/`ResD`/`ResI`) must be released on every exit path.
- One game day is 300 s (`DayLen`), a year is 10 days; winter is the real threat and requires a heated room.
- Enum values are saved in files: append new Obj/Blue/Item/Desig values, never reorder. When the save format changes,
  bump `saveVersion` in persist.go and keep older versions loading.
- Walls and doors share one Obj each (`OWall`, `ODoor`); their material is `Tile.Mat`. `Mat == MatNone` means a legacy tile
  (stone wall, wooden door), so old saves keep working. Use `blueCost()`/`costFor()`/`builtMat()`, never `costs[BWall]` directly.
  Only wood burns (`Tile.flammable()`). Fire tiles are not walkable. Colonists fight fires automatically; tests about
  fire itself must set `g.noFireFight = true` or the colonists will put it out.
- A walled room is only *indoors* (warm, heatable) when the tile also has `Tile.Roof`; use `g.indoors()` / `g.covered()`.
  Perishables (`spoilPerDay` in econ.go) rot unless roofed or frozen. Old saves get roofs on enclosed rooms when loaded.
- Crops are `Tile.CropKind` (0 = vegetables = what old farms grew; keep it that way). Gear (pickaxe, parka) lives in
  `Colonist.Tool`/`Wear`. Skills: saves from before Cooking kept a 7-entry array, so `savedColonist` has both `Skills` (legacy)
  and `SkillList`; add new skills the same careful way. Every panel needs a case in both `buttons()` and `drawPanels()`
  (a test checks).
- Moving a building: `Tile.MoveFrom` (on the destination ghost) / `MoveTo` (on the source) link the two; a colonist holds the
  building in `Colonist.Cargo` between lift and set-down. Anything that ends a job must go through `abort()` so cargo is put
  back; `snapshot()` also sets cargo down in the saved copy.
- Illness is a race between `Illness.Sev` (severity) and `Imm` (immunity); `careMults`, rest, cold and hunger tilt it (see
  disease.go). Anything that should stop a sick colonist working goes through `illWork()`/`illSpeed()`. Tests that need a
  quiet colony set `nextOutbreak` via `quiet()`.
- Floors: `Tile.FloorKind` (built; 0 = wood, what old floors were) and `BlueKind` (blueprint). Costs are `floorCost(kind)`;
  use `t.blueCost()`/`t.blueName()`, not `costs[BFloor]`. `stepPath` carries leftover movement across waypoints, so
  per-floor speed bonuses actually show.
- Joy (`Colonist.Joy`): drains slowly, low joy scales work via `moodFactor()`, and a colonist at zero sulks. Stations
  (`stationDefs`) hold two players via `ResT`/`ResD`; always release them through `abort()`/`release()`.
- Wood is the colony's fuel (generators, heaters, stoves, furnaces). Below `fuelReserve` wood, chopping comes first and the
  deep-cold "stay indoors" rule is off; `powerAdvice()` explains what is actually wrong with the power.
- The toolbar wraps into rows by window width (`toolRows`); add tools freely.
- Tool order in `toolInfo` (main.go) must match the `Tool` enum in sim.go.
- Every new building needs: an Obj + Blue enum entry, `costs`, `blueObj`/`blueNames`, a sprite (art2.go), drawing in draw.go,
  and an entry in `buildMenu` (ui2.go). Powered buildings also hook into `updatePower` in econ.go.

## Seeing the game
- `./craft -new -seed 7 -demo prod|trade|build|night|nopower|banner|arrow|walls|fire|roof|weapons|crops|raid|move|sick|joy|rain|snow` sets up a showcase colony (window title
  "Craft (demo)"). The user may be running their own game, titled "Craft", at the same time: capture only the demo window
  (`xwininfo -name "Craft (demo)"` then `xwd -id`), and never touch their saves.
- Saves are in `~/.config/craft/` (`autosave.sav`, `quicksave.sav`). `-new` starts fresh instead of continuing the autosave.
- Sprite sheet: `CRAFT_SHEET=sheet.png go test -run ArtSheet .` Howl audio dump: `CRAFT_WAV=howl.wav go test -run DumpHowl .`
- Put scratch files in the session scratchpad, not the project folder.
