package qkd

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

// BB84Protocol implements the BB84 Quantum Key Distribution protocol
type BB84Protocol struct {
	backend         quantum.QuantumBackend
	keyLength       int
	qberThreshold   float64 // Quantum Bit Error Rate threshold (typically 11%)
	sampleSize      float64 // Fraction of key to sample for error checking (0.0-1.0)
}

// NewBB84Protocol creates a new BB84 protocol instance
func NewBB84Protocol(backend quantum.QuantumBackend, keyLength int) *BB84Protocol {
	return &BB84Protocol{
		backend:       backend,
		keyLength:     keyLength,
		qberThreshold: 0.11,  // 11% - theoretical maximum for secure QKD
		sampleSize:    0.10,  // Sample 10% of bits for error estimation
	}
}

// SetQBERThreshold sets a custom QBER threshold
func (bb *BB84Protocol) SetQBERThreshold(threshold float64) {
	bb.qberThreshold = threshold
}

// SetSampleSize sets the fraction of bits to sample for error checking
func (bb *BB84Protocol) SetSampleSize(size float64) {
	if size > 0 && size < 1 {
		bb.sampleSize = size
	}
}

// AliceSession represents Alice's side of the BB84 protocol
type AliceSession struct {
	Bits   []quantum.Bit
	Bases  []quantum.Basis
	Qubits []quantum.Qubit
	Key    []quantum.Bit
}

// BobSession represents Bob's side of the BB84 protocol
type BobSession struct {
	Bases        []quantum.Basis
	Measurements []quantum.MeasurementResult
	Key          []quantum.Bit
}

// KeyExchangeResult contains the result of BB84 key exchange
type KeyExchangeResult struct {
	Key           []byte
	RawKeyLength  int
	FinalKeyLength int
	QBER          float64
	Secure        bool
	Message       string
}

// AliceGenerateQubits - Step 1: Alice generates random bits and bases, then prepares qubits
func (bb *BB84Protocol) AliceGenerateQubits() (*AliceSession, error) {
	// Generate random bits and bases for transmission
	// We generate more bits than needed to account for key sifting
	transmissionLength := bb.keyLength * 4 // 4x oversampling for key sifting

	alice := &AliceSession{
		Bits:  quantum.GenerateRandomBits(transmissionLength),
		Bases: quantum.GenerateRandomBases(transmissionLength),
	}

	// Prepare qubits using the quantum backend
	qubits, err := bb.backend.PrepareAndSend(alice.Bits, alice.Bases)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare qubits: %w", err)
	}

	alice.Qubits = qubits

	return alice, nil
}

// BobMeasureQubits - Step 2: Bob receives qubits and measures them in random bases
func (bb *BB84Protocol) BobMeasureQubits(qubits []quantum.Qubit) (*BobSession, error) {
	// Bob generates his own random measurement bases
	bob := &BobSession{
		Bases: quantum.GenerateRandomBases(len(qubits)),
	}

	// Bob measures the qubits using his chosen bases
	measurements, err := bb.backend.ReceiveAndMeasure(qubits, bob.Bases)
	if err != nil {
		return nil, fmt.Errorf("failed to measure qubits: %w", err)
	}

	bob.Measurements = measurements

	return bob, nil
}

// SiftedKey represents the result of basis reconciliation
type SiftedKey struct {
	AliceKey []quantum.Bit
	BobKey   []quantum.Bit
	Indices  []int // Indices where bases matched
}

// BasisReconciliation - Step 3: Alice and Bob compare bases (public channel)
// Returns only the bits where Alice and Bob used the same basis
func (bb *BB84Protocol) BasisReconciliation(alice *AliceSession, bob *BobSession) (*SiftedKey, error) {
	if len(alice.Bases) != len(bob.Bases) {
		return nil, fmt.Errorf("alice and bob must have same number of bases")
	}

	sifted := &SiftedKey{
		AliceKey: make([]quantum.Bit, 0),
		BobKey:   make([]quantum.Bit, 0),
		Indices:  make([]int, 0),
	}

	// Compare bases and keep bits where bases match
	for i := 0; i < len(alice.Bases); i++ {
		if alice.Bases[i] == bob.Bases[i] {
			// Bases match - keep this bit
			sifted.AliceKey = append(sifted.AliceKey, alice.Bits[i])
			sifted.BobKey = append(sifted.BobKey, bob.Measurements[i].MeasuredBit)
			sifted.Indices = append(sifted.Indices, i)
		}
	}

	return sifted, nil
}

// EstimateQBER - Step 4: Estimate Quantum Bit Error Rate
// Alice and Bob sacrifice a random subset of their sifted key to check for errors
func (bb *BB84Protocol) EstimateQBER(sifted *SiftedKey) (float64, error) {
	if len(sifted.AliceKey) == 0 {
		return 0, fmt.Errorf("sifted key is empty")
	}

	// Calculate how many bits to sample
	sampleCount := int(float64(len(sifted.AliceKey)) * bb.sampleSize)
	if sampleCount < 1 {
		sampleCount = 1
	}
	if sampleCount > len(sifted.AliceKey) {
		sampleCount = len(sifted.AliceKey)
	}

	// Randomly select indices to sample (without replacement)
	sampledIndices := make(map[int]bool)
	for len(sampledIndices) < sampleCount {
		idx, err := cryptoRandInt(len(sifted.AliceKey))
		if err != nil {
			return 0, err
		}
		sampledIndices[idx] = true
	}

	// Compare sampled bits to calculate error rate
	errors := 0
	for idx := range sampledIndices {
		if sifted.AliceKey[idx] != sifted.BobKey[idx] {
			errors++
		}
	}

	qber := float64(errors) / float64(sampleCount)
	return qber, nil
}

// RemoveSampledBits removes the bits that were used for QBER estimation
func (bb *BB84Protocol) RemoveSampledBits(sifted *SiftedKey, sampledIndices []int) *SiftedKey {
	// Create a map for quick lookup
	toRemove := make(map[int]bool)
	for _, idx := range sampledIndices {
		toRemove[idx] = true
	}

	// Create new sifted key without sampled bits
	newSifted := &SiftedKey{
		AliceKey: make([]quantum.Bit, 0),
		BobKey:   make([]quantum.Bit, 0),
		Indices:  make([]int, 0),
	}

	for i := 0; i < len(sifted.AliceKey); i++ {
		if !toRemove[i] {
			newSifted.AliceKey = append(newSifted.AliceKey, sifted.AliceKey[i])
			newSifted.BobKey = append(newSifted.BobKey, sifted.BobKey[i])
			newSifted.Indices = append(newSifted.Indices, sifted.Indices[i])
		}
	}

	return newSifted
}

// PerformKeyExchange executes the complete BB84 protocol between Alice and Bob
func (bb *BB84Protocol) PerformKeyExchange() (*KeyExchangeResult, error) {
	result := &KeyExchangeResult{}

	// Step 1: Alice generates qubits
	alice, err := bb.AliceGenerateQubits()
	if err != nil {
		return nil, fmt.Errorf("alice qubit generation failed: %w", err)
	}

	// Step 2: Bob measures qubits
	bob, err := bb.BobMeasureQubits(alice.Qubits)
	if err != nil {
		return nil, fmt.Errorf("bob measurement failed: %w", err)
	}

	// Step 3: Basis reconciliation (key sifting)
	sifted, err := bb.BasisReconciliation(alice, bob)
	if err != nil {
		return nil, fmt.Errorf("basis reconciliation failed: %w", err)
	}

	result.RawKeyLength = len(sifted.AliceKey)

	if result.RawKeyLength == 0 {
		return nil, fmt.Errorf("no matching bases found - sifted key is empty")
	}

	// Step 4: Estimate QBER
	qber, err := bb.EstimateQBER(sifted)
	if err != nil {
		return nil, fmt.Errorf("QBER estimation failed: %w", err)
	}

	result.QBER = qber

	// Step 5: Security check
	if qber > bb.qberThreshold {
		result.Secure = false
		result.Message = fmt.Sprintf("INSECURE: QBER (%.2f%%) exceeds threshold (%.2f%%). Possible eavesdropping detected!",
			qber*100, bb.qberThreshold*100)
		return result, nil
	}

	// Step 6: Remove sampled bits (they've been publicly disclosed)
	sampleCount := int(float64(len(sifted.AliceKey)) * bb.sampleSize)
	sampledIndices := make([]int, 0)
	sampledMap := make(map[int]bool)

	for len(sampledMap) < sampleCount {
		idx, err := cryptoRandInt(len(sifted.AliceKey))
		if err != nil {
			return nil, err
		}
		if !sampledMap[idx] {
			sampledMap[idx] = true
			sampledIndices = append(sampledIndices, idx)
		}
	}

	finalSifted := bb.RemoveSampledBits(sifted, sampledIndices)

	// Check if we have enough key material
	if len(finalSifted.AliceKey) < bb.keyLength {
		result.Secure = false
		result.Message = fmt.Sprintf("Insufficient key material: got %d bits, need %d bits",
			len(finalSifted.AliceKey), bb.keyLength)
		return result, nil
	}

	// Truncate to desired key length
	alice.Key = finalSifted.AliceKey[:bb.keyLength]
	bob.Key = finalSifted.BobKey[:bb.keyLength]

	// Verify Alice and Bob have the same key
	keyMatch := true
	for i := 0; i < len(alice.Key); i++ {
		if alice.Key[i] != bob.Key[i] {
			keyMatch = false
			break
		}
	}

	if !keyMatch {
		result.Secure = false
		result.Message = "Key mismatch detected after sifting"
		return result, nil
	}

	// Convert bits to bytes
	result.Key = quantum.BitsToBytes(alice.Key)
	result.FinalKeyLength = len(alice.Key)
	result.Secure = true
	result.Message = fmt.Sprintf("Secure key generated successfully! QBER: %.2f%%", qber*100)

	return result, nil
}

// cryptoRandInt generates a cryptographically secure random integer in range [0, max)
func cryptoRandInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}

	return int(nBig.Int64()), nil
}
