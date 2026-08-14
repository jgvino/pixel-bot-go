<a name="top"></a>

# Pixel Bot

World of Warcraft fishing helper for Windows: watches a region, finds the bobber, reels when a bite is detected, repeats.

## Demo

https://github.com/user-attachments/assets/002c679b-130e-422f-99b9-4503c648b8f3

---

## Quick Start
1. Download or build.
2. Run `pixel-bot-go.exe` as Administrator (required for input injection).
3. In-game: ensure bobber area visible.
4. Select the game/window from the dropdown.
5. Click `Selection` and drag a tight box over the water where the bobber lands.
6. Click `Toggle Capture` to start.
7. Focus the game window (keep it active).
8. Automated fishing loop: Searching → Monitoring → Reeling → Repeat.
9. Click `Toggle Capture` again to end.

---

## Color Detection (this fork)

This fork adds an alternative detector that finds the bobber by **hue** instead
of by template pattern. It is **enabled by default** (`Color Detect` = `true`).

Why: NCC template matching converts frames to grayscale before comparing, which
discards the bobber's strongest signal — saturated red and blue feathers against
desaturated water. It also requires finding the correct template scale, which
varies with resolution and camera distance and is difficult to determine by hand.

Color detection is scale-invariant, rotation-invariant, and cheaper per frame.

**When `Color Detect` is `true`, these settings are ignored:** Min Scale, Max
Scale, Scale Step, Threshold, Stride, Refine, Stop On Score, Return Best Even.

Detection requires a red feather cluster **and** a blue one within `Color Max
Pair Distance`. Requiring both is far more selective than either alone: a lone
red flower or a lone blue quest marker is rejected.

Tuned defaults ship ready to use — `ROI Size Px` 120, `Max Cast Duration
Seconds` 25, `Cast Settle Ms` 2000. `Cast Settle Ms` blocks detection briefly
after each cast so the fading previous bobber is not mistaken for the new one.

Tune `Color Min Pixels` only. Nothing found → lower to 8. Locking onto gear or
NPCs → raise to 20.

### Settings location

Config is stored at `%APPDATA%\pixel-bot-go\pixle_bot_config.json`, so your
settings survive downloading a new build into a new folder. An existing
`pixle_bot_config.json` next to the executable still takes priority.

Set `Color Detect` to `false` to restore the original NCC behavior unchanged.

Full reference: [Color Detection](docs/COLOR_DETECTION.md)

---

Docs (Diataxis):
* [Tutorials](docs/TUTORIALS.md) – first fishing cycle & guided optimizations.
* [How-To Guides](docs/HOW_TO.md) – profiling, tuning, troubleshooting steps.
* [Reference](docs/REFERENCE.md) – algorithms, config parameters, state/event tables, FAQ.
* [Explanation](docs/EXPLANATION.md) – mental models, rationale, architecture concepts.
* [Color Detection](docs/COLOR_DETECTION.md) – hue-based bobber detection, config and tuning.

---

> Play nice: automation is generally NOT allowed on official or private servers. This is a helper, not a 24/7 unattended bot farm. Use it sparingly, stay present, and remember: you are 100% responsible for any consequences.

MIT License in `LICENSE`.



