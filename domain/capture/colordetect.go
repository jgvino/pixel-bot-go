package capture

import "image"

// ColorOptions configures hue-based bobber detection.
//
// Detection requires a red feather cluster AND a blue feather cluster within
// MaxPairDistance of each other. Requiring both is far more selective than
// either alone: a lone red flower or a lone blue quest marker is rejected,
// because almost nothing in a lake scene has saturated red immediately
// adjacent to saturated blue.
type ColorOptions struct {
	RedDelta        int // how much R must exceed B for a red feather pixel
	BlueDelta       int // how much B must exceed R for a blue feather pixel
	MinValue        int // minimum channel brightness; rejects dark noise
	MinPixels       int // smallest accepted red cluster
	MinBluePixels   int // smallest accepted blue cluster (blue is scarcer)
	MaxPixels       int // largest accepted cluster; rejects UI/terrain
	MaxPairDistance int // max centroid separation between red and blue
}

// DefaultColorOptions returns thresholds tuned for antialiased bobbers at
// native resolution.
func DefaultColorOptions() ColorOptions {
	return ColorOptions{
		RedDelta:        40,
		BlueDelta:       35,
		MinValue:        80,
		MinPixels:       12,
		MinBluePixels:   5,
		MaxPixels:       4000,
		MaxPairDistance: 40,
	}
}

// blob is a connected component: pixel count plus centroid.
type blob struct {
	count int
	cx    int
	cy    int
}

// findBlobs returns all 8-connected components in mask sized within
// [minPixels, maxPixels]. Coordinates are mask-relative.
func findBlobs(mask []bool, w, h, minPixels, maxPixels int) []blob {
	visited := make([]bool, len(mask))
	stack := make([]int, 0, 512)
	out := make([]blob, 0, 8)

	for start := 0; start < len(mask); start++ {
		if !mask[start] || visited[start] {
			continue
		}
		stack = stack[:0]
		stack = append(stack, start)
		visited[start] = true
		count, sumX, sumY := 0, 0, 0

		for len(stack) > 0 {
			p := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			px, py := p%w, p/w
			count++
			sumX += px
			sumY += py

			for dy := -1; dy <= 1; dy++ {
				ny := py + dy
				if ny < 0 || ny >= h {
					continue
				}
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx := px + dx
					if nx < 0 || nx >= w {
						continue
					}
					n := ny*w + nx
					if mask[n] && !visited[n] {
						visited[n] = true
						stack = append(stack, n)
					}
				}
			}
		}

		if count >= minPixels && count <= maxPixels {
			out = append(out, blob{count: count, cx: sumX / count, cy: sumY / count})
		}
	}
	return out
}

// DetectByColor locates the bobber by finding a red cluster paired with a
// nearby blue cluster, and returns the midpoint between their centroids.
//
// Scale-invariant and rotation-invariant: no template, no scale sweep.
// Scale is always reported as 1.0 since no scaling is performed.
func DetectByColor(frame *image.RGBA, opts ColorOptions) MultiScaleResult {
	var res MultiScaleResult
	res.Scale = 1.0
	res.ScalesEvaluated = 1

	if frame == nil {
		return res
	}
	b := frame.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return res
	}
	if opts.MinPixels <= 0 {
		opts.MinPixels = 12
	}
	if opts.MinBluePixels <= 0 {
		opts.MinBluePixels = 5
	}
	if opts.MaxPixels <= opts.MinPixels {
		opts.MaxPixels = 4000
	}
	if opts.MaxPairDistance <= 0 {
		opts.MaxPairDistance = 40
	}

	// Pass 1: build separate red and blue masks.
	redMask := make([]bool, w*h)
	blueMask := make([]bool, w*h)
	for y := 0; y < h; y++ {
		rowBase := y * w
		for x := 0; x < w; x++ {
			i := frame.PixOffset(b.Min.X+x, b.Min.Y+y)
			if frame.Pix[i+3] == 0 {
				continue
			}
			r := int(frame.Pix[i])
			bl := int(frame.Pix[i+2])
			if r-bl >= opts.RedDelta && r >= opts.MinValue {
				redMask[rowBase+x] = true
			} else if bl-r >= opts.BlueDelta && bl >= opts.MinValue {
				blueMask[rowBase+x] = true
			}
		}
	}

	// Pass 2: connected components in each mask.
	reds := findBlobs(redMask, w, h, opts.MinPixels, opts.MaxPixels)
	if len(reds) == 0 {
		return res
	}
	blues := findBlobs(blueMask, w, h, opts.MinBluePixels, opts.MaxPixels)
	if len(blues) == 0 {
		return res
	}

	// Pass 3: pair each red cluster with its nearest blue cluster. Among all
	// valid pairs, prefer the one with the largest combined pixel count, which
	// favors the closest/clearest bobber when several are in frame.
	maxDistSq := opts.MaxPairDistance * opts.MaxPairDistance
	bestScoreCount := 0
	bestX, bestY := 0, 0
	found := false

	for _, r := range reds {
		nearestDistSq := -1
		var nearest blob
		for _, bb := range blues {
			dx := r.cx - bb.cx
			dy := r.cy - bb.cy
			d := dx*dx + dy*dy
			if d <= maxDistSq && (nearestDistSq < 0 || d < nearestDistSq) {
				nearestDistSq = d
				nearest = bb
			}
		}
		if nearestDistSq < 0 {
			continue
		}
		combined := r.count + nearest.count
		if combined > bestScoreCount {
			bestScoreCount = combined
			bestX = (r.cx + nearest.cx) / 2
			bestY = (r.cy + nearest.cy) / 2
			found = true
		}
	}

	if !found {
		return res
	}

	res.X = b.Min.X + bestX
	res.Y = b.Min.Y + bestY
	res.Found = true

	s := float64(bestScoreCount) / float64((opts.MinPixels+opts.MinBluePixels)*3)
	if s > 1.0 {
		s = 1.0
	}
	res.Score = s
	return res
}
