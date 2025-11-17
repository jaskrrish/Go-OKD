package qkd

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaskrrish/Go-OKD/internal/models/qkd"
	"github.com/jaskrrish/Go-OKD/internal/qkd/crypto"
	"github.com/jaskrrish/Go-OKD/internal/qkd/quantum"
)

// SessionManager manages QKD sessions and orchestrates key generation
type SessionManager struct {
	sessions  map[uuid.UUID]*qkd.QKDSession
	keys      map[uuid.UUID]*qkd.QuantumKey
	mutex     sync.RWMutex
	backend   quantum.QuantumBackend
}

// NewSessionManager creates a new session manager
func NewSessionManager(backend quantum.QuantumBackend) *SessionManager {
	return &SessionManager{
		sessions: make(map[uuid.UUID]*qkd.QKDSession),
		keys:     make(map[uuid.UUID]*qkd.QuantumKey),
		backend:  backend,
	}
}

// CreateSession creates a new QKD session initiated by Alice
func (sm *SessionManager) CreateSession(req *qkd.SessionCreateRequest) (*qkd.QKDSession, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sessionID := uuid.New()
	now := time.Now()

	session := &qkd.QKDSession{
		SessionID:  sessionID,
		AliceID:    req.AliceID,
		Status:     qkd.SessionWaitingForBob,
		Backend:    req.Backend,
		KeyLength:  req.KeyLength,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Duration(req.TTLMinutes) * time.Minute),
	}

	sm.sessions[sessionID] = session

	return session, nil
}

// JoinSession allows Bob to join an existing session
func (sm *SessionManager) JoinSession(sessionID uuid.UUID, bobID string) (*qkd.QKDSession, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, qkd.ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		session.Status = qkd.SessionAborted
		return nil, qkd.ErrSessionExpired
	}

	if session.Status != qkd.SessionWaitingForBob {
		return nil, qkd.ErrSessionInProgress
	}

	session.BobID = bobID
	session.Status = qkd.SessionActive

	return session, nil
}

// ExecuteKeyExchange performs the complete BB84 key exchange for a session
func (sm *SessionManager) ExecuteKeyExchange(sessionID uuid.UUID) (*qkd.QuantumKey, error) {
	sm.mutex.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mutex.Unlock()
		return nil, qkd.ErrSessionNotFound
	}

	if session.Status != qkd.SessionActive {
		sm.mutex.Unlock()
		return nil, fmt.Errorf("session is not active")
	}

	session.Status = qkd.SessionInitiating
	sm.mutex.Unlock()

	// Create BB84 protocol instance
	bb84 := NewBB84Protocol(sm.backend, session.KeyLength)

	// Execute key exchange
	result, err := bb84.PerformKeyExchange()
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, 0, 0, 0, false, err.Error())
		return nil, fmt.Errorf("key exchange failed: %w", err)
	}

	// Update session with results
	sm.updateSessionStatus(
		sessionID,
		qkd.SessionCompleted,
		result.QBER,
		result.RawKeyLength,
		result.FinalKeyLength,
		result.Secure,
		result.Message,
	)

	// If key generation was not secure, don't store the key
	if !result.Secure {
		return nil, fmt.Errorf("key generation was not secure: %s", result.Message)
	}

	// Store the generated key
	keyID := uuid.New()
	now := time.Now()

	quantumKey := &qkd.QuantumKey{
		KeyID:       keyID,
		SessionID:   sessionID,
		KeyMaterial: result.Key,
		KeyLength:   result.FinalKeyLength,
		GeneratedAt: now,
		ExpiresAt:   now.Add(24 * time.Hour), // Keys expire after 24 hours
		IsActive:    true,
	}

	sm.mutex.Lock()
	sm.keys[keyID] = quantumKey
	sm.mutex.Unlock()

	return quantumKey, nil
}

// ExecuteKeyExchangeWithPostProcessing performs BB84 with error correction and privacy amplification
func (sm *SessionManager) ExecuteKeyExchangeWithPostProcessing(sessionID uuid.UUID) (*qkd.QuantumKey, error) {
	sm.mutex.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		sm.mutex.Unlock()
		return nil, qkd.ErrSessionNotFound
	}

	if session.Status != qkd.SessionActive {
		sm.mutex.Unlock()
		return nil, fmt.Errorf("session is not active")
	}

	session.Status = qkd.SessionInitiating
	sm.mutex.Unlock()

	// Step 1: BB84 Protocol
	bb84 := NewBB84Protocol(sm.backend, session.KeyLength*4) // Generate 4x for post-processing overhead

	// Generate qubits (Alice)
	alice, err := bb84.AliceGenerateQubits()
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// Measure qubits (Bob)
	bob, err := bb84.BobMeasureQubits(alice.qubits)
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// Basis reconciliation
	sifted, err := bb84.BasisReconciliation(alice, bob)
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, 0, 0, 0, false, err.Error())
		return nil, err
	}

	// Estimate QBER
	qber, err := bb84.EstimateQBER(sifted)
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, 0, 0, 0, false, err.Error())
		return nil, err
	}

	if qber > bb84.qberThreshold {
		msg := fmt.Sprintf("QBER too high: %.2f%% (threshold: %.2f%%)", qber*100, bb84.qberThreshold*100)
		sm.updateSessionStatus(sessionID, qkd.SessionAborted, qber, len(sifted.AliceKey), 0, false, msg)
		return nil, fmt.Errorf("%s", msg)
	}

	// Step 2: Error Correction
	corrector := crypto.NewCascadeCorrector(qber)
	bobCorrected, disclosedBits, err := corrector.Correct(sifted.AliceKey, sifted.BobKey)
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, qber, len(sifted.AliceKey), 0, false, err.Error())
		return nil, err
	}

	// Verify keys match after error correction
	keysMatch, errorRate := crypto.VerifyKeyCorrectness(sifted.AliceKey, bobCorrected)
	if !keysMatch {
		msg := fmt.Sprintf("Error correction failed: remaining error rate %.2f%%", errorRate*100)
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, qber, len(sifted.AliceKey), 0, false, msg)
		return nil, fmt.Errorf("%s", msg)
	}

	// Step 3: Privacy Amplification
	amplifier := crypto.NewPrivacyAmplifier(crypto.SHA3_256Method)

	// Calculate information leakage
	sampleBits := int(float64(len(sifted.AliceKey)) * bb84.sampleSize)
	totalLeakage := float64(sampleBits+disclosedBits) / float64(len(sifted.AliceKey))

	// Calculate maximum secure key length
	secureLength := crypto.CalculateSecureKeyLength(
		len(sifted.AliceKey),
		qber,
		disclosedBits,
		64, // security parameter
	)

	if secureLength < session.KeyLength {
		msg := fmt.Sprintf("Cannot generate requested key length: max secure length is %d bits", secureLength)
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, qber, len(sifted.AliceKey), secureLength, false, msg)
		return nil, fmt.Errorf("%s", msg)
	}

	// Perform privacy amplification
	finalKey, err := amplifier.Amplify(sifted.AliceKey, totalLeakage, session.KeyLength)
	if err != nil {
		sm.updateSessionStatus(sessionID, qkd.SessionFailed, qber, len(sifted.AliceKey), 0, false, err.Error())
		return nil, err
	}

	// Update session
	msg := fmt.Sprintf("Secure key generated! QBER: %.2f%%, Disclosed bits: %d", qber*100, disclosedBits)
	sm.updateSessionStatus(sessionID, qkd.SessionCompleted, qber, len(sifted.AliceKey), len(finalKey)*8, true, msg)

	// Store key
	keyID := uuid.New()
	now := time.Now()

	quantumKey := &qkd.QuantumKey{
		KeyID:       keyID,
		SessionID:   sessionID,
		KeyMaterial: finalKey,
		KeyLength:   len(finalKey) * 8,
		GeneratedAt: now,
		ExpiresAt:   now.Add(24 * time.Hour),
		IsActive:    true,
	}

	sm.mutex.Lock()
	sm.keys[keyID] = quantumKey
	sm.mutex.Unlock()

	return quantumKey, nil
}

// updateSessionStatus updates a session's status and metrics
func (sm *SessionManager) updateSessionStatus(sessionID uuid.UUID, status qkd.SessionStatus, qber float64, rawKeyLen, finalKeyLen int, secure bool, message string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.Status = status
		session.QBER = qber
		session.RawKeyLength = rawKeyLen
		session.FinalKeyLength = finalKeyLen
		session.IsSecure = secure
		session.Message = message

		if status == qkd.SessionCompleted || status == qkd.SessionFailed || status == qkd.SessionAborted {
			now := time.Now()
			session.CompletedAt = &now
		}
	}
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID uuid.UUID) (*qkd.QKDSession, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, qkd.ErrSessionNotFound
	}

	return session, nil
}

// GetKey retrieves a generated key by ID
func (sm *SessionManager) GetKey(keyID uuid.UUID, userID string) (*qkd.QuantumKey, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	key, exists := sm.keys[keyID]
	if !exists {
		return nil, qkd.ErrKeyNotFound
	}

	// Verify authorization (user must be Alice or Bob)
	session, exists := sm.sessions[key.SessionID]
	if !exists {
		return nil, qkd.ErrSessionNotFound
	}

	if session.AliceID != userID && session.BobID != userID {
		return nil, qkd.ErrUnauthorized
	}

	// Check if key has expired
	if time.Now().After(key.ExpiresAt) {
		key.IsActive = false
		return nil, qkd.ErrKeyExpired
	}

	return key, nil
}

// RevokeKey marks a key as inactive
func (sm *SessionManager) RevokeKey(keyID uuid.UUID) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	key, exists := sm.keys[keyID]
	if !exists {
		return qkd.ErrKeyNotFound
	}

	key.IsActive = false
	now := time.Now()
	key.UsedAt = &now

	return nil
}

// CleanupExpiredSessions removes expired sessions and keys
func (sm *SessionManager) CleanupExpiredSessions() int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	removed := 0

	// Cleanup expired sessions
	for id, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			delete(sm.sessions, id)
			removed++
		}
	}

	// Cleanup expired keys
	for id, key := range sm.keys {
		if now.After(key.ExpiresAt) {
			delete(sm.keys, id)
			removed++
		}
	}

	return removed
}
