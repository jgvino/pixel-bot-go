# Color Detection

An alternative to NCC template matching that locates the bobber by hue instead
of by pattern. Enabled with `use_color_detect` (default `true` in this fork).

---

## Why it exists

NCC template matching converts every frame to grayscale before comparing
(`buildGrayPrecomp` in `ncc.go`), then normalizes brightness and variance away.
For a bobber whose most distinctive property is **saturated red and blue
feathers on desaturated water**, grayscale conversion deletes the single
strongest signal available.

The practical failure modes:

* A heavily upscaled template becomes a low-frequency blob that correlates with
  smooth water gradients, producing matches on plain water.
* The correct scale must be found by sweeping `MinScale`..`MaxScale`, which is
  resolution- and camera-distance-dependent and hard to determine by hand.
* Wide scale ranges multiply per-frame cost.

Color detection sidesteps all three.

---

## How it works

Two passes over the selection rectangle:

```
pass 1: build two masks
    isRed  = (R - B) >= RedDelta  and R >= MinValue
    isBlue = (B - R) >= BlueDelta and B >= MinValue

pass 2: 8-connected flood fill on each mask
    redBlobs  = components sized [MinPixels, MaxPixels]
    blueBlobs = components sized [MinBluePixels, MaxPixels]

pass 3: pair matching
    for each red blob:
        find nearest blue blob within MaxPairDistance
    keep the pair with the largest combined pixel count
    return the midpoint of the two centroids
```

Requiring red **and** blue in proximity is far more selective than either
alone. A lone red flower or a lone blue quest marker is rejected: almost
nothing in a lake scene has saturated red immediately adjacent to saturated
blue. The discriminating power multiplies rather than adds.

`Score` is normalized confidence, reaching `1.0` at three times `MinPixels`.
`Scale` is always reported as `1.0` since no scaling is performed.

### Properties

| Property | Behavior |
| -------- | -------- |
| Scale invariance | Complete. Pixel counts, not fixed-size patterns. |
| Cost | One mask pass plus one fill pass. No scale sweep, no integral images. |
| Rotation invariance | Complete. Blob area is orientation-independent. |
| Lighting sensitivity | Moderate. Channel *differences* survive brightness changes better than absolute values. |
| Environment sensitivity | High. Thresholds are tuned per water body and lighting. |

---

## Config Parameters

| Setting | Meaning | Tradeoff |
| ------- | ------- | -------- |
| UseColorDetect | Switch from NCC to hue-based detection | Ignores all scale/threshold/stride settings |
| ColorRedDelta | How far R must exceed B for a red feather pixel | ↑ stricter, fewer false pixels |
| ColorBlueDelta | How far B must exceed R for a blue feather pixel | ↑ stricter, fewer false pixels |
| ColorMinValue | Minimum channel brightness | ↑ rejects shadow noise, ↓ works in dim light |
| ColorMinPixels | Smallest accepted red cluster | ↑ fewer false positives, ↓ misses distant bobbers |
| ColorMinBluePixels | Smallest accepted blue cluster | Blue is scarcer; keep low |
| ColorMaxPairDistance | Max centroid gap between red and blue | ↑ looser pairing, ↓ rejects valid bobbers |
| ColorMaxPixels | Largest blob accepted | ↓ rejects UI panels and large terrain features |
| CastSettleMs | Detection blocked for this long after a cast | ↑ ignores the fading old bobber, ↓ faster reacquire |

When `UseColorDetect` is `true`, these settings are **ignored**: `MinScale`,
`MaxScale`, `ScaleStep`, `Threshold`, `Stride`, `Refine`, `StopOnScore`,
`ReturnBestEven`. The embedded template asset is not loaded.

---

## Tuning

Start with defaults. Adjust one value at a time.

| Symptom | Fix |
| ------- | --- |
| Nothing found | Lower `ColorMinPixels` to 15 |
| Nothing found at any size | Lower `ColorRedDelta`/`ColorBlueDelta` to 40/35 |
| Locks onto your gear or an NPC | Raise `ColorMinPixels` to 40 |
| Locks onto a static object | Lower `ColorMaxPairDistance` to 25 |
| Misses when blue feathers are dim | Lower `ColorMinBluePixels` to 3 |
| Locks onto UI elements | Lower `ColorMaxPixels` to 1500 |
| Works in daylight, fails at night | Lower `ColorMinValue` to 70 |
| Finds a bobber 0.2s after cast | Blob thresholds too loose; raise all three deltas |
| Locks onto the fading previous bobber | Raise `CastSettleMs` to 3000 |

### Verifying a real detection

A correct find appears **1.5–2 seconds after** the cast key, not immediately.
Anything faster is matching background, since the cast animation must complete
before a bobber exists.

A real bite logs a small localized change:

```
changedRatio ~0.2, diffBaseMean ~7-13
```

A false bite from the landing splash fills the ROI:

```
changedRatio ~1.0, diffBaseMean ~95-130
```

---

## Limitations

* Thresholds are derived from specific water and lighting. Different zones,
  weather, time of day, or shader settings will need retuning.
* Assumes a bobber with saturated red/blue feathers. Other bobber skins need
  different channel rules, not just different deltas.
* Anything with red adjacent to blue inside the selection rectangle competes.
  Keep the selection tight on water and exclude your character, UI, and shoreline.
* Blue feathers are the scarcer signal. At distance, in shadow, or at certain
  bob angles they may vanish entirely, and a conjunction fails when either half
  fails. Lower `ColorMinBluePixels` before loosening anything else.

---

## When to prefer NCC

Set `use_color_detect` to `false` and the original path is restored unchanged.
Template matching remains the better choice when the target is defined by
structure rather than hue, or when the environment varies too much for fixed
color thresholds.
