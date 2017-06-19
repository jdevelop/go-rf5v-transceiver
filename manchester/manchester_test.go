package manchester

import (
	"testing"
	"github.com/magiconair/properties/assert"
	"time"
	"fmt"
	"sync"
)

const bps = 1000

func TestDecoding(t *testing.T) {
	m := NewManchesterDriver(bps)
	var bfr uint16 = 0
	var w = func(bit bool) {
		bfr = bfr << 1
		if bit {
			bfr = bfr | 1
		}
	}
	var localTime int64 = 0
	var period2T = func() {
		localTime = localTime + int64(m.SignalT*2)
	}
	var periodT = func() {
		localTime = localTime + int64(m.SignalT)
	}

	// 0 1 0 0 1 1

	m.ReadBit(Down, localTime, w) // 0
	period2T()
	m.ReadBit(Up, localTime, w) // 1
	period2T()
	m.ReadBit(Down, localTime, w) // 0
	periodT()
	m.ReadBit(Up, localTime, w) // no value
	periodT()
	m.ReadBit(Down, localTime, w) // 0
	period2T()
	m.ReadBit(Up, localTime, w) // 1
	periodT()
	m.ReadBit(Down, localTime, w) // no value
	periodT()
	m.ReadBit(Up, localTime, w) // 1
	assert.Equal(t, bfr, uint16(19))
}

func TestReadWriteSimpleData(t *testing.T) {
	m := NewManchesterDriver(bps)
	r := NewManchesterDriver(bps)

	var recv int64 = 0

	var ch = make(chan bool)

	reader := func(bit bool) {
		recv = recv << 1
		if bit {
			recv = recv | 1
		}
	}

	writer := func(bit bool) {
		ch <- bit
	}

	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			select {
			case bit, ok := <-ch:
				if !ok {
					break
				}
				switch bit {
				case true:
					r.ReadBit(Up, time.Now().UnixNano(), reader)
				case false:
					r.ReadBit(Down, time.Now().UnixNano(), reader)
				}
			}
		}
	}()

	//  1  0   0  1
	//   __   _    _
	// _|  |_| |__|

	var dataValue int64 = time.Now().UnixNano()

	src := fmt.Sprintf("%b", dataValue)

	for _, v := range src {
		switch v {
		case '0':
			m.WriteBit(false, writer)
		case '1':
			m.WriteBit(true, writer)
		}
	}

	close(ch)
	wait.Wait()

	assert.Equal(t, recv, dataValue)

}

func TestReadWriteArray(t *testing.T) {
	w := NewManchesterDriver(bps)
	r := NewManchesterDriver(bps)

	recv := make([]byte, 17)

	bitIdx := 0

	var ch = make(chan bool)

	reader := func(bit bool) {
		idx := bitIdx / 8
		if bit {
			recv[idx] = recv[idx] | (1 << uint(bitIdx%8))
		}
		bitIdx++
	}

	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			bit, ok := <-ch
			dt := time.Now().UnixNano()
			if !ok {
				break
			}
			switch bit {
			case true:
				r.ReadBit(Up, dt, reader)
			case false:
				r.ReadBit(Down, dt, reader)
			}
		}
	}()

	src := []byte{155, 11, 72, 69, 76, 76, 79, 32, 87, 79, 82, 76, 68, 135, 229, 134, 91}

	for _, v := range src {
		for b := 0; b < 8; b++ {
			w.WriteBit(v&(1<<uint(b)) > 0, func(bit bool) {
				ch <- bit
			})
		}
	}

	close(ch)
	wait.Wait()

	assert.Equal(t, recv, src)

}

func TestReadFrameRaw(t *testing.T) {
	driverR := NewManchesterDriver(bps)

	driverW := NewManchesterDriver(bps)

	pinChannel := make(chan bool)

	testData := []byte{155, 11, 72, 69, 76, 76, 79, 32, 87, 79, 82, 76, 68, 135, 229, 134, 91}

	dest := NewDataFrame()

	frameBitReader := func(bit bool) {
		dest.ReadBit(bit)
	}

	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			bit, ok := <-pinChannel
			if !ok {
				break
			}
			dt := time.Now().UnixNano()
			switch bit {
			case true:
				driverR.ReadBit(Up, dt, frameBitReader)
			case false:
				driverR.ReadBit(Down, dt, frameBitReader)
			}
		}
	}()

	for _, v := range testData {
		for b := 7; b >= 0; b-- {
			driverW.WriteBit(v&(1<<uint(b)) > 0, func(bit bool) {
				pinChannel <- bit
			})
		}
	}

	close(pinChannel)
	wait.Wait()

	assert.Equal(t, dest.Preamble, byte(155), "Preamble failed")
	assert.Equal(t, dest.Size, byte(11), "Size failed")
	assert.Equal(t, dest.Data, []byte("HELLO WORLD"), "Data failed")
	assert.Equal(t, string(dest.Data), "HELLO WORLD", "Text failed")
	assert.Equal(t, dest.Checksum, uint32(2279966299), "Checksum failed")

}

func TestReadWriteFrame(t *testing.T) {
	driverR := NewManchesterDriver(bps)

	driverW := NewManchesterDriver(bps)

	type timedEvent struct {
		pin       bool
		timestamp int64
	}

	pinChannel := make(chan timedEvent)

	src := BuildDataFrame([]byte("AA"))
	dest := NewDataFrame()

	frameBitReader := func(bit bool) {
		dest.ReadBit(bit)
	}

	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		for {
			bit, ok := <-pinChannel
			if !ok {
				break
			}
			switch bit.pin {
			case true:
				driverR.ReadBit(Up, bit.timestamp, frameBitReader)
			case false:
				driverR.ReadBit(Down, bit.timestamp, frameBitReader)
			}
		}
	}()

	src.WriteFrame(func(v bool) {
		driverW.WriteBit(v, func(bit bool) {
			pinChannel <- timedEvent{pin: bit, timestamp: time.Now().UnixNano()}
		})
	})

	close(pinChannel)
	wait.Wait()

	assert.Equal(t, dest.Preamble, byte(155), "Preamble failed")
	assert.Equal(t, dest.Size, src.Size, "Size failed")
	assert.Equal(t, dest.Data, src.Data, "Data failed")
	assert.Equal(t, dest.Checksum, src.Checksum, "Checksum failed")

}
