package manchester

import (
	"testing"
	"github.com/magiconair/properties/assert"
	"time"
	"fmt"
)

const Intv = 50000

func TestDecoding(t *testing.T) {
	m := NewManchesterDriver(10000, 1)
	var bfr uint16 = 0
	var w = func(bit bool) {
		bfr = bfr << 1
		if bit {
			bfr = bfr | 1
		}
	}
	var localTime int64 = 0
	var period2T = func() {
		localTime = localTime + Intv
	}
	var periodT = func() {
		localTime = localTime + Intv/2
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

func TestReadWrite(t *testing.T) {
	m := NewManchesterDriver(5, 1000000)
	fmt.Printf("T : %.2f seconds", m.signalT.Seconds())

	var recv uint8 = 0

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

	r := func() {
		for {
			select {
			case bit, ok := <-ch:
				if !ok {
					break
				}
				switch bit {
				case true:
					m.ReadBit(Up, time.Now().UnixNano(), reader)
				case false:
					m.ReadBit(Down, time.Now().UnixNano(), reader)
				}
			}
		}
	}

	go r()

	//  1  0   0  1
	//   __   _    _
	// _|  |_| |__|

	dataValue := 211

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

	assert.Equal(t, recv, uint8(dataValue))

}