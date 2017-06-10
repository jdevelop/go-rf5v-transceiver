package manchester

import (
	"time"
)

type Signal func(bool)

type Edge int

const (
	Down = iota
	Up
)

type Manchester struct {
	signalT          time.Duration
	prevBit          bool
	lastEdgeChangeNs int64
	Sensitivity      int64
}

func (m *Manchester) WriteBit(bit bool, writer Signal) {
	if bit {
		if m.prevBit {
			writer(false)
		}
		time.Sleep(m.signalT)
		writer(true)
		time.Sleep(m.signalT)
		m.prevBit = true
	} else {
		if !m.prevBit {
			writer(true)
		}
		time.Sleep(m.signalT)
		writer(false)
		time.Sleep(m.signalT)
		m.prevBit = false
	}
}

func signalDuration(currentTimeNs int64, m *Manchester) int64 {
	if m.lastEdgeChangeNs == -1 {
		return 0
	} else {
		return currentTimeNs - m.lastEdgeChangeNs
	}
}

func intervalMultiplierRounded(m *Manchester, currentTimeNs int64) int {
	duration := signalDuration(currentTimeNs, m)
	ns := m.signalT.Nanoseconds()
	dur := int(1 + (duration-m.Sensitivity)/ns)
	return dur
}

func (m *Manchester) ReadBit(edge Edge, currentTimeNs int64, callback Signal) {
	interval := intervalMultiplierRounded(m, currentTimeNs)
	switch edge {
	case Up:
		if m.lastEdgeChangeNs == -1 || interval == 2 {
			callback(true)
			m.lastEdgeChangeNs = currentTimeNs
		}
	case Down:
		if m.lastEdgeChangeNs == -1 || interval == 2 {
			callback(false)
			m.lastEdgeChangeNs = currentTimeNs
		}
	}
}

func NewManchesterDriver(transferSpeed int64, sensitivity int64) (m Manchester) {
	m.signalT = time.Duration(1000000000/transferSpeed/4) * time.Nanosecond
	m.lastEdgeChangeNs = -1
	m.Sensitivity = sensitivity
	return m
}
