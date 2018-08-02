package manchester

import (
	"time"
)

type (
	// Signal function type to send the bit to the transport level.
	Signal func(bool)
	// Edge integer type thar represents the value of the Edge. Could be Up or Down.
	Edge uint8
)

const (
	// Down falling edge
	Down Edge = iota
	// Up rising edge
	Up
	// PrecisionNs helper function to calculate the multiplier for the nanosecond precision.
	PrecisionNs = int64(time.Second / time.Nanosecond)
)

// Manchester the data driver for the manchester encoding function.
type Manchester struct {
	SignalT           time.Duration
	prevBit           bool
	lastPeriodStartNs int64
	Sensitivity       int64

	Sleep func()
}

// WriteBit writes the bit of the data with the given delay based on the previous bit value.
//
// 	bit the bit value (true = 1, false = 0)
// 	writer the function that writes the bit value to the underlying transport.
func (m *Manchester) WriteBit(bit bool, writer Signal) {
	if bit {
		if m.prevBit {
			writer(false)
		}
		m.prevBit = true
		m.Sleep()
		writer(true)
		m.Sleep()
	} else {
		if !m.prevBit {
			writer(true)
		}
		m.prevBit = false
		m.Sleep()
		writer(false)
		m.Sleep()
	}
}

func signalDuration(currentTimeNs int64, m *Manchester) int64 {
	if m.lastPeriodStartNs == -1 {
		return 0
	} else {
		return currentTimeNs - m.lastPeriodStartNs
	}
}

func intervalMultiplierRounded(m *Manchester, currentTimeNs int64) int {
	duration := signalDuration(currentTimeNs, m)
	ns := m.SignalT.Nanoseconds()
	dur := int(1 + (duration-m.Sensitivity)/ns)
	return dur
}

// ReadBit reads the bit based on the curent edge and current timing.
//
//	edge the detected Edge
//	currentTimeNs current time in nanoseconds
// 	callback the callback function to report the bit.
func (m *Manchester) ReadBit(edge Edge, currentTimeNs int64, callback Signal) {

	updater := func(s bool) {
		callback(s)
		m.lastPeriodStartNs = currentTimeNs - m.SignalT.Nanoseconds()
	}

	fsm := func(s bool) {
		if m.lastPeriodStartNs == -1 {
			updater(s)
		} else {
			interval := intervalMultiplierRounded(m, currentTimeNs)
			if interval == 2 {
				m.lastPeriodStartNs = currentTimeNs
			} else if interval == 1 || interval == 3 {
				updater(s)
			} else {
				m.lastPeriodStartNs = -1
			}
		}
	}

	switch edge {
	case Up:
		fsm(true)
	case Down:
		fsm(false)
	}
}

// NewManchesterDriver creates the instance of Manchester encoder.
//
//	transferSpeed the transfer speedm bytes per second.
func NewManchesterDriver(transferSpeed int64) *Manchester {
	m := Manchester{}
	m.SignalT = time.Duration(PrecisionNs/transferSpeed/2) * time.Nanosecond
	if m.SignalT > time.Duration(500)*time.Microsecond {
		m.Sleep = func() {
			time.Sleep(m.SignalT)
		}
	} else {
		localSleepT := m.SignalT / 10000
		m.Sleep = func() {
			start := time.Now().UnixNano()
			for {
				time.Sleep(localSleepT)
				if time.Now().UnixNano()-start >= m.SignalT.Nanoseconds() {
					break
				}
			}
		}
	}
	m.lastPeriodStartNs = -1
	m.Sensitivity = int64(float64(m.SignalT) * 0.6)
	m.prevBit = false
	return &m
}
