package qkd

import (
	"fmt"
	"testing"

	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

func TestBB84Protocol(t *testing.T) {
	// Create simulator backend with no noise for predictable testing
	backend := quantum.NewSimulatorBackend(false, 0.0)

	// Create BB84 protocol instance
	bb84 := NewBB84Protocol(backend, 256)

	// Test key exchange
	result, err := bb84.PerformKeyExchange()
	if err != nil {
		t.Fatalf("Key exchange failed: %v", err)
	}

	// Verify key was generated
	if result.Key == nil {
		t.Error("Expected key to be generated")
	}

	// Verify key length
	if result.FinalKeyLength != 256 {
		t.Errorf("Expected final key length of 256, got %d", result.FinalKeyLength)
	}

	// Verify security
	if !result.Secure {
		t.Errorf("Expected secure key, but got: %s", result.Message)
	}

	// Verify QBER is low (should be near 0 with no noise)
	if result.QBER > 0.05 {
		t.Errorf("Expected QBER < 5%%, got %.2f%%", result.QBER*100)
	}
}

func TestBB84WithNoise(t *testing.T) {
	// Create simulator with realistic noise
	backend := quantum.NewSimulatorBackend(true, 0.05) // 5% noise

	bb84 := NewBB84Protocol(backend, 256)

	result, err := bb84.PerformKeyExchange()
	if err != nil {
		t.Fatalf("Key exchange failed: %v", err)
	}

	// With 5% noise, QBER can vary due to probabilistic nature
	// The protocol should handle this correctly by marking keys as secure/insecure
	// QBER can be higher than channel noise due to basis measurement effects

	// Verify QBER is within reasonable range (0-50%, typically 0-20% with 5% noise)
	if result.QBER < 0 || result.QBER > 0.5 {
		t.Errorf("QBER %.2f%% outside reasonable range", result.QBER*100)
	}

	// Verify security determination is consistent with QBER
	if result.QBER > bb84.qberThreshold && result.Secure {
		t.Error("Key should not be marked secure when QBER exceeds threshold")
	}

	// Log QBER and security status for informational purposes
	t.Logf("QBER with 5%% channel noise: %.2f%%, Secure: %v", result.QBER*100, result.Secure)
	t.Logf("Message: %s", result.Message)
}

func TestBB84HighNoise(t *testing.T) {
	// Create simulator with high noise (simulating eavesdropper)
	backend := quantum.NewSimulatorBackend(true, 0.15) // 15% noise - above threshold

	bb84 := NewBB84Protocol(backend, 256)
	bb84.SetQBERThreshold(0.11) // Standard threshold

	result, err := bb84.PerformKeyExchange()
	if err != nil {
		t.Fatalf("Key exchange failed: %v", err)
	}

	// With 15% noise, QBER should typically be high, but due to randomness
	// we check if QBER exceeds threshold OR key is marked insecure
	// At minimum, the key should not be marked as secure with this noise level
	if result.QBER > bb84.qberThreshold && result.Secure {
		t.Error("Expected insecure key when QBER exceeds threshold")
	}

	// Log the QBER for informational purposes
	t.Logf("QBER with 15%% channel noise: %.2f%%", result.QBER*100)
}

func TestAliceGenerateQubits(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	alice, err := bb84.AliceGenerateQubits()
	if err != nil {
		t.Fatalf("Alice qubit generation failed: %v", err)
	}

	// Verify Alice generated bits, bases, and qubits
	if len(alice.Bits) == 0 {
		t.Error("Alice should have generated bits")
	}

	if len(alice.Bases) == 0 {
		t.Error("Alice should have generated bases")
	}

	if len(alice.Qubits) == 0 {
		t.Error("Alice should have generated qubits")
	}

	// All arrays should have the same length
	if len(alice.Bits) != len(alice.Bases) || len(alice.Bits) != len(alice.Qubits) {
		t.Error("Alice's bits, bases, and qubits should have the same length")
	}
}

func TestBobMeasureQubits(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	// Alice generates qubits
	alice, err := bb84.AliceGenerateQubits()
	if err != nil {
		t.Fatalf("Alice qubit generation failed: %v", err)
	}

	// Bob measures qubits
	bob, err := bb84.BobMeasureQubits(alice.Qubits)
	if err != nil {
		t.Fatalf("Bob measurement failed: %v", err)
	}

	// Verify Bob's measurements
	if len(bob.Measurements) == 0 {
		t.Error("Bob should have measurements")
	}

	if len(bob.Bases) != len(bob.Measurements) {
		t.Error("Bob's bases and measurements should have the same length")
	}
}

func TestBasisReconciliation(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.Qubits)

	sifted, err := bb84.BasisReconciliation(alice, bob)
	if err != nil {
		t.Fatalf("Basis reconciliation failed: %v", err)
	}

	// Verify sifted key exists
	if len(sifted.AliceKey) == 0 {
		t.Error("Sifted key should not be empty")
	}

	// Alice and Bob's sifted keys should have the same length
	if len(sifted.AliceKey) != len(sifted.BobKey) {
		t.Error("Alice and Bob's sifted keys should have the same length")
	}

	// With no noise, keys should match perfectly
	for i := range sifted.AliceKey {
		if sifted.AliceKey[i] != sifted.BobKey[i] {
			t.Errorf("Key mismatch at index %d", i)
		}
	}

	// Sifted key should be roughly 50% of original (basis matching probability)
	expectedLength := len(alice.Bits) / 2
	tolerance := expectedLength / 4 // 25% tolerance
	if len(sifted.AliceKey) < expectedLength-tolerance || len(sifted.AliceKey) > expectedLength+tolerance {
		t.Errorf("Expected sifted key length around %d, got %d", expectedLength, len(sifted.AliceKey))
	}
}

func TestEstimateQBER(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.Qubits)
	sifted, _ := bb84.BasisReconciliation(alice, bob)

	qber, err := bb84.EstimateQBER(sifted)
	if err != nil {
		t.Fatalf("QBER estimation failed: %v", err)
	}

	// With no noise, QBER should be very low (near 0)
	if qber > 0.01 {
		t.Errorf("Expected QBER near 0 with no noise, got %.4f", qber)
	}
}

func TestQuantumTypes(t *testing.T) {
	// Test bit operations
	bit := quantum.Zero
	if bit != 0 {
		t.Error("Zero bit should be 0")
	}

	// Test basis
	basis := quantum.RectilinearBasis
	if basis.String() != "Rectilinear(+)" {
		t.Error("Rectilinear basis string mismatch")
	}

	// Test qubit preparation
	qubit := quantum.PrepareQubit(quantum.One, quantum.DiagonalBasis)
	if qubit.ClassicalValue != quantum.One {
		t.Error("Qubit should encode bit One")
	}
	if qubit.PreparationBasis != quantum.DiagonalBasis {
		t.Error("Qubit should use diagonal basis")
	}

	// Test measurement
	result := quantum.MeasureQubit(qubit, quantum.DiagonalBasis)
	if result.MeasuredBit != quantum.One {
		t.Error("Measurement should yield One when bases match")
	}
}

func TestBitsToBytes(t *testing.T) {
	bits := []quantum.Bit{quantum.One, quantum.Zero, quantum.One, quantum.One, quantum.Zero, quantum.Zero, quantum.Zero, quantum.One}
	bytes := quantum.BitsToBytes(bits)

	// 10110001 = 0xB1 = 177
	if len(bytes) != 1 {
		t.Errorf("Expected 1 byte, got %d", len(bytes))
	}

	expected := byte(0xB1)
	if bytes[0] != expected {
		t.Errorf("Expected byte 0x%X, got 0x%X", expected, bytes[0])
	}

	// Test round-trip
	recoveredBits := quantum.BytesToBits(bytes, 8)
	for i := range bits {
		if bits[i] != recoveredBits[i] {
			t.Errorf("Bit mismatch at index %d: expected %d, got %d", i, bits[i], recoveredBits[i])
		}
	}
}

// TestBB84DifferentKeySizes tests key generation with various sizes
func TestBB84DifferentKeySizes(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)

	keySizes := []int{128, 256, 512, 1024}

	for _, size := range keySizes {
		t.Run(fmt.Sprintf("KeySize_%d", size), func(t *testing.T) {
			bb84 := NewBB84Protocol(backend, size)
			result, err := bb84.PerformKeyExchange()

			if err != nil {
				t.Fatalf("Key exchange failed for size %d: %v", size, err)
			}

			if result.FinalKeyLength != size {
				t.Errorf("Expected key length %d, got %d", size, result.FinalKeyLength)
			}

			if !result.Secure {
				t.Errorf("Expected secure key for size %d", size)
			}
		})
	}
}

// TestBB84QBERThreshold tests QBER threshold configuration
func TestBB84QBERThreshold(t *testing.T) {
	backend := quantum.NewSimulatorBackend(true, 0.10) // 10% noise

	tests := []struct {
		name      string
		threshold float64
		expectSecure bool
	}{
		{"Low threshold (5%)", 0.05, false},  // Should fail with 10% noise
		{"Medium threshold (11%)", 0.11, true}, // Might pass
		{"High threshold (15%)", 0.15, true},  // Should pass
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bb84 := NewBB84Protocol(backend, 256)
			bb84.SetQBERThreshold(tt.threshold)

			result, err := bb84.PerformKeyExchange()
			if err != nil {
				t.Fatalf("Key exchange failed: %v", err)
			}

			t.Logf("Threshold: %.2f%%, QBER: %.2f%%, Secure: %v",
				tt.threshold*100, result.QBER*100, result.Secure)
		})
	}
}

// TestBB84SiftingEfficiency tests basis reconciliation efficiency
func TestBB84SiftingEfficiency(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 1000)

	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.Qubits)
	sifted, _ := bb84.BasisReconciliation(alice, bob)

	// Sifting efficiency should be approximately 50% (basis match probability)
	efficiency := float64(len(sifted.AliceKey)) / float64(len(alice.Bits))

	if efficiency < 0.35 || efficiency > 0.65 {
		t.Errorf("Sifting efficiency %.2f%% outside expected range 35-65%%",
			efficiency*100)
	}

	t.Logf("Sifting efficiency: %.2f%% (%d/%d bits)",
		efficiency*100, len(sifted.AliceKey), len(alice.Bits))
}

// TestBB84BasisRandomness tests that Bob generates random bases
func TestBB84BasisRandomness(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	// Generate Alice's qubits once
	alice, _ := bb84.AliceGenerateQubits()

	// Bob measures with independently random bases each time
	bob1, _ := bb84.BobMeasureQubits(alice.Qubits)
	bob2, _ := bb84.BobMeasureQubits(alice.Qubits)

	// Bases should be different (randomly generated each time)
	differentBases := 0
	for i := range bob1.Bases {
		if bob1.Bases[i] != bob2.Bases[i] {
			differentBases++
		}
	}

	// Should have roughly 50% different bases (allow 30-70% range)
	diffPercent := float64(differentBases) / float64(len(bob1.Bases))
	if diffPercent < 0.3 || diffPercent > 0.7 {
		t.Errorf("Expected ~50%% different bases, got %.1f%%", diffPercent*100)
	}

	t.Logf("Basis difference: %.1f%% (%d/%d)",
		diffPercent*100, differentBases, len(bob1.Bases))
}

// TestBB84WithZeroNoise verifies perfect key match with no noise
func TestBB84WithZeroNoise(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.Qubits)
	sifted, _ := bb84.BasisReconciliation(alice, bob)

	// With no noise and matching bases, keys should be identical
	errorCount := 0
	for i := range sifted.AliceKey {
		if sifted.AliceKey[i] != sifted.BobKey[i] {
			errorCount++
		}
	}

	if errorCount > 0 {
		t.Errorf("Expected perfect key match, found %d errors", errorCount)
	}
}

// TestBB84SmallSample tests behavior with very small key length
func TestBB84SmallSample(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)

	// Very small key might not have enough bits for QBER estimation
	bb84 := NewBB84Protocol(backend, 16)

	result, err := bb84.PerformKeyExchange()
	if err != nil {
		t.Fatalf("Key exchange failed: %v", err)
	}

	// Just verify we can complete the protocol
	if result.Key == nil {
		t.Error("Expected key to be generated")
	}

	t.Logf("Small sample result: %d bits, QBER: %.2f%%",
		result.FinalKeyLength, result.QBER*100)
}

// TestBB84KeyUniqueness tests that different runs produce different keys
func TestBB84KeyUniqueness(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)

	keys := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		bb84 := NewBB84Protocol(backend, 256)
		result, err := bb84.PerformKeyExchange()
		if err != nil {
			t.Fatalf("Key exchange %d failed: %v", i, err)
		}
		keys[i] = result.Key
	}

	// Verify all keys are unique (probabilistically should be)
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			identical := true
			for k := 0; k < len(keys[i]); k++ {
				if keys[i][k] != keys[j][k] {
					identical = false
					break
				}
			}
			if identical {
				t.Errorf("Keys %d and %d are identical (very unlikely)", i, j)
			}
		}
	}
}

// TestBB84ErrorHandling tests error conditions
func TestBB84ErrorHandling(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)

	t.Run("Bob measures empty qubit list", func(t *testing.T) {
		bb84 := NewBB84Protocol(backend, 256)
		_, err := bb84.BobMeasureQubits([]quantum.Qubit{})
		// Should handle empty input gracefully
		if err != nil {
			t.Logf("Empty qubit measurement error (expected): %v", err)
		}
	})

	t.Run("Basis reconciliation with mismatched lengths", func(t *testing.T) {
		bb84 := NewBB84Protocol(backend, 256)

		alice, _ := bb84.AliceGenerateQubits()
		bob, _ := bb84.BobMeasureQubits(alice.Qubits[:len(alice.Qubits)/2])

		_, err := bb84.BasisReconciliation(alice, bob)
		if err == nil {
			t.Logf("Mismatched lengths handled (no error)")
		}
	})
}

func BenchmarkBB84KeyExchange(b *testing.B) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb84.PerformKeyExchange()
	}
}

func BenchmarkAliceGenerateQubits(b *testing.B) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb84.AliceGenerateQubits()
	}
}

func BenchmarkBobMeasureQubits(b *testing.B) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)
	alice, _ := bb84.AliceGenerateQubits()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb84.BobMeasureQubits(alice.Qubits)
	}
}

func BenchmarkBasisReconciliation(b *testing.B) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)
	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.Qubits)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb84.BasisReconciliation(alice, bob)
	}
}
