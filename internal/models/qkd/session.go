package qkd

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the current state of a QKD session
type SessionStatus string

const (
	SessionInitiating SessionStatus = "initiating"
	SessionWaitingForBob SessionStatus = "waiting_for_bob"
	SessionActive SessionStatus = "active"
	SessionCompleted SessionStatus = "completed"
	SessionAborted SessionStatus = "aborted"
	SessionFailed SessionStatus = "failed"
)

// QuantumBackendType represents the quantum computing backend being used
type QuantumBackendType string

const (
	BackendSimulator QuantumBackendType = "simulator"
	BackendQiskit QuantumBackendType = "qiskit"
	BackendBraket QuantumBackendType = "braket"
)

// QKDSession represents a quantum key distribution session between Alice and Bob
type QKDSession struct {
	SessionID       uuid.UUID          `json:"session_id"`
	AliceID         string             `json:"alice_id"`
	BobID           string             `json:"bob_id,omitempty"`
	Status          SessionStatus      `json:"status"`
	Backend         QuantumBackendType `json:"backend"`
	KeyLength       int                `json:"key_length"`
	QBER            float64            `json:"qber"`
	RawKeyLength    int                `json:"raw_key_length"`
	FinalKeyLength  int                `json:"final_key_length"`
	IsSecure        bool               `json:"is_secure"`
	Message         string             `json:"message,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	ExpiresAt       time.Time          `json:"expires_at"`
}

// QuantumKey represents a generated quantum key
type QuantumKey struct {
	KeyID           uuid.UUID  `json:"key_id"`
	SessionID       uuid.UUID  `json:"session_id"`
	KeyMaterial     []byte     `json:"-"` // Never expose in JSON
	KeyLength       int        `json:"key_length"`
	GeneratedAt     time.Time  `json:"generated_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	UsedAt          *time.Time `json:"used_at,omitempty"`
	IsActive        bool       `json:"is_active"`
}

// SessionCreateRequest represents a request to create a new QKD session
type SessionCreateRequest struct {
	AliceID    string             `json:"alice_id"`
	KeyLength  int                `json:"key_length"`
	Backend    QuantumBackendType `json:"backend,omitempty"`
	TTLMinutes int                `json:"ttl_minutes,omitempty"`
}

// SessionJoinRequest represents a request from Bob to join a session
type SessionJoinRequest struct {
	SessionID string `json:"session_id"`
	BobID     string `json:"bob_id"`
}

// SessionResponse represents the response when creating or querying a session
type SessionResponse struct {
	Session *QKDSession `json:"session"`
	Error   string      `json:"error,omitempty"`
}

// KeyResponse represents the response when requesting a generated key
type KeyResponse struct {
	KeyID      string    `json:"key_id"`
	SessionID  string    `json:"session_id"`
	KeyHex     string    `json:"key_hex,omitempty"` // Hex encoded key (only for initial retrieval)
	KeyLength  int       `json:"key_length"`
	ExpiresAt  time.Time `json:"expires_at"`
	Error      string    `json:"error,omitempty"`
}

// SessionMetrics represents metrics for a QKD session
type SessionMetrics struct {
	SessionID         uuid.UUID `json:"session_id"`
	TotalQubits       int       `json:"total_qubits"`
	SiftedKeyLength   int       `json:"sifted_key_length"`
	SiftingEfficiency float64   `json:"sifting_efficiency"`
	QBER              float64   `json:"qber"`
	ErrorsCorrected   int       `json:"errors_corrected"`
	DisclosedBits     int       `json:"disclosed_bits"`
	FinalKeyLength    int       `json:"final_key_length"`
	ProcessingTimeMs  int64     `json:"processing_time_ms"`
}

// Validate validates a session create request
func (r *SessionCreateRequest) Validate() error {
	if r.AliceID == "" {
		return ErrInvalidAliceID
	}

	if r.KeyLength < 128 || r.KeyLength > 4096 {
		return ErrInvalidKeyLength
	}

	// Set default backend if not specified
	if r.Backend == "" {
		r.Backend = BackendSimulator
	}

	// Set default TTL if not specified (24 hours)
	if r.TTLMinutes == 0 {
		r.TTLMinutes = 1440
	}

	if r.TTLMinutes < 1 || r.TTLMinutes > 10080 { // Max 7 days
		return ErrInvalidTTL
	}

	return nil
}

// Validate validates a session join request
func (r *SessionJoinRequest) Validate() error {
	if r.SessionID == "" {
		return ErrInvalidSessionID
	}

	if r.BobID == "" {
		return ErrInvalidBobID
	}

	return nil
}

// Custom errors
type QKDError struct {
	Message string
}

func (e *QKDError) Error() string {
	return e.Message
}

var (
	ErrInvalidAliceID    = &QKDError{"invalid Alice ID"}
	ErrInvalidBobID      = &QKDError{"invalid Bob ID"}
	ErrInvalidSessionID  = &QKDError{"invalid session ID"}
	ErrInvalidKeyLength  = &QKDError{"key length must be between 128 and 4096 bits"}
	ErrInvalidTTL        = &QKDError{"TTL must be between 1 and 10080 minutes"}
	ErrSessionNotFound   = &QKDError{"session not found"}
	ErrSessionExpired    = &QKDError{"session has expired"}
	ErrKeyNotFound       = &QKDError{"key not found"}
	ErrKeyExpired        = &QKDError{"key has expired"}
	ErrUnauthorized      = &QKDError{"unauthorized access"}
	ErrSessionInProgress = &QKDError{"session already in progress"}
)
