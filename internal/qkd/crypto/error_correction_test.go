package crypto

import (
	"testing"

	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

// TestCalculateParity tests the parity calculation function
func TestCalculateParity(t *testing.T) {
	tests := []struct {
		name     string
		bits     []quantum.Bit
		expected quantum.Bit
	}{
		{"Empty", []quantum.Bit{}, quantum.Zero},
		{"Single Zero", []quantum.Bit{quantum.Zero}, quantum.Zero},
		{"Single One", []quantum.Bit{quantum.One}, quantum.One},
		{"Two Zeros", []quantum.Bit{quantum.Zero, quantum.Zero}, quantum.Zero},
		{"Two Ones", []quantum.Bit{quantum.One, quantum.One}, quantum.Zero},
		{"One Zero One One", []quantum.Bit{quantum.One, quantum.Zero, quantum.One, quantum.One}, quantum.One},
		{"Four Ones", []quantum.Bit{quantum.One, quantum.One, quantum.One, quantum.One}, quantum.Zero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateParity(tt.bits)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestCascadeCorrector tests the Cascade error correction algorithm
func TestCascadeCorrector(t *testing.T) {
	t.Run("No errors", func(t *testing.T) {
		corrector := NewCascadeCorrector(0.0)

		alice := []quantum.Bit{quantum.Zero, quantum.One, quantum.Zero, quantum.One, quantum.Zero, quantum.One}
		bob := []quantum.Bit{quantum.Zero, quantum.One, quantum.Zero, quantum.One, quantum.Zero, quantum.One}

		corrected, disclosed, err := corrector.Correct(alice, bob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify no changes needed
		for i := range alice {
			if corrected[i] != alice[i] {
				t.Errorf("bit %d changed unnecessarily", i)
			}
		}

		// Some disclosure still happens (parity checks)
		if disclosed == 0 {
			t.Error("expected some bit disclosure even with no errors")
		}
	})

	t.Run("Single error correction", func(t *testing.T) {
		corrector := NewCascadeCorrector(0.05)

		alice := []quantum.Bit{quantum.Zero, quantum.One, quantum.Zero, quantum.One}
		bob := []quantum.Bit{quantum.Zero, quantum.Zero, quantum.Zero, quantum.One} // Error at index 1

		corrected, disclosed, err := corrector.Correct(alice, bob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify error was corrected
		for i := range alice {
			if corrected[i] != alice[i] {
				t.Errorf("bit %d: expected %d, got %d", i, alice[i], corrected[i])
			}
		}

		if disclosed == 0 {
			t.Error("expected bit disclosure")
		}
	})

	t.Run("Multiple errors correction", func(t *testing.T) {
		corrector := NewCascadeCorrector(0.10)

		// Create a key with 10% errors
		keyLength := 100
		alice := make([]quantum.Bit, keyLength)
		bob := make([]quantum.Bit, keyLength)

		for i := 0; i < keyLength; i++ {
			alice[i] = quantum.Bit(i % 2)
			bob[i] = alice[i]
		}

		// Introduce 10 errors
		errorIndices := []int{5, 15, 25, 35, 45, 55, 65, 75, 85, 95}
		for _, idx := range errorIndices {
			bob[idx] = 1 - bob[idx]
		}

		corrected, disclosed, err := corrector.Correct(alice, bob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify all errors were corrected
		errors := 0
		for i := range alice {
			if corrected[i] != alice[i] {
				errors++
			}
		}

		if errors > 0 {
			t.Errorf("still have %d errors after correction", errors)
		}

		t.Logf("Disclosed %d bits to correct 10 errors", disclosed)
	})

	t.Run("High error rate", func(t *testing.T) {
		corrector := NewCascadeCorrector(0.15)

		keyLength := 50
		alice := quantum.GenerateRandomBits(keyLength)
		bob := make([]quantum.Bit, keyLength)
		copy(bob, alice)

		// Introduce 15% errors (7-8 errors)
		errorCount := keyLength * 15 / 100
		for i := 0; i < errorCount; i++ {
			bob[i*keyLength/errorCount] = 1 - bob[i*keyLength/errorCount]
		}

		corrected, _, err := corrector.Correct(alice, bob)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify correction
		match, errorRate := VerifyKeyCorrectness(alice, corrected)
		if !match {
			t.Errorf("correction failed, remaining error rate: %.2f%%", errorRate*100)
		}
	})
}

// TestVerifyKeyCorrectness tests key verification
func TestVerifyKeyCorrectness(t *testing.T) {
	tests := []struct {
		name          string
		alice         []quantum.Bit
		bob           []quantum.Bit
		expectMatch   bool
		expectedError float64
	}{
		{
			"Perfect match",
			[]quantum.Bit{quantum.Zero, quantum.One, quantum.Zero, quantum.One},
			[]quantum.Bit{quantum.Zero, quantum.One, quantum.Zero, quantum.One},
			true,
			0.0,
		},
		{
			"One error",
			[]quantum.Bit{quantum.Zero, quantum.One, quantum.Zero, quantum.One},
			[]quantum.Bit{quantum.One, quantum.One, quantum.Zero, quantum.One},
			false,
			0.25,
		},
		{
			"All errors",
			[]quantum.Bit{quantum.Zero, quantum.Zero, quantum.Zero, quantum.Zero},
			[]quantum.Bit{quantum.One, quantum.One, quantum.One, quantum.One},
			false,
			1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, errorRate := VerifyKeyCorrectness(tt.alice, tt.bob)

			if match != tt.expectMatch {
				t.Errorf("expected match=%v, got %v", tt.expectMatch, match)
			}

			if errorRate != tt.expectedError {
				t.Errorf("expected error rate %.2f, got %.2f", tt.expectedError, errorRate)
			}
		})
	}
}

// TestCalculateInformationLeakage tests information leakage calculation
func TestCalculateInformationLeakage(t *testing.T) {
	tests := []struct {
		name          string
		disclosedBits int
		keyLength     int
		expected      float64
	}{
		{"No leakage", 0, 100, 0.0},
		{"10% leakage", 10, 100, 0.10},
		{"50% leakage", 50, 100, 0.50},
		{"Small key", 5, 10, 0.50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leakage := CalculateInformationLeakage(tt.disclosedBits, tt.keyLength)
			if leakage != tt.expected {
				t.Errorf("expected %.2f, got %.2f", tt.expected, leakage)
			}
		})
	}
}

// Benchmark error correction
func BenchmarkCascadeCorrect_NoErrors(b *testing.B) {
	corrector := NewCascadeCorrector(0.0)
	alice := quantum.GenerateRandomBits(256)
	bob := make([]quantum.Bit, 256)
	copy(bob, alice)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		corrector.Correct(alice, bob)
	}
}

func BenchmarkCascadeCorrect_5PercentErrors(b *testing.B) {
	corrector := NewCascadeCorrector(0.05)
	alice := quantum.GenerateRandomBits(256)
	bob := make([]quantum.Bit, 256)
	copy(bob, alice)

	// Introduce 5% errors
	for i := 0; i < 13; i++ {
		bob[i*20] = 1 - bob[i*20]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bobCopy := make([]quantum.Bit, 256)
		copy(bobCopy, bob)
		corrector.Correct(alice, bobCopy)
	}
}

func BenchmarkCalculateParity(b *testing.B) {
	bits := quantum.GenerateRandomBits(256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateParity(bits)
	}
}
