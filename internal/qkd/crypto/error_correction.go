package crypto

import (
	"fmt"

	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

// ErrorCorrection implements error correction algorithms for QKD
// Primary algorithm: Cascade - interactive error correction protocol

// CascadeCorrector implements the Cascade error correction algorithm
type CascadeCorrector struct {
	passes    int     // Number of Cascade passes
	blockSize int     // Initial block size
	errorRate float64 // Estimated error rate
}

// NewCascadeCorrector creates a new Cascade error corrector
func NewCascadeCorrector(errorRate float64) *CascadeCorrector {
	// Initial block size based on error rate (heuristic)
	blockSize := 1
	if errorRate > 0 {
		blockSize = int(0.73 / errorRate)
		if blockSize < 1 {
			blockSize = 1
		}
	}

	return &CascadeCorrector{
		passes:    4,         // Standard: 4 passes
		blockSize: blockSize,
		errorRate: errorRate,
	}
}

// Block represents a block of bits with parity
type Block struct {
	StartIndex int
	EndIndex   int
	Parity     quantum.Bit
}

// CalculateParity calculates the XOR parity of a slice of bits
func CalculateParity(bits []quantum.Bit) quantum.Bit {
	parity := quantum.Zero
	for _, bit := range bits {
		parity = parity ^ bit
	}
	return parity
}

// Correct performs Cascade error correction between Alice and Bob's keys
// Alice's key is the reference, Bob's key will be corrected
func (c *CascadeCorrector) Correct(aliceKey, bobKey []quantum.Bit) ([]quantum.Bit, int, error) {
	if len(aliceKey) != len(bobKey) {
		return nil, 0, fmt.Errorf("keys must have the same length")
	}

	keyLength := len(aliceKey)
	corrected := make([]quantum.Bit, keyLength)
	copy(corrected, bobKey)

	totalDisclosedBits := 0
	blockSize := c.blockSize

	// Perform multiple Cascade passes
	for pass := 0; pass < c.passes; pass++ {
		// Divide key into blocks
		numBlocks := (keyLength + blockSize - 1) / blockSize
		blocks := make([]Block, numBlocks)

		for i := 0; i < numBlocks; i++ {
			startIdx := i * blockSize
			endIdx := startIdx + blockSize
			if endIdx > keyLength {
				endIdx = keyLength
			}

			blocks[i] = Block{
				StartIndex: startIdx,
				EndIndex:   endIdx,
			}
		}

		// Binary search for errors in each block
		for i := range blocks {
			aliceBlock := aliceKey[blocks[i].StartIndex:blocks[i].EndIndex]
			bobBlock := corrected[blocks[i].StartIndex:blocks[i].EndIndex]

			aliceParity := CalculateParity(aliceBlock)
			bobParity := CalculateParity(bobBlock)

			// If parities differ, there's an odd number of errors in this block
			if aliceParity != bobParity {
				// Binary search to find and correct the error
				errorIdx, disclosed := c.binarySearch(aliceKey, corrected, blocks[i].StartIndex, blocks[i].EndIndex)
				totalDisclosedBits += disclosed

				if errorIdx >= 0 {
					// Flip the erroneous bit
					corrected[errorIdx] = 1 - corrected[errorIdx]
				}
			}

			totalDisclosedBits++ // Each parity comparison discloses 1 bit of information
		}

		// Double block size for next pass (Cascade heuristic)
		blockSize *= 2
	}

	return corrected, totalDisclosedBits, nil
}

// binarySearch performs binary search to find an error within a block
func (c *CascadeCorrector) binarySearch(aliceKey, bobKey []quantum.Bit, start, end int) (int, int) {
	disclosedBits := 0

	for start < end-1 {
		mid := (start + end) / 2

		aliceParity := CalculateParity(aliceKey[start:mid])
		bobParity := CalculateParity(bobKey[start:mid])
		disclosedBits++

		if aliceParity != bobParity {
			// Error is in first half
			end = mid
		} else {
			// Error is in second half
			start = mid
		}
	}

	return start, disclosedBits
}

// SimpleParityCorrector implements a simple parity-based error correction
// Less efficient than Cascade but simpler to understand and implement
type SimpleParityCorrector struct{}

// NewSimpleParityCorrector creates a new simple parity corrector
func NewSimpleParityCorrector() *SimpleParityCorrector {
	return &SimpleParityCorrector{}
}

// Correct performs simple parity-based error correction
func (s *SimpleParityCorrector) Correct(aliceKey, bobKey []quantum.Bit, blockSize int) ([]quantum.Bit, int, error) {
	if len(aliceKey) != len(bobKey) {
		return nil, 0, fmt.Errorf("keys must have the same length")
	}

	keyLength := len(aliceKey)
	corrected := make([]quantum.Bit, keyLength)
	copy(corrected, bobKey)

	totalDisclosedBits := 0
	numBlocks := (keyLength + blockSize - 1) / blockSize

	for i := 0; i < numBlocks; i++ {
		startIdx := i * blockSize
		endIdx := startIdx + blockSize
		if endIdx > keyLength {
			endIdx = keyLength
		}

		aliceBlock := aliceKey[startIdx:endIdx]
		bobBlock := corrected[startIdx:endIdx]

		aliceParity := CalculateParity(aliceBlock)
		bobParity := CalculateParity(bobBlock)

		totalDisclosedBits++ // Parity disclosure

		// If parities match, assume block is correct
		if aliceParity == bobParity {
			continue
		}

		// If parities differ and block size is 1, we found the error
		if len(aliceBlock) == 1 {
			corrected[startIdx] = aliceBlock[0]
			continue
		}

		// Otherwise, recursively divide the block
		// This is a simplified approach - production would use full Cascade
		for j := startIdx; j < endIdx; j++ {
			if aliceKey[j] != corrected[j] {
				corrected[j] = aliceKey[j]
				break // Fix first error found
			}
		}
	}

	return corrected, totalDisclosedBits, nil
}

// LDPCCorrector implements LDPC (Low-Density Parity-Check) error correction
// This is a placeholder for a more sophisticated error correction method
type LDPCCorrector struct {
	codeRate float64 // Code rate (k/n)
}

// NewLDPCCorrector creates a new LDPC corrector
func NewLDPCCorrector(codeRate float64) *LDPCCorrector {
	return &LDPCCorrector{
		codeRate: codeRate,
	}
}

// Correct performs LDPC error correction
// TODO: Implement full LDPC decoding algorithm
func (l *LDPCCorrector) Correct(aliceKey, bobKey []quantum.Bit) ([]quantum.Bit, int, error) {
	// Placeholder: In production, implement belief propagation algorithm
	// For now, fall back to Cascade
	cascade := NewCascadeCorrector(0.05)
	return cascade.Correct(aliceKey, bobKey)
}

// VerifyKeyCorrectness checks if Alice and Bob's keys match after error correction
func VerifyKeyCorrectness(aliceKey, bobKey []quantum.Bit) (bool, float64) {
	if len(aliceKey) != len(bobKey) {
		return false, 1.0
	}

	errors := 0
	for i := range aliceKey {
		if aliceKey[i] != bobKey[i] {
			errors++
		}
	}

	errorRate := float64(errors) / float64(len(aliceKey))
	return errors == 0, errorRate
}

// CalculateInformationLeakage calculates how much information was leaked during error correction
// According to Shannon's theorem, leaked information = disclosed bits
func CalculateInformationLeakage(disclosedBits int, keyLength int) float64 {
	if keyLength == 0 {
		return 0
	}
	return float64(disclosedBits) / float64(keyLength)
}
