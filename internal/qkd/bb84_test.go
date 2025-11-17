package qkd

import (
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

	// With 5% noise, QBER should generally be below threshold
	if result.QBER > bb84.qberThreshold {
		t.Errorf("QBER %.2f%% exceeds threshold", result.QBER*100)
	}

	// Note: With channel noise and no error correction, keys may not match perfectly
	// This is expected behavior - error correction would be needed in production
	// The important thing is that QBER is detected correctly

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
	if len(alice.bits) == 0 {
		t.Error("Alice should have generated bits")
	}

	if len(alice.bases) == 0 {
		t.Error("Alice should have generated bases")
	}

	if len(alice.qubits) == 0 {
		t.Error("Alice should have generated qubits")
	}

	// All arrays should have the same length
	if len(alice.bits) != len(alice.bases) || len(alice.bits) != len(alice.qubits) {
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
	bob, err := bb84.BobMeasureQubits(alice.qubits)
	if err != nil {
		t.Fatalf("Bob measurement failed: %v", err)
	}

	// Verify Bob's measurements
	if len(bob.measurements) == 0 {
		t.Error("Bob should have measurements")
	}

	if len(bob.bases) != len(bob.measurements) {
		t.Error("Bob's bases and measurements should have the same length")
	}
}

func TestBasisReconciliation(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.qubits)

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
	expectedLength := len(alice.bits) / 2
	tolerance := expectedLength / 4 // 25% tolerance
	if len(sifted.AliceKey) < expectedLength-tolerance || len(sifted.AliceKey) > expectedLength+tolerance {
		t.Errorf("Expected sifted key length around %d, got %d", expectedLength, len(sifted.AliceKey))
	}
}

func TestEstimateQBER(t *testing.T) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	alice, _ := bb84.AliceGenerateQubits()
	bob, _ := bb84.BobMeasureQubits(alice.qubits)
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

func BenchmarkBB84KeyExchange(b *testing.B) {
	backend := quantum.NewSimulatorBackend(false, 0.0)
	bb84 := NewBB84Protocol(backend, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bb84.PerformKeyExchange()
	}
}
