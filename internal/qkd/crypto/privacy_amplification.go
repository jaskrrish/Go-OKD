package crypto

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"

	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
	"golang.org/x/crypto/sha3"
)

// PrivacyAmplifier implements privacy amplification for QKD
// Privacy amplification removes any information that an eavesdropper might have

// AmplificationMethod defines the hash function used for privacy amplification
type AmplificationMethod string

const (
	// SHA256Method uses SHA-256 for privacy amplification
	SHA256Method AmplificationMethod = "SHA256"
	// SHA512Method uses SHA-512 for privacy amplification
	SHA512Method AmplificationMethod = "SHA512"
	// SHA3_256Method uses SHA3-256 for privacy amplification
	SHA3_256Method AmplificationMethod = "SHA3-256"
	// SHA3_512Method uses SHA3-512 for privacy amplification
	SHA3_512Method AmplificationMethod = "SHA3-512"
)

// PrivacyAmplifier performs privacy amplification on quantum keys
type PrivacyAmplifier struct {
	method AmplificationMethod
}

// NewPrivacyAmplifier creates a new privacy amplifier with specified method
func NewPrivacyAmplifier(method AmplificationMethod) *PrivacyAmplifier {
	return &PrivacyAmplifier{
		method: method,
	}
}

// Amplify performs privacy amplification to compress the key and remove eavesdropper knowledge
// Parameters:
//   - key: The reconciled key after error correction
//   - informationLeakage: Total information leaked (QBER sample + error correction bits)
//   - targetLength: Desired final key length in bits
func (pa *PrivacyAmplifier) Amplify(key []quantum.Bit, informationLeakage float64, targetLength int) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("input key is empty")
	}

	if targetLength <= 0 {
		return nil, fmt.Errorf("target length must be positive")
	}

	// Calculate secure key length using leftover hash lemma
	// Secure length = Original length - Information leakage - Security parameter
	securityParameter := 64 // bits (standard security parameter)
	leakedBits := int(informationLeakage * float64(len(key)))
	maxSecureLength := len(key) - leakedBits - securityParameter

	if maxSecureLength < targetLength {
		return nil, fmt.Errorf("cannot generate secure key of length %d: max secure length is %d bits",
			targetLength, maxSecureLength)
	}

	// Convert bits to bytes for hashing
	keyBytes := quantum.BitsToBytes(key)

	// Apply universal hash function (cryptographic hash as approximation)
	// If we need more bits, apply hash expansion
	finalKey := make([]byte, 0)
	counter := 0

	for len(finalKey)*8 < targetLength {
		h, _ := pa.getHasher()
		h.Write(keyBytes)
		h.Write([]byte(fmt.Sprintf("%d", counter)))
		hashResult := h.Sum(nil)
		finalKey = append(finalKey, hashResult...)
		counter++
	}

	// Truncate to exact target length
	targetBytes := (targetLength + 7) / 8
	if len(finalKey) > targetBytes {
		finalKey = finalKey[:targetBytes]
	}

	return finalKey, nil
}

// getHasher returns the appropriate hash function based on the amplification method
func (pa *PrivacyAmplifier) getHasher() (hash.Hash, error) {
	switch pa.method {
	case SHA256Method:
		return sha256.New(), nil
	case SHA512Method:
		return sha512.New(), nil
	case SHA3_256Method:
		return sha3.New256(), nil
	case SHA3_512Method:
		return sha3.New512(), nil
	default:
		return nil, fmt.Errorf("unknown amplification method: %s", pa.method)
	}
}

// TwoUniversalHash implements a 2-universal hash family for privacy amplification
// This is more theoretically sound than using cryptographic hashes
type TwoUniversalHash struct {
	a uint64 // Random coefficient a
	b uint64 // Random coefficient b
	p uint64 // Large prime number
}

// NewTwoUniversalHash creates a new 2-universal hash function
func NewTwoUniversalHash(seed1, seed2 uint64) *TwoUniversalHash {
	// Use a large prime (Mersenne prime 2^61 - 1)
	p := uint64(2305843009213693951)

	return &TwoUniversalHash{
		a: seed1 % p,
		b: seed2 % p,
		p: p,
	}
}

// Hash computes the hash of input x
// h(x) = (ax + b) mod p
func (tuh *TwoUniversalHash) Hash(x uint64) uint64 {
	return (tuh.a*x + tuh.b) % tuh.p
}

// AmplifyWithUniversalHash performs privacy amplification using 2-universal hashing
func (pa *PrivacyAmplifier) AmplifyWithUniversalHash(key []quantum.Bit, seed1, seed2 uint64, targetLength int) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("input key is empty")
	}

	hasher := NewTwoUniversalHash(seed1, seed2)

	// Convert key to uint64 chunks and hash each chunk
	keyBytes := quantum.BitsToBytes(key)
	result := make([]byte, 0)

	for i := 0; i < len(keyBytes); i += 8 {
		chunk := uint64(0)
		for j := 0; j < 8 && i+j < len(keyBytes); j++ {
			chunk |= uint64(keyBytes[i+j]) << (j * 8)
		}

		hashed := hasher.Hash(chunk)

		// Convert hash result to bytes
		for j := 0; j < 8; j++ {
			result = append(result, byte(hashed>>(j*8)))
		}
	}

	// Truncate to target length
	targetBytes := (targetLength + 7) / 8
	if len(result) > targetBytes {
		result = result[:targetBytes]
	}

	return result, nil
}

// CalculateSecureKeyLength calculates the maximum secure key length after privacy amplification
// Based on the leftover hash lemma
func CalculateSecureKeyLength(rawKeyLength int, qber float64, disclosedBits int, securityParameter int) int {
	// Information leaked to Eve:
	// 1. QBER estimation (sample bits)
	// 2. Error correction (disclosed parity bits)
	// 3. Theoretical upper bound from QBER

	// Shannon limit: leaked information ≈ h(QBER) * n
	// where h is binary entropy function
	shannonLeakage := binaryEntropy(qber) * float64(rawKeyLength)

	totalLeakage := int(shannonLeakage) + disclosedBits
	secureLength := rawKeyLength - totalLeakage - securityParameter

	if secureLength < 0 {
		return 0
	}

	return secureLength
}

// binaryEntropy calculates the binary entropy function H(x) = -x*log2(x) - (1-x)*log2(1-x)
func binaryEntropy(p float64) float64 {
	if p <= 0 || p >= 1 {
		return 0
	}

	// H(p) = -p*log2(p) - (1-p)*log2(1-p)
	log2 := func(x float64) float64 {
		if x <= 0 {
			return 0
		}
		return math_log2(x)
	}

	return -p*log2(p) - (1-p)*log2(1-p)
}

// math_log2 computes log base 2
func math_log2(x float64) float64 {
	// log2(x) = log(x) / log(2)
	// Using natural log
	const ln2 = 0.693147180559945309417232121458
	if x <= 0 {
		return 0
	}
	return math_log(x) / ln2
}

// math_log computes natural logarithm (basic implementation)
func math_log(x float64) float64 {
	// Using Taylor series for ln(x) around x=1
	// ln(x) = ln(1 + (x-1)) = sum(((-1)^(n+1) * (x-1)^n) / n)
	// For better convergence, we'll use a simple approximation

	if x <= 0 {
		return 0
	}

	// For simplicity, using a polynomial approximation
	// In production, use math.Log from standard library
	// This is just to avoid import for demonstration

	// Simple approximation: ln(x) ≈ 2 * ((x-1)/(x+1))
	// Better approximation would use more terms
	z := (x - 1) / (x + 1)
	return 2 * z * (1 + z*z/3 + z*z*z*z/5)
}
