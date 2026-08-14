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
pass 1: build boolean mask
    for each pixel:
        isRed  = (R - B) >= RedDelta  and R >= MinValue
        isBlue = (B - R) >= BlueDelta and B >= MinValue
        mask   = isRed or isBlue

pass 2: 8-connected flood fill
    find all connected blobs in mask
    keep the largest
    reject if size < MinPixels or size > MaxPixels
    return centroid
```

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
| ColorMinPixels | Smallest blob accepted as a bobber | ↑ fewer false positives, ↓ misses distant bobbers |
| ColorMaxPixels | Largest blob accepted | ↓ rejects UI panels and large terrain features |

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
| Locks onto UI elements | Lower `ColorMaxPixels` to 1500 |
| Works in daylight, fails at night | Lower `ColorMinValue` to 70 |
| Finds a bobber 0.2s after cast | Blob thresholds too loose; raise all three deltas |

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
* Anything else red or blue inside the selection rectangle competes. Keep the
  selection tight on water and exclude your character, UI, and shoreline.
* Selecting the largest blob means a larger red object in frame wins outright.

---

## When to prefer NCC

Set `use_color_detect` to `false` and the original path is restored unchanged.
Template matching remains the better choice when the target is defined by
structure rather than hue, or when the environment varies too much for fixed
color thresholds.
