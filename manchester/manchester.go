package manchester

import (
	"time"
)

type Signal func(bool)

type Edge uint8

const (
	Down Edge = iota
	Up
)

type Manchester struct {
	SignalT           time.Duration
	prevBit           bool
	lastPeriodStartNs int64
	Sensitivity       int64

	sleepF func()
}

func (m *Manchester) WriteBit(bit bool, writer Signal) {
	if bit {
		if m.prevBit {
			writer(false)
		}
		m.prevBit = true
		m.sleepF()
		writer(true)
		m.sleepF()
	} else {
		if !m.prevBit {
			writer(true)
		}
		m.prevBit = false
		m.sleepF()
		writer(false)
		m.sleepF()
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

const PrecisionNs = int64(time.Second / time.Nanosecond)

func NewManchesterDriver(transferSpeed int64) (m Manchester) {
	m.SignalT = time.Duration(PrecisionNs/transferSpeed/2) * time.Nanosecond
	if m.SignalT > time.Duration(500)*time.Microsecond {
		m.sleepF = func() {
			time.Sleep(m.SignalT)
		}
	} else {
		localSleepT := m.SignalT / 200
		m.sleepF = func() {
			start := time.Now().UnixNano()
			for {
				time.Sleep(localSleepT)
				diff := time.Now().UnixNano() - start
				if diff >= m.SignalT.Nanoseconds() {
					break
				}
			}
		}
	}
	m.lastPeriodStartNs = -1
	m.Sensitivity = int64(float64(m.SignalT) * 0.6)
	m.prevBit = false
	return m
}
