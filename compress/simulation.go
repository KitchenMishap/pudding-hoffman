package compress

import (
	"fmt"
	"github.com/KitchenMishap/pudding-huffman/huffman"
	"github.com/KitchenMishap/pudding-huffman/kmeans"
	"math"
	"math/bits"
	"runtime"
	"sync"
)

type CompressionStats struct {
	TotalBits     uint64
	CelebrityHits uint64
	KMeansHits    uint64
	LiteralHits   uint64
}

func ParallelAmountStatistics(amounts []int64,
	blocksPerEpoch int,
	blockToTxo []int64,
	celebCodes map[int64]huffman.BitCode,
	max_base_10_exp int) (CompressionStats, [][]int64, []int64, []int64) {

	blocks := len(blockToTxo)
	epochs := blocks/blocksPerEpoch + 1

	fmt.Printf("Parallel phase...")

	numWorkers := runtime.NumCPU()
	if numWorkers > 4 {
		numWorkers -= 2
	} // Leave some free for OS

	// Channels for distribution and collection
	jobs := make(chan int, 100)
	type workerResult struct {
		stats          CompressionStats
		literalsSample [][]int64
		mags           []int64 // Base-2 magnitudes (for literals)
		expFreqs       []int64 // Base-10 exponents (for K-Means)
	}
	resultsChan := make(chan workerResult, numWorkers)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := workerResult{
				literalsSample: make([][]int64, epochs),
				mags:           make([]int64, 65),
				expFreqs:       make([]int64, max_base_10_exp),
			}
			// Optimization: Pre-allocate a 'slab' for each epoch in this worker
			// Based on 5% sampling of a typical block, 5,000 is a very safe starting capacity
			for e := 0; e < epochs; e++ {
				local.literalsSample[e] = make([]int64, 0, 5000)
			}

			for blockIdx := range jobs {
				epochID := blockIdx / blocksPerEpoch
				firstTxo := blockToTxo[blockIdx]
				lastTxo := int64(len(amounts)) // Rare fallback
				if blockIdx+1 < blocks {       // Usual case
					lastTxo = blockToTxo[blockIdx+1]
				}

				for txo := firstTxo; txo < lastTxo; txo++ {
					amount := amounts[txo]

					// Stage 1: Celebrity
					if _, ok := celebCodes[amount]; ok {
						local.stats.CelebrityHits++
						continue
					}

					// (The new) Stage 2: Literal (Initial Pass)
					local.stats.LiteralHits++
					// The following append was probably a RAM Killer (froze my 768GB RAM machine!)
					// We now only sample only 5% of data in a block to train the kmeans with.
					// But we retain the first 100 samples, to avoid data starvation when blocks are small
					if txo-firstTxo < 100 || txo%20 == 0 {
						local.literalsSample[epochID] = append(local.literalsSample[epochID], amount)
					}
					if amount > 0 {
						local.mags[bits.Len64(uint64(amount))]++
						exponent := int(math.Floor(math.Log10(float64(amount))))
						if exponent >= 0 && exponent < len(local.expFreqs) {
							local.expFreqs[exponent]++
						}
					}
				}
			}
			resultsChan <- local
		}()
	}

	// Feed the workers
	for b := 0; b < blocks; b++ {
		jobs <- b
	}
	close(jobs)
	wg.Wait()
	close(resultsChan)

	fmt.Printf("Reduce phase (serial)...")
	// --- REDUCE PHASE ---
	finalStats := CompressionStats{}
	finalMags := make([]int64, 65)
	finalLiterals := make([][]int64, epochs)
	finalExpFreqs := make([]int64, max_base_10_exp)

	for res := range resultsChan {
		finalStats.CelebrityHits += res.stats.CelebrityHits
		finalStats.LiteralHits += res.stats.LiteralHits

		for i := 0; i < 65; i++ {
			finalMags[i] += res.mags[i]
		}
		for i := 0; i < max_base_10_exp; i++ {
			finalExpFreqs[i] += res.expFreqs[i]
		}

		for e := 0; e < epochs; e++ {
			if len(res.literalsSample[e]) > 0 {
				finalLiterals[e] = append(finalLiterals[e], res.literalsSample[e]...)
			}
		}
	}

	return finalStats, finalLiterals, finalMags, finalExpFreqs
}

func ParallelSimulateCompressionWithKMeans(amounts []int64,
	blocksPerEpoch int,
	blockToTxo []int64,
	celebCodes map[int64]huffman.BitCode,
	expCodes map[int64]huffman.BitCode,
	residualCodes map[int64]huffman.BitCode,
	magnitudeCodes map[int64]huffman.BitCode,
	epochToPhasePeaks [][]float64,
	escapeValue int64) (CompressionStats, [][7]int64) {

	blocks := len(blockToTxo)
	epochs := blocks/blocksPerEpoch + 1

	numWorkers := runtime.NumCPU()
	if numWorkers > 4 {
		numWorkers -= 2
	} // Leave some free for OS

	jobs := make(chan int, 100)
	type workerResult struct {
		stats         CompressionStats
		peakStrengths [][7]int64
	}
	resultsChan := make(chan workerResult, numWorkers)
	var wg sync.WaitGroup

	esc1Len := celebCodes[escapeValue].Length // Something that means "Not a celebrity"
	esc2Len := expCodes[escapeValue].Length   // Something that means "Not a K-Means"

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := workerResult{
				peakStrengths: make([][7]int64, epochs),
			}

			for blockIdx := range jobs {
				epochID := blockIdx / blocksPerEpoch
				firstTxo := blockToTxo[blockIdx]
				lastTxo := int64(len(amounts)) // Rare fallback
				if blockIdx+1 < blocks {       // Usual case
					lastTxo = blockToTxo[blockIdx+1]
				}

				for txo := firstTxo; txo < lastTxo; txo++ {
					amount := amounts[txo]

					// Stage 1: Celebrity
					if aCode, ok := celebCodes[amount]; ok {
						local.stats.TotalBits += uint64(aCode.Length)
						local.stats.CelebrityHits++
						continue
					}

					// (The new) Stage 2: K-Means
					local.stats.TotalBits += uint64(esc1Len) // The "not a celebrity" escape code penalty
					if epochToPhasePeaks[epochID] != nil {
						e, peakIdx, r := kmeans.ExpPeakResidual(amount, epochToPhasePeaks[epochID])
						if rCode, ok := residualCodes[r]; ok {
							local.stats.TotalBits += 3 // 3 bits peak index (for a peak index between 0 and 6)
							local.peakStrengths[epochID][peakIdx]++
							if eCode, ok := expCodes[int64(e)]; ok {
								local.stats.TotalBits += uint64(eCode.Length)
							} else {
								panic("missing exp code")
							}
							local.stats.TotalBits += uint64(rCode.Length)
							local.stats.KMeansHits++
							continue
						}
					}

					// Stage 3: Magnitude-encoded Literal
					local.stats.TotalBits += uint64(esc2Len) // This sits in the expCode and means "Not a K-Means"
					mag := int64(bits.Len64(uint64(amount)))
					// One bit saving is clever. Because we can assume "0" is a celebrity (in fact we found that
					// it's the most popular celebrity!), we know that amount is non zero. So we don't need
					// to store mag bits, because we ALWAYS ALREADY KNOW that the first bit will be a 1. Why store it?
					const oneBitSaving = 1
					local.stats.TotalBits += uint64(magnitudeCodes[mag].Length) + uint64(mag-oneBitSaving)
					local.stats.LiteralHits++
				}
			}
			resultsChan <- local
		}()
	}

	// Dispatcher
	for b := 0; b < blocks; b++ {
		jobs <- b
	}
	close(jobs)
	wg.Wait()
	close(resultsChan)

	// Final Reduction
	globalStats := CompressionStats{}
	globalStrengths := make([][7]int64, epochs)
	for res := range resultsChan {
		globalStats.TotalBits += res.stats.TotalBits
		globalStats.CelebrityHits += res.stats.CelebrityHits
		globalStats.KMeansHits += res.stats.KMeansHits
		globalStats.LiteralHits += res.stats.LiteralHits

		for e := 0; e < epochs; e++ {
			for p := 0; p < 7; p++ {
				globalStrengths[e][p] += res.peakStrengths[e][p]
			}
		}
	}

	return globalStats, globalStrengths
}
