package quantum

import (
	"fmt"
	"math/rand"
)

// Basis represents the measurement basis in BB84 protocol
type Basis int

const (
	// RectilinearBasis represents the computational basis (Z-basis): |0⟩, |1⟩
	RectilinearBasis Basis = 0
	// DiagonalBasis represents the Hadamard basis (X-basis): |+⟩, |−⟩
	DiagonalBasis Basis = 1
)

func (b Basis) String() string {
	switch b {
	case RectilinearBasis:
		return "Rectilinear(+)"
	case DiagonalBasis:
		return "Diagonal(×)"
	default:
		return "Unknown"
	}
}

// Bit represents a classical bit (0 or 1)
type Bit int

const (
	Zero Bit = 0
	One  Bit = 1
)

// Qubit represents a quantum bit state
type Qubit struct {
	// ClassicalValue is the bit value encoded in the qubit
	ClassicalValue Bit
	// PreparationBasis is the basis used to prepare this qubit
	PreparationBasis Basis
}

// MeasurementResult represents the outcome of measuring a qubit
type MeasurementResult struct {
	// MeasuredBit is the classical bit obtained from measurement
	MeasuredBit Bit
	// MeasurementBasis is the basis used for measurement
	MeasurementBasis Basis
}

// QuantumChannel represents a simulated quantum communication channel
type QuantumChannel struct {
	// NoiseLevel represents the probability of bit flip error (0.0 to 1.0)
	NoiseLevel float64
	// InterceptProbability simulates eavesdropper presence (0.0 to 1.0)
	InterceptProbability float64
}

// NewQuantumChannel creates a new quantum channel with specified noise characteristics
func NewQuantumChannel(noiseLevel, interceptProbability float64) *QuantumChannel {
	return &QuantumChannel{
		NoiseLevel:           noiseLevel,
		InterceptProbability: interceptProbability,
	}
}

// Transmit simulates transmission of a qubit through the quantum channel
func (qc *QuantumChannel) Transmit(qubit Qubit) Qubit {
	// Simulate eavesdropper interception
	if rand.Float64() < qc.InterceptProbability {
		// Eve intercepts and measures in random basis
		eveBasis := Basis(rand.Intn(2))
		// Eve's measurement collapses the state
		// If bases match, state is preserved; if not, it's disturbed
		if eveBasis != qubit.PreparationBasis {
			// 50% chance of bit flip when wrong basis is used
			if rand.Float64() < 0.5 {
				qubit.ClassicalValue = 1 - qubit.ClassicalValue
			}
		}
	}

	// Simulate channel noise (decoherence)
	if rand.Float64() < qc.NoiseLevel {
		qubit.ClassicalValue = 1 - qubit.ClassicalValue
	}

	return qubit
}

// PrepareQubit prepares a qubit in a specific state using the given basis
func PrepareQubit(bit Bit, basis Basis) Qubit {
	return Qubit{
		ClassicalValue:   bit,
		PreparationBasis: basis,
	}
}

// MeasureQubit simulates measuring a qubit in a specified basis
func MeasureQubit(qubit Qubit, measurementBasis Basis) MeasurementResult {
	measuredBit := qubit.ClassicalValue

	// If measurement basis doesn't match preparation basis,
	// outcome is random (50/50) due to quantum superposition
	if measurementBasis != qubit.PreparationBasis {
		if rand.Float64() < 0.5 {
			measuredBit = 1 - measuredBit
		}
	}

	return MeasurementResult{
		MeasuredBit:      measuredBit,
		MeasurementBasis: measurementBasis,
	}
}

// GenerateRandomBits generates a slice of random classical bits
func GenerateRandomBits(length int) []Bit {
	bits := make([]Bit, length)
	for i := 0; i < length; i++ {
		bits[i] = Bit(rand.Intn(2))
	}
	return bits
}

// GenerateRandomBases generates a slice of random measurement bases
func GenerateRandomBases(length int) []Basis {
	bases := make([]Basis, length)
	for i := 0; i < length; i++ {
		bases[i] = Basis(rand.Intn(2))
	}
	return bases
}

// BitsToBytes converts a slice of Bits to a byte array
func BitsToBytes(bits []Bit) []byte {
	numBytes := (len(bits) + 7) / 8
	bytes := make([]byte, numBytes)

	for i, bit := range bits {
		if bit == One {
			byteIndex := i / 8
			bitIndex := uint(7 - (i % 8))
			bytes[byteIndex] |= (1 << bitIndex)
		}
	}

	return bytes
}

// BytesToBits converts a byte array to a slice of Bits
func BytesToBits(bytes []byte, bitLength int) []Bit {
	bits := make([]Bit, bitLength)

	for i := 0; i < bitLength; i++ {
		byteIndex := i / 8
		bitIndex := uint(7 - (i % 8))
		if bytes[byteIndex]&(1<<bitIndex) != 0 {
			bits[i] = One
		} else {
			bits[i] = Zero
		}
	}

	return bits
}

// CalculateBitError calculates the error rate between two bit sequences
func CalculateBitError(bits1, bits2 []Bit) (float64, error) {
	if len(bits1) != len(bits2) {
		return 0, fmt.Errorf("bit sequences must have the same length")
	}

	if len(bits1) == 0 {
		return 0, nil
	}

	errors := 0
	for i := range bits1 {
		if bits1[i] != bits2[i] {
			errors++
		}
	}

	return float64(errors) / float64(len(bits1)), nil
}
