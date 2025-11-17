# Quantum Key Distribution (QKD) API Guide

## Overview

This API implements the **BB84 Quantum Key Distribution Protocol**, allowing Alice and Bob to generate secure shared cryptographic keys using quantum mechanics principles.

## Security Features

- ✅ **Information-theoretic security** - Security guaranteed by laws of physics
- ✅ **Eavesdropper detection** - Any interception attempt is detectable
- ✅ **Error correction** - Cascade algorithm for key reconciliation
- ✅ **Privacy amplification** - SHA3-based universal hashing
- ✅ **QBER monitoring** - Real-time quantum bit error rate tracking

## API Endpoints

### Base URL
```
http://localhost:8080/api/v1/qkd
```

---

### 1. Health Check

**GET** `/health`

Check if the QKD service is operational.

**Response:**
```json
{
  "status": "healthy",
  "service": "Quantum Key Distribution",
  "version": "1.0.0"
}
```

---

### 2. Initiate Session (Alice)

**POST** `/session/initiate`

Alice creates a new QKD session.

**Request Body:**
```json
{
  "alice_id": "alice@example.com",
  "key_length": 256,
  "backend": "simulator",
  "ttl_minutes": 60
}
```

**Parameters:**
- `alice_id` (required): Unique identifier for Alice
- `key_length` (required): Desired key length in bits (128-4096)
- `backend` (optional): Quantum backend - `simulator`, `qiskit`, or `braket` (default: `simulator`)
- `ttl_minutes` (optional): Session time-to-live in minutes (default: 1440 = 24 hours)

**Response (201 Created):**
```json
{
  "session": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "alice_id": "alice@example.com",
    "status": "waiting_for_bob",
    "backend": "simulator",
    "key_length": 256,
    "created_at": "2025-11-17T10:30:00Z",
    "expires_at": "2025-11-18T10:30:00Z"
  }
}
```

---

### 3. Join Session (Bob)

**POST** `/session/join`

Bob joins an existing QKD session.

**Request Body:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "bob_id": "bob@example.com"
}
```

**Response (200 OK):**
```json
{
  "session": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "alice_id": "alice@example.com",
    "bob_id": "bob@example.com",
    "status": "active",
    "backend": "simulator",
    "key_length": 256,
    "created_at": "2025-11-17T10:30:00Z",
    "expires_at": "2025-11-18T10:30:00Z"
  }
}
```

---

### 4. Execute Key Exchange

**POST** `/session/{session_id}/execute`

Executes the complete BB84 protocol including:
1. Quantum transmission (Alice → Bob)
2. Basis reconciliation
3. QBER estimation
4. Error correction (Cascade algorithm)
5. Privacy amplification (SHA3-256)

**Response (200 OK):**
```json
{
  "session": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "alice_id": "alice@example.com",
    "bob_id": "bob@example.com",
    "status": "completed",
    "backend": "simulator",
    "key_length": 256,
    "qber": 0.048,
    "raw_key_length": 512,
    "final_key_length": 256,
    "is_secure": true,
    "message": "Secure key generated! QBER: 4.80%, Disclosed bits: 51",
    "created_at": "2025-11-17T10:30:00Z",
    "completed_at": "2025-11-17T10:30:15Z",
    "expires_at": "2025-11-18T10:30:00Z"
  },
  "key_id": "660e8400-e29b-41d4-a716-446655440001",
  "message": "Quantum key generated successfully!"
}
```

**Error Response (if eavesdropper detected):**
```json
{
  "session": {
    "status": "aborted",
    "qber": 0.152,
    "is_secure": false,
    "message": "QBER too high: 15.20% (threshold: 11.00%)"
  },
  "error": "QBER too high: 15.20% (threshold: 11.00%)"
}
```

---

### 5. Get Session Info

**GET** `/session/{session_id}`

Retrieve information about a specific session.

**Headers:**
- `X-User-ID`: User identifier (Alice or Bob)

**Response (200 OK):**
```json
{
  "session": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "qber": 0.048,
    "final_key_length": 256,
    "is_secure": true
  }
}
```

---

### 6. Retrieve Quantum Key

**GET** `/key/{key_id}`

Retrieve a generated quantum key.

⚠️ **SECURITY**: Only Alice or Bob can retrieve their shared key.

**Headers:**
- `X-User-ID` (required): Must be Alice or Bob from the session

**Response (200 OK):**
```json
{
  "key_id": "660e8400-e29b-41d4-a716-446655440001",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "key_hex": "a3f5b8c2d9e6f1a4b7c8d2e5f9a1b4c7d8e2f5a8b1c4d7e9f2a5b8c1d4e7f9a2",
  "key_length": 256,
  "expires_at": "2025-11-18T10:30:15Z"
}
```

**Error Responses:**
- `401 Unauthorized`: Missing user authentication
- `403 Forbidden`: User is not authorized for this key
- `404 Not Found`: Key does not exist
- `410 Gone`: Key has expired

---

### 7. Revoke Key

**DELETE** `/key/{key_id}`

Revoke a quantum key (marks as inactive).

**Response (200 OK):**
```json
{
  "message": "Key revoked successfully"
}
```

---

## Complete Usage Example

### Using cURL

```bash
# 1. Alice initiates a session
curl -X POST http://localhost:8080/api/v1/qkd/session/initiate \
  -H "Content-Type: application/json" \
  -d '{
    "alice_id": "alice@example.com",
    "key_length": 256,
    "backend": "simulator"
  }'

# Save the session_id from response

# 2. Bob joins the session
curl -X POST http://localhost:8080/api/v1/qkd/session/join \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "bob_id": "bob@example.com"
  }'

# 3. Execute key exchange
curl -X POST http://localhost:8080/api/v1/qkd/session/550e8400-e29b-41d4-a716-446655440000/execute

# Save the key_id from response

# 4. Alice retrieves the key
curl -X GET http://localhost:8080/api/v1/qkd/key/660e8400-e29b-41d4-a716-446655440001 \
  -H "X-User-ID: alice@example.com"

# 5. Bob retrieves the same key
curl -X GET http://localhost:8080/api/v1/qkd/key/660e8400-e29b-41d4-a716-446655440001 \
  -H "X-User-ID: bob@example.com"

# Both will receive the same quantum key!
```

---

## BB84 Protocol Flow

```
Alice                          Quantum Channel                      Bob
  │                                   │                              │
  │ 1. Generate random bits & bases   │                              │
  │    [bits: 01101...]               │                              │
  │    [bases: +×+×+...]              │                              │
  │                                   │                              │
  │ 2. Encode into qubits             │                              │
  │    |0⟩, |+⟩, |1⟩, |-⟩...          │                              │
  │                                   │                              │
  │ 3. Send qubits ──────────────────>│───────────────────────────>  │
  │                                   │                              │
  │                                   │   4. Generate random bases   │
  │                                   │      [bases: +×××+...]       │
  │                                   │                              │
  │                                   │   5. Measure qubits          │
  │                                   │      [results: 01100...]     │
  │                                   │                              │
  │ 6. Compare bases (classical channel)                             │
  │ <─────────────────────────────────────────────────────────────> │
  │                                   │                              │
  │ 7. Keep bits where bases match (Key Sifting)                     │
  │    Matched indices: [0, 2, 4, ...]│                              │
  │    Sifted key: [0, 1, 1, ...]     │    Sifted key: [0, 1, 1...] │
  │                                   │                              │
  │ 8. Sample bits to estimate QBER                                  │
  │ <─────────────────────────────────────────────────────────────> │
  │    QBER = 4.8% ✓ (below 11% threshold)                           │
  │                                   │                              │
  │ 9. Error Correction (Cascade)                                    │
  │ <─────────────────────────────────────────────────────────────> │
  │    Exchange parity information                                   │
  │                                   │                              │
  │ 10. Privacy Amplification (SHA3)                                 │
  │    Apply universal hash function  │    Apply same hash function  │
  │    Final Key: [a3f5b8c2...]       │    Final Key: [a3f5b8c2...]  │
  │                                   │                              │
  │ ✓ Shared quantum key established! │                              │
```

---

## Security Considerations

### 1. QBER Threshold
- **Standard threshold**: 11%
- Below 11%: Secure key generation
- Above 11%: Possible eavesdropper - abort session

### 2. Key Expiration
- Default: 24 hours
- After expiration, keys are automatically deleted
- Use keys immediately after generation

### 3. Authentication
- In production, use **post-quantum signatures** (e.g., Dilithium)
- Current demo uses simple `X-User-ID` header
- Implement proper OAuth2/JWT authentication

### 4. Quantum Backends

#### Simulator (Development)
- Perfect for testing and development
- Configurable noise levels
- No real quantum hardware required

#### IBM Qiskit (Production)
- Real quantum hardware
- NISQ devices (Noisy Intermediate-Scale Quantum)
- ~2% error rate
- Requires IBM Quantum account

#### AWS Braket (Enterprise)
- Multiple quantum hardware providers
- IonQ, Rigetti, D-Wave support
- Reserved access available
- Requires AWS account

---

## Error Codes

| Code | Description |
|------|-------------|
| 400 | Invalid request parameters |
| 401 | Authentication required |
| 403 | Unauthorized access |
| 404 | Session or key not found |
| 410 | Key expired |
| 500 | Internal server error |

---

## Metrics & Monitoring

Each session generates metrics:

- **Total Qubits**: Number of qubits transmitted
- **Sifting Efficiency**: % of bits retained after basis reconciliation
- **QBER**: Quantum Bit Error Rate
- **Disclosed Bits**: Bits revealed during error correction
- **Final Key Length**: Length of secure key in bits
- **Processing Time**: Total time for key generation

---

## Best Practices

1. ✅ **Always check QBER** - Abort if > 11%
2. ✅ **Use keys immediately** - Don't store for long periods
3. ✅ **Implement key rotation** - Generate new keys regularly
4. ✅ **Monitor sessions** - Track failed attempts
5. ✅ **Use production backends** - Move to Qiskit/Braket for real security
6. ✅ **Enable audit logging** - Track all key generation events
7. ✅ **Implement rate limiting** - Prevent abuse

---

## Frequently Asked Questions

### Q: How long does key generation take?
**A:** Typically 1-5 seconds on simulator, 10-60 seconds on real quantum hardware.

### Q: What key lengths are supported?
**A:** 128-4096 bits. Recommended: 256 bits (AES-256 equivalent).

### Q: Is this post-quantum secure?
**A:** Yes! QKD provides information-theoretic security, not computational security. It's secure against all attacks, including quantum computers.

### Q: Can I reuse keys?
**A:** No! Keys should be used once (one-time pad) for perfect security.

### Q: What if QBER is too high?
**A:** Abort the session and try again. High QBER indicates eavesdropping or channel issues.

---

## Support

For issues or questions:
- GitHub Issues: [Go-OKD Repository](https://github.com/jaskrrish/Go-OKD)
- Documentation: `/docs/QKD_TECHNICAL_GUIDE.md`
- Examples: `/examples/qkd_demo.go`
