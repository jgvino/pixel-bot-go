package capture

import "image"

// ColorOptions configures hue-based bobber detection.
//
// Unlike NCC template matching, this approach is scale-invariant: it looks for
// saturated red/blue feather pixels rather than a fixed-size pattern, so the
// MinScale/MaxScale/ScaleStep/Threshold/Stride settings do not apply.
type ColorOptions struct {
	RedDelta  int // how much R must exceed B to count as a red feather pixel
	BlueDelta int // how much B must exceed R to count as a blue feather pixel
	MinValue  int // minimum channel brightness; rejects dark noise
	MinPixels int // smallest blob accepted as a bobber
	MaxPixels int // largest blob accepted; rejects large UI/terrain regions
}

// DefaultColorOptions returns thresholds derived from Stonetalon lake samples.
func DefaultColorOptions() ColorOptions {
	return ColorOptions{
		RedDelta:  60,
		BlueDelta: 50,
		MinValue:  100,
		MinPixels: 25,
		MaxPixels: 4000,
	}
}

// DetectByColor finds the largest connected cluster of saturated red/blue
// pixels in the frame and returns its centroid in frame coordinates.
//
// Score is a normalized confidence: it reaches 1.0 at three times MinPixels.
// Scale is reported as 1.0 since no scaling is performed.
func DetectByColor(frame *image.RGBA, opts ColorOptions) MultiScaleResult {
	var res MultiScaleResult
	if frame == nil {
		return res
	}
	b := frame.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return res
	}
	if opts.MinPixels <= 0 {
		opts.MinPixels = 25
	}
	if opts.MaxPixels <= 0 {
		opts.MaxPixels = 4000
	}

	// Pass 1: build the feather mask.
	mask := make([]bool, w*h)
	for y := 0; y < h; y++ {
		rowBase := y * w
		for x := 0; x < w; x++ {
			i := frame.PixOffset(b.Min.X+x, b.Min.Y+y)
			if frame.Pix[i+3] == 0 {
				continue
			}
			r := int(frame.Pix[i])
			bl := int(frame.Pix[i+2])
			isRed := r-bl >= opts.RedDelta && r >= opts.MinValue
			isBlue := bl-r >= opts.BlueDelta && bl >= opts.MinValue
			if isRed || isBlue {
				mask[rowBase+x] = true
			}
		}
	}

	// Pass 2: iterative flood fill (8-connected) to find the largest blob.
	visited := make([]bool, w*h)
	stack := make([]int, 0, 1024)
	bestCount, bestSumX, bestSumY := 0, 0, 0

	for start := 0; start < w*h; start++ {
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

		if count > bestCount {
			bestCount, bestSumX, bestSumY = count, sumX, sumY
		}
	}

	res.ScalesEvaluated = 1
	res.Scale = 1.0
	if bestCount < opts.MinPixels || bestCount > opts.MaxPixels {
		return res
	}

	res.X = b.Min.X + bestSumX/bestCount
	res.Y = b.Min.Y + bestSumY/bestCount
	res.Found = true

	s := float64(bestCount) / float64(opts.MinPixels*3)
	if s > 1.0 {
		s = 1.0
	}
	res.Score = s
	return res
}
