package quantum

import (
	"fmt"
	"math/rand"
	"time"
)

// QuantumBackend defines the interface for quantum computing backends
type QuantumBackend interface {
	// Name returns the name of the quantum backend
	Name() string

	// PrepareAndSend prepares qubits and sends them through the quantum channel
	PrepareAndSend(bits []Bit, bases []Basis) ([]Qubit, error)

	// ReceiveAndMeasure receives qubits and measures them in specified bases
	ReceiveAndMeasure(qubits []Qubit, bases []Basis) ([]MeasurementResult, error)

	// GetNoiseLevel returns the current noise level of the backend
	GetNoiseLevel() float64

	// IsSimulator returns true if this is a simulator, false for real hardware
	IsSimulator() bool
}

// SimulatorBackend implements a quantum simulator for development and testing
type SimulatorBackend struct {
	name           string
	channel        *QuantumChannel
	simulateNoise  bool
	noiseLevel     float64
}

// NewSimulatorBackend creates a new quantum simulator backend
func NewSimulatorBackend(simulateNoise bool, noiseLevel float64) *SimulatorBackend {
	return &SimulatorBackend{
		name:          "QuantumSimulator",
		channel:       NewQuantumChannel(noiseLevel, 0.0),
		simulateNoise: simulateNoise,
		noiseLevel:    noiseLevel,
	}
}

// Name returns the name of the simulator backend
func (s *SimulatorBackend) Name() string {
	return s.name
}

// PrepareAndSend prepares qubits according to BB84 protocol and simulates transmission
func (s *SimulatorBackend) PrepareAndSend(bits []Bit, bases []Basis) ([]Qubit, error) {
	if len(bits) != len(bases) {
		return nil, fmt.Errorf("bits and bases must have the same length")
	}

	qubits := make([]Qubit, len(bits))
	for i := range bits {
		// Prepare qubit in the specified basis
		qubits[i] = PrepareQubit(bits[i], bases[i])

		// Simulate transmission through quantum channel
		if s.simulateNoise {
			qubits[i] = s.channel.Transmit(qubits[i])
		}
	}

	return qubits, nil
}

// ReceiveAndMeasure simulates receiving qubits and measuring them
func (s *SimulatorBackend) ReceiveAndMeasure(qubits []Qubit, bases []Basis) ([]MeasurementResult, error) {
	if len(qubits) != len(bases) {
		return nil, fmt.Errorf("qubits and bases must have the same length")
	}

	results := make([]MeasurementResult, len(qubits))
	for i := range qubits {
		results[i] = MeasureQubit(qubits[i], bases[i])
	}

	return results, nil
}

// GetNoiseLevel returns the noise level of the simulator
func (s *SimulatorBackend) GetNoiseLevel() float64 {
	return s.noiseLevel
}

// IsSimulator returns true since this is a simulator
func (s *SimulatorBackend) IsSimulator() bool {
	return true
}

// QiskitBackend implements integration with IBM Qiskit Runtime
type QiskitBackend struct {
	name         string
	client       *QiskitClient
	backendName  string
	noiseLevel   float64
	shots        int
	useSimulator bool
	fallback     *SimulatorBackend // Fallback to local simulator on errors
}

// NewQiskitBackend creates a new Qiskit backend with REST API integration
func NewQiskitBackend(apiKey, backendName string, shots int) (*QiskitBackend, error) {
	config := &QiskitConfig{
		APIKey:      apiKey,
		BaseURL:     DefaultQiskitURL,
		BackendName: backendName,
	}

	client, err := NewQiskitClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qiskit client: %w", err)
	}

	// Determine if backend is simulator
	isSimulator := backendName == "ibmq_qasm_simulator" ||
	               backendName == "simulator_statevector" ||
	               backendName == "simulator_mps"

	// Create fallback simulator
	fallback := NewSimulatorBackend(true, 0.02)

	return &QiskitBackend{
		name:         "IBM-Qiskit-" + backendName,
		client:       client,
		backendName:  backendName,
		noiseLevel:   0.02, // Typical NISQ device error rate
		shots:        shots,
		useSimulator: isSimulator,
		fallback:     fallback,
	}, nil
}

// Name returns the name of the Qiskit backend
func (q *QiskitBackend) Name() string {
	return q.name
}

// PrepareAndSend prepares qubits using IBM Qiskit
func (q *QiskitBackend) PrepareAndSend(bits []Bit, bases []Basis) ([]Qubit, error) {
	if len(bits) != len(bases) {
		return nil, fmt.Errorf("bits and bases must have the same length")
	}

	// Build OpenQASM circuit for Alice's preparation
	qasmCode := BuildBB84AliceCircuit(bits, bases)

	circuit := &QiskitCircuit{
		QASM:    qasmCode,
		Shots:   q.shots,
		Backend: q.backendName,
	}

	// Submit job to IBM Quantum
	result, err := q.client.ExecuteCircuitSync(circuit, 5*time.Minute)
	if err != nil {
		// Fallback to local simulator on error
		fmt.Printf("Warning: Qiskit execution failed (%v), using local simulator\n", err)
		return q.fallback.PrepareAndSend(bits, bases)
	}

	// Parse results and construct qubits
	qubits := make([]Qubit, len(bits))
	if result.Success {
		// Extract measurement results from counts
		measuredBits := ParseQASMResult(result.Counts, len(bits))

		for i := range bits {
			qubits[i] = Qubit{
				ClassicalValue:   measuredBits[i],
				PreparationBasis: bases[i],
			}
		}
	} else {
		// If execution failed, use fallback
		return q.fallback.PrepareAndSend(bits, bases)
	}

	return qubits, nil
}

// ReceiveAndMeasure measures qubits using IBM Qiskit
func (q *QiskitBackend) ReceiveAndMeasure(qubits []Qubit, bases []Basis) ([]MeasurementResult, error) {
	if len(qubits) != len(bases) {
		return nil, fmt.Errorf("qubits and bases must have the same length")
	}

	// Extract Alice's bits and bases from qubits
	aliceBits := make([]Bit, len(qubits))
	aliceBases := make([]Basis, len(qubits))
	for i, q := range qubits {
		aliceBits[i] = q.ClassicalValue
		aliceBases[i] = q.PreparationBasis
	}

	// Build combined circuit for Alice preparation + Bob measurement
	qasmCode := BuildBB84CombinedCircuit(aliceBits, aliceBases, bases)

	circuit := &QiskitCircuit{
		QASM:    qasmCode,
		Shots:   q.shots,
		Backend: q.backendName,
	}

	// Execute circuit
	result, err := q.client.ExecuteCircuitSync(circuit, 5*time.Minute)
	if err != nil {
		// Fallback to local simulator
		fmt.Printf("Warning: Qiskit execution failed (%v), using local simulator\n", err)
		return q.fallback.ReceiveAndMeasure(qubits, bases)
	}

	// Parse measurement results
	measurements := make([]MeasurementResult, len(qubits))
	if result.Success {
		measuredBits := ParseQASMResult(result.Counts, len(qubits))

		for i := range qubits {
			measurements[i] = MeasurementResult{
				MeasuredBit:     measuredBits[i],
				MeasurementBasis: bases[i],
			}
		}
	} else {
		// Use fallback
		return q.fallback.ReceiveAndMeasure(qubits, bases)
	}

	return measurements, nil
}

// GetNoiseLevel returns the noise level of the Qiskit backend
func (q *QiskitBackend) GetNoiseLevel() float64 {
	return q.noiseLevel
}

// IsSimulator returns true if using IBM simulator, false for real hardware
func (q *QiskitBackend) IsSimulator() bool {
	return q.useSimulator
}

// BraketBackend implements integration with AWS Braket (placeholder)
type BraketBackend struct {
	name       string
	region     string
	deviceArn  string
	noiseLevel float64
}

// NewBraketBackend creates a new AWS Braket backend
// Note: This is a placeholder. Real implementation would use AWS SDK
func NewBraketBackend(region, deviceArn string) *BraketBackend {
	return &BraketBackend{
		name:       "AWS-Braket-" + deviceArn,
		region:     region,
		deviceArn:  deviceArn,
		noiseLevel: 0.015, // AWS Braket typical error rate
	}
}

// Name returns the name of the Braket backend
func (b *BraketBackend) Name() string {
	return b.name
}

// PrepareAndSend prepares qubits using AWS Braket
// TODO: Implement actual AWS Braket SDK integration
func (b *BraketBackend) PrepareAndSend(bits []Bit, bases []Basis) ([]Qubit, error) {
	if len(bits) != len(bases) {
		return nil, fmt.Errorf("bits and bases must have the same length")
	}

	// Placeholder implementation
	qubits := make([]Qubit, len(bits))
	for i := range bits {
		qubits[i] = PrepareQubit(bits[i], bases[i])

		if rand.Float64() < b.noiseLevel {
			qubits[i].ClassicalValue = 1 - qubits[i].ClassicalValue
		}
	}

	return qubits, nil
}

// ReceiveAndMeasure measures qubits using AWS Braket
// TODO: Implement actual AWS Braket SDK integration
func (b *BraketBackend) ReceiveAndMeasure(qubits []Qubit, bases []Basis) ([]MeasurementResult, error) {
	if len(qubits) != len(bases) {
		return nil, fmt.Errorf("qubits and bases must have the same length")
	}

	results := make([]MeasurementResult, len(qubits))
	for i := range qubits {
		results[i] = MeasureQubit(qubits[i], bases[i])
	}

	return results, nil
}

// GetNoiseLevel returns the noise level of the Braket backend
func (b *BraketBackend) GetNoiseLevel() float64 {
	return b.noiseLevel
}

// IsSimulator returns false for Braket
func (b *BraketBackend) IsSimulator() bool {
	return false
}
