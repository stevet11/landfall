# Landfall Colony
Top-down 2D colony builder (RimWorld-style) in Go + Ebitengine. Linux and Windows.

Build: `make` (Linux needs X11/GL dev headers; Windows cross-builds with no extra tools).
Run: `./landfall [-seed N]`. Headless AI check: `make test`.

Systems: colonist jobs (chop/mine/harvest/haul/build), hunger, sleep + beds, farming, deer hunting,
wolf raids, seasons and temperature, rooms (walls + doors), fuel-burning heaters.
Controls: WASD pan, wheel zoom, tool bar / hotkeys 1-9,0,-,=,Backspace, Space pause, Tab speed, H help.

Saving: L opens the save slots (autosave, quicksave and five numbered slots, with day and colonist count; overwriting asks twice). F5 quick save, F9 quick load, F10 load autosave. Autosaves every 2 minutes and on exit; the game
continues from the autosave on launch (`-new` for a fresh world). Saves live in your user config dir
(`~/.config/landfall` on Linux, `%AppData%\landfall` on Windows). `go test` checks the save/load round trip.

Art and sound are generated in code at startup (pixel-art sprites in art.go, synthesised audio in sound.go),
so there are no asset files. M mutes. `LANDFALL_SHEET=sheet.png go test -run ArtSheet .` renders a sprite sheet.

Balance: `LANDFALL_BALANCE=1 go test -run Balance -v` plays scripted strategies ("competent", "lazy") over several seeds
for 12 game days and prints survival, raid and death statistics. Use it after changing any tuning constant.

Skills & economy: 7 skills (Combat, Construction, Mining, Farming, Medicine, Crafting, Engineering) that level by use or by
studying at a Desk; traits and passions per colonist. B = build menu (workbench, desk, lamp, generator, solar panel, fabricator,
electric heater), P = production targets, T = trade with visiting traders. Press "!" on a production row, or use the Priority tool (U) on orders and blueprints, to have colonists do them first. `./landfall -new -demo prod|trade|build|night` sets up a
showcase colony for screenshots.

Building materials: walls and doors can be wood (cheap, can burn), stone, or iron (mine Ore, smelt it at a Furnace; turn on Iron
in the Production panel). Pick the material in the palette that opens with the Wall/Door tool. Heaters, generators and furnaces
next to wooden things can start fires that spread; stone and iron never burn.

Roofs: the Roof tool (R, 2 wood per tile) is needed to make a walled room indoors: warm and heatable. Roofed goods never spoil; food, medicine and books left in the open rot (frozen goods keep). Your starting stockpile comes roofed.

Fire-fighting: colonists rush to put out fires by hand (waking sleepers), with some risk of burns. Hurt colonists (health 40 or
lower) keep out of it. Stone and iron stop a fire from spreading; roof and wall choices decide how much a fire can destroy.

Weapons: blades (Knife, Axe, Sword) are forged from iron at a Workbench (Crafting skill); Pistols, Rifles and Ammo come from a
Fabricator (Engineering skill, power). Colonists equip the best weapon in the stockpile on their own. Guns shoot from range if
there is a clear line (walls and trees block), use stockpiled Ammo and fall back to being a club without it. An Axe also chops faster.

Farming and cooking: the Farm tool opens a crop picker (vegetables, wheat, corn, cotton, aloe, tobacco); each crop has its own growing time
and minimum temperature. Aloe grows medicine. Cook bread (wheat) and meals (corn + food) at a Stove (Cooking skill, burns wood, can start
fires). Weather comes in spells of rain or snow: rain waters outdoor crops and puts out fires, clouds dim solar panels, and anyone outside gets colder and slower, so roofs matter. Cotton becomes cloth and parkas that protect against the cold. A Pickaxe mines 60% faster; colonists pick up tools and parkas on their own (a worn parka shows as a blue coat with a fur hood).

Raiders: every 3-5 days (from about day 5) armed raiders arrive from the edge of the map, announced by a horn, a red banner and an arrow.
They go for the nearest colonist; if nobody can be reached they raid the stockpile for valuables. Beat enough of them and the rest flee.
Kill them for silver and their weapons. Walls stop bullets; arm your colonists (see Weapons).

Moving buildings: the Move tool (V) picks up a furnace, stove, desk, bed, bench or machine; click a free spot and a colonist carries it
there, with its fuel, and nothing is lost. Right-click lets go; clicking a building that is already being moved cancels the move.

Disease: outbreaks (flu in winter, marsh fever later in the game) spread between colonists who stay close, faster inside shared rooms.
Sick colonists work slower and go to bed; a medic nurses them (medicine works much better, and skilled medics catch less). Left untreated
in the cold, marsh fever kills. Eating raw food left in the open on a warm day can cause food poisoning, so roof your stockpile. Recovery
brings immunity for a while.

Floors: the Floor tool opens a picker with wood planks, stone slabs, ceramic tile and a black-and-white checkerboard. Tile is fired from stone at a
Furnace (turn on Tile in the Production panel). Laying a different floor over an old one replaces it and returns half of the old materials. Harder
floors are quicker to walk on (+5% wood, +10% stone, +15% tile).

Joy: colonists need some fun. Joy drains slowly; below 50 it slows work, and a colonist with none left sulks and stops working for a while.
Build a Horseshoe pit, a Checkers table (two players enjoy it more) or a Meditation shrine (needs a Sculpture; leaves a calm that speeds up
work) from the Build menu. Colonists with low spirits drop their work to play, and idle ones play when they are not fully happy. A good meal
helps too. No games are played while there is danger about.

## License and builds
MIT, see `LICENSE`. GitHub Actions (`.github/workflows/build.yml`) vets and tests every push and pull request and builds Linux and Windows
(amd64) downloads; pushing a tag such as `v0.1.0` also publishes them as a GitHub release.
