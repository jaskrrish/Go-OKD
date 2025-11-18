package quantum

import (
	"fmt"
	"strings"
)

// QASMBuilder builds OpenQASM 2.0 circuits for QKD operations
type QASMBuilder struct {
	version     string
	includeStmt string
	registers   []string
	gates       []string
	measurements []string
}

// NewQASMBuilder creates a new OpenQASM circuit builder
func NewQASMBuilder(numQubits int, numClassical int) *QASMBuilder {
	builder := &QASMBuilder{
		version:     "OPENQASM 2.0;",
		includeStmt: "include \"qelib1.inc\";",
		registers:   make([]string, 0),
		gates:       make([]string, 0),
		measurements: make([]string, 0),
	}

	// Add quantum and classical registers
	builder.registers = append(builder.registers,
		fmt.Sprintf("qreg q[%d];", numQubits),
		fmt.Sprintf("creg c[%d];", numClassical),
	)

	return builder
}

// AddGate adds a quantum gate operation
func (b *QASMBuilder) AddGate(gate string) {
	b.gates = append(b.gates, gate)
}

// AddMeasurement adds a measurement operation
func (b *QASMBuilder) AddMeasurement(qubit int, classical int) {
	b.measurements = append(b.measurements,
		fmt.Sprintf("measure q[%d] -> c[%d];", qubit, classical))
}

// Build generates the complete QASM circuit string
func (b *QASMBuilder) Build() string {
	var circuit strings.Builder

	circuit.WriteString(b.version + "\n")
	circuit.WriteString(b.includeStmt + "\n")
	circuit.WriteString("\n")

	for _, reg := range b.registers {
		circuit.WriteString(reg + "\n")
	}
	circuit.WriteString("\n")

	for _, gate := range b.gates {
		circuit.WriteString(gate + "\n")
	}
	circuit.WriteString("\n")

	for _, meas := range b.measurements {
		circuit.WriteString(meas + "\n")
	}

	return circuit.String()
}

// BuildBB84AliceCircuit creates a circuit for Alice's qubit preparation
// Alice prepares qubits in either rectilinear or diagonal basis
func BuildBB84AliceCircuit(bits []Bit, bases []Basis) string {
	if len(bits) != len(bases) {
		panic("bits and bases must have the same length")
	}

	numQubits := len(bits)
	builder := NewQASMBuilder(numQubits, numQubits)

	for i := 0; i < numQubits; i++ {
		// Prepare qubit in computational basis (|0⟩ or |1⟩)
		if bits[i] == One {
			builder.AddGate(fmt.Sprintf("x q[%d];", i)) // Flip to |1⟩
		}

		// Apply basis rotation
		if bases[i] == DiagonalBasis {
			// Apply Hadamard to rotate to diagonal basis
			builder.AddGate(fmt.Sprintf("h q[%d];", i))
		}
		// Rectilinear basis: no additional gate needed
	}

	// Measure all qubits (for verification/simulation purposes)
	for i := 0; i < numQubits; i++ {
		builder.AddMeasurement(i, i)
	}

	return builder.Build()
}

// BuildBB84BobCircuit creates a circuit for Bob's qubit measurement
// Bob measures qubits in randomly chosen bases
func BuildBB84BobCircuit(numQubits int, bases []Basis) string {
	if len(bases) != numQubits {
		panic("bases length must match numQubits")
	}

	builder := NewQASMBuilder(numQubits, numQubits)

	for i := 0; i < numQubits; i++ {
		// Apply basis rotation before measurement
		if bases[i] == DiagonalBasis {
			// Apply Hadamard to measure in diagonal basis
			builder.AddGate(fmt.Sprintf("h q[%d];", i))
		}
		// Rectilinear basis: measure directly in computational basis

		// Measure qubit
		builder.AddMeasurement(i, i)
	}

	return builder.Build()
}

// BuildBB84CombinedCircuit creates a combined circuit for BB84 protocol
// This simulates the complete Alice->Bob transmission in one circuit
func BuildBB84CombinedCircuit(aliceBits []Bit, aliceBases []Basis, bobBases []Basis) string {
	if len(aliceBits) != len(aliceBases) || len(aliceBits) != len(bobBases) {
		panic("all input arrays must have the same length")
	}

	numQubits := len(aliceBits)
	builder := NewQASMBuilder(numQubits, numQubits)

	for i := 0; i < numQubits; i++ {
		// Alice's preparation
		if aliceBits[i] == One {
			builder.AddGate(fmt.Sprintf("x q[%d];", i))
		}

		if aliceBases[i] == DiagonalBasis {
			builder.AddGate(fmt.Sprintf("h q[%d];", i))
		}

		// Simulate quantum channel (in real hardware, qubits would be transmitted)
		// For simulation, we can add noise with depolarizing channel
		// This is done through backend noise models

		// Bob's measurement
		if bobBases[i] == DiagonalBasis {
			builder.AddGate(fmt.Sprintf("h q[%d];", i))
		}

		builder.AddMeasurement(i, i)
	}

	return builder.Build()
}

// BuildBB84PrepareCircuit creates a circuit for qubit preparation only (no measurement)
// Useful for real quantum hardware where measurement happens separately
func BuildBB84PrepareCircuit(bits []Bit, bases []Basis) string {
	if len(bits) != len(bases) {
		panic("bits and bases must have the same length")
	}

	numQubits := len(bits)
	builder := NewQASMBuilder(numQubits, numQubits)

	for i := 0; i < numQubits; i++ {
		if bits[i] == One {
			builder.AddGate(fmt.Sprintf("x q[%d];", i))
		}

		if bases[i] == DiagonalBasis {
			builder.AddGate(fmt.Sprintf("h q[%d];", i))
		}
	}

	// No measurements - qubits are prepared for transmission

	return builder.Build()
}

// BuildBB84MeasureCircuit creates a circuit for qubit measurement only
// Useful for real quantum hardware where preparation happened separately
func BuildBB84MeasureCircuit(numQubits int, bases []Basis) string {
	if len(bases) != numQubits {
		panic("bases length must match numQubits")
	}

	builder := NewQASMBuilder(numQubits, numQubits)

	for i := 0; i < numQubits; i++ {
		if bases[i] == DiagonalBasis {
			builder.AddGate(fmt.Sprintf("h q[%d];", i))
		}

		builder.AddMeasurement(i, i)
	}

	return builder.Build()
}

// BuildBellPairCircuit creates a Bell pair (EPR pair) circuit for testing
func BuildBellPairCircuit() string {
	builder := NewQASMBuilder(2, 2)

	// Create Bell state |Φ+⟩ = (|00⟩ + |11⟩)/√2
	builder.AddGate("h q[0];")     // Hadamard on first qubit
	builder.AddGate("cx q[0],q[1];") // CNOT with first as control

	// Measure both qubits
	builder.AddMeasurement(0, 0)
	builder.AddMeasurement(1, 1)

	return builder.Build()
}

// BuildGHZStateCircuit creates a GHZ state for testing entanglement
func BuildGHZStateCircuit(numQubits int) string {
	if numQubits < 2 {
		panic("GHZ state requires at least 2 qubits")
	}

	builder := NewQASMBuilder(numQubits, numQubits)

	// Create GHZ state: |0...0⟩ + |1...1⟩
	builder.AddGate("h q[0];") // Hadamard on first qubit

	// Apply CNOT gates to entangle all qubits
	for i := 1; i < numQubits; i++ {
		builder.AddGate(fmt.Sprintf("cx q[0],q[%d];", i))
	}

	// Measure all qubits
	for i := 0; i < numQubits; i++ {
		builder.AddMeasurement(i, i)
	}

	return builder.Build()
}

// ParseQASMResult parses measurement results from QASM execution
// Results are in the format "counts": {"0000": 512, "1111": 512}
func ParseQASMResult(counts map[string]int, numBits int) []Bit {
	// Find the most frequent outcome
	maxCount := 0
	maxOutcome := ""

	for outcome, count := range counts {
		if count > maxCount {
			maxCount = count
			maxOutcome = outcome
		}
	}

	// Convert outcome string to bits
	bits := make([]Bit, numBits)
	for i := 0; i < numBits && i < len(maxOutcome); i++ {
		if maxOutcome[i] == '1' {
			bits[i] = One
		} else {
			bits[i] = Zero
		}
	}

	return bits
}

// ParseQASMProbabilities calculates probabilities from measurement counts
func ParseQASMProbabilities(counts map[string]int) map[string]float64 {
	totalShots := 0
	for _, count := range counts {
		totalShots += count
	}

	probabilities := make(map[string]float64)
	for outcome, count := range counts {
		probabilities[outcome] = float64(count) / float64(totalShots)
	}

	return probabilities
}
