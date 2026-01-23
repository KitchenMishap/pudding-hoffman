package kmeans

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
)

const USE_125 = false

func FindEpochPeaks(amounts []int64, k int) []float64 {

	if USE_125 {
		return findEpochPeaks125(amounts, k)
	}

	result := make([]float64, 0)
	bestBadness := math.MaxFloat64
	for try := 0; try < 5; try++ {
		guess, badness := guessEpochPeaksClock(amounts, k)
		if badness < bestBadness {
			bestBadness = badness
			result = guess
		}
	}
	return result
}

func findEpochPeaks125(amounts []int64, k int) []float64 {
	result := make([]float64, 0)
	bestBadness := math.MaxFloat64
	for try := 0; try < 5; try++ {
		guess, badness := guessEpochPeaksClock125(amounts, k)
		if badness < bestBadness {
			bestBadness = badness
			result = guess
		}
	}
	return result
}

func guessEpochPeaksClock(amounts []int64, k int) (logCentroids []float64, badnessScore float64) {
	// 1. Map all mantissas to the 0.0 to 1.0 "Clock face"
	phases := make([]float64, len(amounts))
	for i, v := range amounts {
		// log10(v) % 1 gives the position on the clock
		_, phases[i] = math.Modf(math.Log10(float64(v)))
		if phases[i] < 0.0 {
			phases[i] += 1.0
		}
	}

	logCentroids = initializeCentroids(phases, k)

	for i := 0; i < 10; i++ { // 10 iterations is usually enough for 1D
		clusters := make([][]float64, k)

		// 2. Assign to nearest centroid
		badnessScore = float64(0)
		for _, val := range phases {
			best := 0
			minDist := cyclicDistance(val, logCentroids[0])
			for j := 1; j < k; j++ {
				d := cyclicDistance(val, logCentroids[j])
				if d < minDist {
					minDist = d
					best = j
				}
			}
			clusters[best] = append(clusters[best], val)
			badnessScore += minDist
		}

		// 3. Update centroids using a "circular median" or mean
		for j := 0; j < k; j++ {
			if len(clusters[j]) > 0 {
				logCentroids[j] = circularMean(clusters[j])
			}
		}
		// badnessScore is one iteration out of date, but let's not get too picky!
	}
	return
}

const (
	log1 = 0.0           // math.Log10(1)
	log2 = 0.30102999566 // math.Log10(2)
	log5 = 0.69897000433 // math.Log10(5)
)

func guessEpochPeaksClock125(amounts []int64, k int) (logCentroids []float64, badnessScore float64) {
	phases := make([]float64, len(amounts))
	for i, v := range amounts {
		_, phases[i] = math.Modf(math.Log10(float64(v)))
		if phases[i] < 0.0 {
			phases[i] += 1.0
		}
	}

	logCentroids = initializeCentroids(phases, k)

	for i := 0; i < 10; i++ {
		clusters := make([][]float64, k)
		badnessScore = 0

		for _, val := range phases {
			bestCentroidIdx := 0
			minDist := 2.0 // Sentinel

			// Try every centroid's three harmonic possibilities
			for j := 0; j < k; j++ {
				// Find which harmonic of centroid[j] is closest to 'val'
				// We use the 1x base phase (logCentroids[j]
				d, _ := cyclicDistance125(val, logCentroids[j])

				if d < minDist {
					minDist = d
					bestCentroidIdx = j
				}
			}

			// NORMALIZE the value before appending it to the cluster
			// This rotates 2x and 5x points back to the 1x 'Root'
			normalizedVal := normalizeToFundamental(val, logCentroids[bestCentroidIdx])
			clusters[bestCentroidIdx] = append(clusters[bestCentroidIdx], normalizedVal)
			badnessScore += minDist
		}

		// Update centroids (now purely on normalized 1x bases)
		for j := 0; j < k; j++ {
			if len(clusters[j]) > 0 {
				logCentroids[j] = circularMean(clusters[j])
			}
		}
	}
	return
}

func normalizeToFundamental(val, centroid float64) float64 {
	d1 := cyclicDistance(val, centroid)
	d2 := cyclicDistance(val, math.Mod(centroid+log2, 1.0))
	d5 := cyclicDistance(val, math.Mod(centroid+log5, 1.0))

	if d1 <= d2 && d1 <= d5 {
		return val // Already at 1x
	} else if d2 < d5 {
		// It's a 2x hit, rotate back by log2
		res := val - log2
		if res < 0 {
			res += 1.0
		}
		return res
	} else {
		// It's a 5x hit, rotate back by log5
		res := val - log5
		if res < 0 {
			res += 1.0
		}
		return res
	}
}

func cyclicDistance(a, b float64) float64 {
	diff := math.Abs(a - b)
	if diff > 0.5 {
		return 1.0 - diff
	}
	return diff
}

func cyclicDistance125(a, b float64) (float64, int) {
	// The three harmonic "images" of the centroid in log space
	h1 := b                  // 1x
	h2 := fmod(b+0.301, 1.0) // 2x (b + log10(2))
	h5 := fmod(b+0.699, 1.0) // 5x (b + log10(5))

	d1 := cyclicDistance(a, h1)
	d2 := cyclicDistance(a, h2)
	d5 := cyclicDistance(a, h5)

	// Find the winner
	if d1 <= d2 && d1 <= d5 {
		return d1, 1
	} else if d2 <= d1 && d2 <= d5 {
		return d2, 2
	}
	return d5, 5
}

// Helper to keep the log phase within [0, 1)
func fmod(x, y float64) float64 {
	res := math.Mod(x, y)
	if res < 0 {
		res += y
	}
	return res
}

func initializeCentroids(mantissas []float64, k int) []float64 {
	result := make([]float64, k)
	count := len(mantissas)
	for i := 0; i < k; i++ {
		r := rand.Intn(count)
		c := mantissas[r]
		result[i] = c
	}
	return result
}

func circularMean(phases []float64) float64 {
	if len(phases) == 0 {
		return 0
	}

	var sumSin, sumCos float64
	for _, p := range phases {
		// 1. Convert phase (0..1) to radians (0..2π)
		angle := p * 2.0 * math.Pi

		// 2. Sum the Cartesian coordinates
		sumSin += math.Sin(angle)
		sumCos += math.Cos(angle)
	}

	// 3. Use Atan2 to find the angle of the average vector
	avgAngle := math.Atan2(sumSin, sumCos)

	// 4. Convert back from radians to phase (0..1)
	avgPhase := avgAngle / (2.0 * math.Pi)

	// 5. Ensure the result is in the [0, 1) range
	if avgPhase < 0 {
		avgPhase += 1.0
	}
	return avgPhase
}

func ExpPeakResidual(amount int64, logCentroids []float64) (exp int, peak int, harmonic int, residual int64) {
	if USE_125 {
		exp, peak, harmonic, residual = expPeakResidual125(amount, logCentroids)
		return
	}
	// If we-re not doing 1-2-5 harmonics, we'll just have to specify the harmonic as zero
	harmonic = 0

	// log10(v) % 1 gives the position on the clock
	e, logCentroid := math.Modf(math.Log10(float64(amount)))
	if logCentroid < 0.0 {
		logCentroid += 1.0
		e -= 1.0
	}
	exp = int(e)

	bestPeak := 0
	bestDiff := cyclicDistance(logCentroid, logCentroids[bestPeak])
	for p := 1; p < len(logCentroids); p++ {
		diff := cyclicDistance(logCentroid, logCentroids[p])
		if diff < bestDiff {
			bestDiff = diff
			bestPeak = p
		}
	}
	peak = bestPeak

	peakAmount := int64(math.Round(math.Pow(10, logCentroids[bestPeak]+float64(exp))))

	residual = amount - peakAmount

	return
}

func expPeakResidual125(amount int64, logCentroids []float64) (exp int, peak int, harmonic int, residual int64) {
	e, logVal := math.Modf(math.Log10(float64(amount)))
	if logVal < 0.0 {
		logVal += 1.0
		e -= 1.0
	}
	exp = int(e)

	bestPeak := 0
	bestHarmonic := 1
	minDist := 2.0

	for p := 0; p < len(logCentroids); p++ {
		d, h := cyclicDistance125(logVal, logCentroids[p])
		if d < minDist {
			minDist = d
			bestPeak = p
			bestHarmonic = h
		}
	}

	peak = bestPeak
	harmonic = bestHarmonic

	// Calculate the target amount based on the harmonic winner
	// peakAmount = 10^(exp) * 10^(logCentroid) * harmonic
	// Using Round to avoid floating point 49.999999 issues
	baseAmount := math.Pow(10, logCentroids[peak]+float64(exp))
	targetAmount := int64(math.Round(baseAmount * float64(harmonic)))

	residual = amount - targetAmount
	return
}

func ParallelKMeans(amountsEachEpoch [][]int64, epochs int64) [][]float64 {
	epochToPhasePeaks := make([][]float64, epochs)
	var wg sync.WaitGroup
	var completed int64 // atomic counter

	// Use a semaphore to limit concurrency to CPU count
	sem := make(chan struct{}, runtime.NumCPU())

	for i := int64(0); i < epochs; i++ {
		wg.Add(1)
		go func(epochID int64) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire token
			defer func() { <-sem }() // Release token

			if len(amountsEachEpoch[epochID]) < 7 {
				epochToPhasePeaks[epochID] = nil
			} else {
				// This is the heavy lifting
				epochToPhasePeaks[epochID] = FindEpochPeaks(amountsEachEpoch[epochID], 7)
			}

			// Report progress on completion
			done := atomic.AddInt64(&completed, 1)
			if done%10 == 0 || done == epochs {
				fmt.Printf("\r> KMeans Progress: [%d/%d] epochs (%.1f%%)    ",
					done, epochs, float64(done)/float64(epochs)*100)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("\nKmeans done.")
	return epochToPhasePeaks
}
