package crypto

import (
	"fmt"
	"testing"

	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

// TestPrivacyAmplifier tests privacy amplification
func TestPrivacyAmplifier(t *testing.T) {
	tests := []struct {
		name             string
		method           AmplificationMethod
		keyLength        int
		leakage          float64
		targetLength     int
		shouldSucceed    bool
	}{
		{
			"SHA256 amplification",
			SHA256Method,
			512,
			0.1,
			256,
			true,
		},
		{
			"SHA512 amplification",
			SHA512Method,
			1024,
			0.2,
			512,
			true,
		},
		{
			"SHA3-256 amplification",
			SHA3_256Method,
			512,
			0.15,
			256,
			true,
		},
		{
			"SHA3-512 amplification",
			SHA3_512Method,
			1024,
			0.1,
			512,
			true,
		},
		{
			"Insufficient key material",
			SHA256Method,
			100,
			0.5,  // 50% leakage
			256,  // Want 256 bits but only have ~50 bits secure
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amplifier := NewPrivacyAmplifier(tt.method)
			key := quantum.GenerateRandomBits(tt.keyLength)

			result, err := amplifier.Amplify(key, tt.leakage, tt.targetLength)

			if tt.shouldSucceed {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				expectedBytes := (tt.targetLength + 7) / 8
				if len(result) != expectedBytes {
					t.Errorf("expected %d bytes, got %d", expectedBytes, len(result))
				}
			} else {
				if err == nil {
					t.Error("expected error due to insufficient key material")
				}
			}
		})
	}
}

// TestAmplificationDeterminism tests that amplification is deterministic
func TestAmplificationDeterminism(t *testing.T) {
	amplifier := NewPrivacyAmplifier(SHA3_256Method)
	key := quantum.GenerateRandomBits(512)

	result1, err1 := amplifier.Amplify(key, 0.1, 256)
	result2, err2 := amplifier.Amplify(key, 0.1, 256)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}

	if len(result1) != len(result2) {
		t.Error("results have different lengths")
	}

	for i := range result1 {
		if result1[i] != result2[i] {
			t.Error("amplification is not deterministic")
			break
		}
	}
}

// TestCalculateSecureKeyLength tests secure key length calculation
func TestCalculateSecureKeyLength(t *testing.T) {
	tests := []struct {
		name              string
		rawKeyLength      int
		qber              float64
		disclosedBits     int
		securityParameter int
		minExpected       int
	}{
		{
			"Low QBER",
			1000,
			0.05,  // 5% QBER
			50,    // Disclosed bits
			64,    // Security parameter
			600,   // Expect > 600 bits secure (adjusted for Shannon limit)
		},
		{
			"Medium QBER",
			1000,
			0.10,  // 10% QBER
			100,   // Disclosed bits
			64,
			380,   // Expect > 380 bits secure (adjusted for Shannon limit)
		},
		{
			"High disclosure",
			1000,
			0.05,
			300,   // High disclosure
			64,
			380,   // Expect > 380 bits secure (adjusted for high disclosure)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secureLength := CalculateSecureKeyLength(
				tt.rawKeyLength,
				tt.qber,
				tt.disclosedBits,
				tt.securityParameter,
			)

			if secureLength < tt.minExpected {
				t.Errorf("secure length %d is less than minimum expected %d",
					secureLength, tt.minExpected)
			}

			t.Logf("Secure length: %d bits (%.1f%% of raw key)",
				secureLength,
				float64(secureLength)/float64(tt.rawKeyLength)*100)
		})
	}
}

// TestTwoUniversalHash tests 2-universal hash function
func TestTwoUniversalHash(t *testing.T) {
	t.Run("Deterministic hashing", func(t *testing.T) {
		hasher := NewTwoUniversalHash(123, 456)

		result1 := hasher.Hash(1000)
		result2 := hasher.Hash(1000)

		if result1 != result2 {
			t.Error("hash function not deterministic")
		}
	})

	t.Run("Different seeds produce different hashes", func(t *testing.T) {
		hasher1 := NewTwoUniversalHash(123, 456)
		hasher2 := NewTwoUniversalHash(789, 012)

		result1 := hasher1.Hash(1000)
		result2 := hasher2.Hash(1000)

		if result1 == result2 {
			t.Error("different seeds should produce different hashes")
		}
	})

	t.Run("Different inputs produce different hashes", func(t *testing.T) {
		hasher := NewTwoUniversalHash(123, 456)

		result1 := hasher.Hash(1000)
		result2 := hasher.Hash(2000)

		if result1 == result2 {
			t.Error("different inputs should produce different hashes")
		}
	})
}

// TestAmplifyWithUniversalHash tests 2-universal hash based amplification
func TestAmplifyWithUniversalHash(t *testing.T) {
	amplifier := NewPrivacyAmplifier(SHA3_256Method)
	key := quantum.GenerateRandomBits(512)

	result, err := amplifier.AmplifyWithUniversalHash(key, 12345, 67890, 256)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedBytes := (256 + 7) / 8
	if len(result) != expectedBytes {
		t.Errorf("expected %d bytes, got %d", expectedBytes, len(result))
	}
}

// TestBinaryEntropy tests the binary entropy function
func TestBinaryEntropy(t *testing.T) {
	tests := []struct {
		p        float64
		expected float64
		tolerance float64
	}{
		{0.0, 0.0, 0.01},
		{1.0, 0.0, 0.01},
		{0.5, 1.0, 0.01},  // Maximum entropy at 0.5
		{0.1, 0.47, 0.05},  // H(0.1) ≈ 0.47
		{0.25, 0.81, 0.05}, // H(0.25) ≈ 0.81
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("p=%.2f", tt.p), func(t *testing.T) {
			result := binaryEntropy(tt.p)

			if result < tt.expected-tt.tolerance || result > tt.expected+tt.tolerance {
				t.Errorf("H(%.2f) = %.2f, expected ≈%.2f", tt.p, result, tt.expected)
			}
		})
	}
}

// Benchmark privacy amplification
func BenchmarkAmplify_SHA256(b *testing.B) {
	amplifier := NewPrivacyAmplifier(SHA256Method)
	key := quantum.GenerateRandomBits(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		amplifier.Amplify(key, 0.1, 256)
	}
}

func BenchmarkAmplify_SHA3_256(b *testing.B) {
	amplifier := NewPrivacyAmplifier(SHA3_256Method)
	key := quantum.GenerateRandomBits(512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		amplifier.Amplify(key, 0.1, 256)
	}
}

func BenchmarkCalculateSecureKeyLength(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculateSecureKeyLength(1000, 0.08, 100, 64)
	}
}
