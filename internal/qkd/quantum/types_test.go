package quantum

import (
	"fmt"
	"testing"
)

// TestBasisString tests the String method for Basis types
func TestBasisString(t *testing.T) {
	tests := []struct {
		name     string
		basis    Basis
		expected string
	}{
		{"Rectilinear basis", RectilinearBasis, "Rectilinear(+)"},
		{"Diagonal basis", DiagonalBasis, "Diagonal(×)"},
		{"Invalid basis", Basis(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.basis.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestBitOperations tests basic bit operations
func TestBitOperations(t *testing.T) {
	t.Run("Bit constants", func(t *testing.T) {
		if Zero != 0 {
			t.Error("Zero should be 0")
		}
		if One != 1 {
			t.Error("One should be 1")
		}
	})

	t.Run("Bit XOR", func(t *testing.T) {
		if Zero^Zero != Zero {
			t.Error("0 XOR 0 should be 0")
		}
		if Zero^One != One {
			t.Error("0 XOR 1 should be 1")
		}
		if One^Zero != One {
			t.Error("1 XOR 0 should be 1")
		}
		if One^One != Zero {
			t.Error("1 XOR 1 should be 0")
		}
	})
}

// TestPrepareQubit tests qubit preparation
func TestPrepareQubit(t *testing.T) {
	tests := []struct {
		name  string
		bit   Bit
		basis Basis
	}{
		{"Prepare |0⟩ in rectilinear", Zero, RectilinearBasis},
		{"Prepare |1⟩ in rectilinear", One, RectilinearBasis},
		{"Prepare |+⟩ (0 in diagonal)", Zero, DiagonalBasis},
		{"Prepare |-⟩ (1 in diagonal)", One, DiagonalBasis},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qubit := PrepareQubit(tt.bit, tt.basis)

			if qubit.ClassicalValue != tt.bit {
				t.Errorf("expected bit %d, got %d", tt.bit, qubit.ClassicalValue)
			}
			if qubit.PreparationBasis != tt.basis {
				t.Errorf("expected basis %v, got %v", tt.basis, qubit.PreparationBasis)
			}
		})
	}
}

// TestMeasureQubit tests qubit measurement
func TestMeasureQubit(t *testing.T) {
	t.Run("Measurement in same basis (deterministic)", func(t *testing.T) {
		// When measuring in the same basis, result should be deterministic
		for _, bit := range []Bit{Zero, One} {
			for _, basis := range []Basis{RectilinearBasis, DiagonalBasis} {
				qubit := PrepareQubit(bit, basis)
				result := MeasureQubit(qubit, basis)

				if result.MeasuredBit != bit {
					t.Errorf("measuring %d in same basis %v: expected %d, got %d",
						bit, basis, bit, result.MeasuredBit)
				}
				if result.MeasurementBasis != basis {
					t.Errorf("expected measurement basis %v, got %v", basis, result.MeasurementBasis)
				}
			}
		}
	})

	t.Run("Measurement in different basis (probabilistic)", func(t *testing.T) {
		// When measuring in different basis, result is random (50/50)
		// We'll run many trials and check distribution
		trials := 1000
		zeros := 0
		ones := 0

		qubit := PrepareQubit(Zero, RectilinearBasis)
		for i := 0; i < trials; i++ {
			result := MeasureQubit(qubit, DiagonalBasis)
			if result.MeasuredBit == Zero {
				zeros++
			} else {
				ones++
			}
		}

		// Should be roughly 50/50 (allow 40-60% range)
		zeroPercent := float64(zeros) / float64(trials)
		if zeroPercent < 0.4 || zeroPercent > 0.6 {
			t.Errorf("different basis measurement distribution off: %.1f%% zeros (expected ~50%%)",
				zeroPercent*100)
		}
	})
}

// TestGenerateRandomBits tests random bit generation
func TestGenerateRandomBits(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Generate 0 bits", 0},
		{"Generate 1 bit", 1},
		{"Generate 10 bits", 10},
		{"Generate 100 bits", 100},
		{"Generate 1000 bits", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bits := GenerateRandomBits(tt.length)

			if len(bits) != tt.length {
				t.Errorf("expected %d bits, got %d", tt.length, len(bits))
			}

			// Check all values are 0 or 1
			for i, bit := range bits {
				if bit != Zero && bit != One {
					t.Errorf("bit at index %d has invalid value %d", i, bit)
				}
			}

			// For larger samples, check distribution is reasonable
			if tt.length >= 100 {
				ones := 0
				for _, bit := range bits {
					if bit == One {
						ones++
					}
				}
				onePercent := float64(ones) / float64(tt.length)
				if onePercent < 0.3 || onePercent > 0.7 {
					t.Errorf("distribution seems off: %.1f%% ones (expected ~50%%)",
						onePercent*100)
				}
			}
		})
	}
}

// TestGenerateRandomBases tests random basis generation
func TestGenerateRandomBases(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Generate 0 bases", 0},
		{"Generate 1 basis", 1},
		{"Generate 10 bases", 10},
		{"Generate 100 bases", 100},
		{"Generate 1000 bases", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bases := GenerateRandomBases(tt.length)

			if len(bases) != tt.length {
				t.Errorf("expected %d bases, got %d", tt.length, len(bases))
			}

			// Check all values are valid bases
			for i, basis := range bases {
				if basis != RectilinearBasis && basis != DiagonalBasis {
					t.Errorf("basis at index %d has invalid value %d", i, basis)
				}
			}

			// For larger samples, check distribution
			if tt.length >= 100 {
				rectilinear := 0
				for _, basis := range bases {
					if basis == RectilinearBasis {
						rectilinear++
					}
				}
				rectilinearPercent := float64(rectilinear) / float64(tt.length)
				if rectilinearPercent < 0.3 || rectilinearPercent > 0.7 {
					t.Errorf("distribution seems off: %.1f%% rectilinear (expected ~50%%)",
						rectilinearPercent*100)
				}
			}
		})
	}
}

// TestBitsToBytes tests bit-to-byte conversion
func TestBitsToBytes(t *testing.T) {
	tests := []struct {
		name     string
		bits     []Bit
		expected []byte
	}{
		{
			name:     "Empty",
			bits:     []Bit{},
			expected: []byte{},
		},
		{
			name:     "Single 0 bit",
			bits:     []Bit{Zero},
			expected: []byte{0x00},
		},
		{
			name:     "Single 1 bit",
			bits:     []Bit{One},
			expected: []byte{0x80},
		},
		{
			name:     "8 bits all zero",
			bits:     []Bit{Zero, Zero, Zero, Zero, Zero, Zero, Zero, Zero},
			expected: []byte{0x00},
		},
		{
			name:     "8 bits all one",
			bits:     []Bit{One, One, One, One, One, One, One, One},
			expected: []byte{0xFF},
		},
		{
			name:     "Pattern 10110001",
			bits:     []Bit{One, Zero, One, One, Zero, Zero, Zero, One},
			expected: []byte{0xB1},
		},
		{
			name:     "16 bits",
			bits:     []Bit{One, Zero, One, Zero, One, Zero, One, Zero, Zero, One, Zero, One, Zero, One, Zero, One},
			expected: []byte{0xAA, 0x55},
		},
		{
			name:     "Not full byte (5 bits)",
			bits:     []Bit{One, One, One, One, One},
			expected: []byte{0xF8},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BitsToBytes(tt.bits)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d bytes, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("byte %d: expected 0x%02X, got 0x%02X", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestBytesToBits tests byte-to-bit conversion
func TestBytesToBits(t *testing.T) {
	tests := []struct {
		name      string
		bytes     []byte
		bitLength int
		expected  []Bit
	}{
		{
			name:      "Empty",
			bytes:     []byte{},
			bitLength: 0,
			expected:  []Bit{},
		},
		{
			name:      "Single byte 0x00",
			bytes:     []byte{0x00},
			bitLength: 8,
			expected:  []Bit{Zero, Zero, Zero, Zero, Zero, Zero, Zero, Zero},
		},
		{
			name:      "Single byte 0xFF",
			bytes:     []byte{0xFF},
			bitLength: 8,
			expected:  []Bit{One, One, One, One, One, One, One, One},
		},
		{
			name:      "Pattern 0xB1",
			bytes:     []byte{0xB1},
			bitLength: 8,
			expected:  []Bit{One, Zero, One, One, Zero, Zero, Zero, One},
		},
		{
			name:      "Partial byte (5 bits from 0xF8)",
			bytes:     []byte{0xF8},
			bitLength: 5,
			expected:  []Bit{One, One, One, One, One},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToBits(tt.bytes, tt.bitLength)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d bits, got %d", len(tt.expected), len(result))
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("bit %d: expected %d, got %d", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

// TestBitsRoundTrip tests conversion roundtrip
func TestBitsRoundTrip(t *testing.T) {
	tests := []int{1, 8, 16, 32, 64, 128, 256}

	for _, bitLength := range tests {
		t.Run(fmt.Sprintf("%d bits", bitLength), func(t *testing.T) {
			original := GenerateRandomBits(bitLength)
			bytes := BitsToBytes(original)
			recovered := BytesToBits(bytes, bitLength)

			if len(recovered) != len(original) {
				t.Fatalf("length mismatch: original %d, recovered %d", len(original), len(recovered))
			}

			for i := range original {
				if original[i] != recovered[i] {
					t.Errorf("bit %d mismatch: original %d, recovered %d", i, original[i], recovered[i])
				}
			}
		})
	}
}

// TestCalculateBitError tests error rate calculation
func TestCalculateBitError(t *testing.T) {
	tests := []struct {
		name          string
		bits1         []Bit
		bits2         []Bit
		expectedError float64
		shouldError   bool
	}{
		{
			name:          "No errors (identical)",
			bits1:         []Bit{Zero, One, Zero, One},
			bits2:         []Bit{Zero, One, Zero, One},
			expectedError: 0.0,
			shouldError:   false,
		},
		{
			name:          "25% error",
			bits1:         []Bit{Zero, One, Zero, One},
			bits2:         []Bit{Zero, Zero, Zero, One},
			expectedError: 0.25,
			shouldError:   false,
		},
		{
			name:          "50% error",
			bits1:         []Bit{Zero, One, Zero, One},
			bits2:         []Bit{One, Zero, One, Zero},
			expectedError: 1.0,
			shouldError:   false,
		},
		{
			name:          "100% error",
			bits1:         []Bit{Zero, Zero, Zero, Zero},
			bits2:         []Bit{One, One, One, One},
			expectedError: 1.0,
			shouldError:   false,
		},
		{
			name:          "Length mismatch",
			bits1:         []Bit{Zero, One},
			bits2:         []Bit{Zero},
			expectedError: 0.0,
			shouldError:   true,
		},
		{
			name:          "Empty sequences",
			bits1:         []Bit{},
			bits2:         []Bit{},
			expectedError: 0.0,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorRate, err := CalculateBitError(tt.bits1, tt.bits2)

			if tt.shouldError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if errorRate != tt.expectedError {
				t.Errorf("expected error rate %.2f, got %.2f", tt.expectedError, errorRate)
			}
		})
	}
}

// TestQuantumChannel tests quantum channel simulation
func TestQuantumChannel(t *testing.T) {
	t.Run("Perfect channel (no noise)", func(t *testing.T) {
		channel := NewQuantumChannel(0.0, 0.0)

		if channel.NoiseLevel != 0.0 {
			t.Errorf("expected noise level 0.0, got %.2f", channel.NoiseLevel)
		}

		// Transmit qubits through perfect channel
		qubit := PrepareQubit(Zero, RectilinearBasis)
		transmitted := channel.Transmit(qubit)

		if transmitted.ClassicalValue != Zero {
			t.Error("perfect channel should preserve qubit state")
		}
	})

	t.Run("Noisy channel", func(t *testing.T) {
		channel := NewQuantumChannel(0.5, 0.0) // 50% noise

		flips := 0
		trials := 1000

		for i := 0; i < trials; i++ {
			qubit := PrepareQubit(Zero, RectilinearBasis)
			transmitted := channel.Transmit(qubit)

			if transmitted.ClassicalValue != qubit.ClassicalValue {
				flips++
			}
		}

		flipRate := float64(flips) / float64(trials)
		// Should be around 50% (allow 40-60%)
		if flipRate < 0.4 || flipRate > 0.6 {
			t.Errorf("noise rate off: expected ~0.5, got %.2f", flipRate)
		}
	})
}

// Benchmark tests
func BenchmarkPrepareQubit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		PrepareQubit(One, DiagonalBasis)
	}
}

func BenchmarkMeasureQubit(b *testing.B) {
	qubit := PrepareQubit(One, DiagonalBasis)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MeasureQubit(qubit, RectilinearBasis)
	}
}

func BenchmarkGenerateRandomBits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRandomBits(256)
	}
}

func BenchmarkBitsToBytes(b *testing.B) {
	bits := GenerateRandomBits(256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BitsToBytes(bits)
	}
}
