package manchester

import (
	"hash/crc32"
)

type Stage string

const (
	Preamble Stage = "preamble"
	Size     Stage = "size"
	Data     Stage = "data"
	Checksum Stage = "checksum"
	Done     Stage = "done"
)

type dataFrame struct {
	Preamble byte
	Size     byte
	Data     []byte
	Checksum uint32

	Stage Stage

	sizeBit     uint8
	dataBit     uint16
	dataBits    uint16
	checksumBit uint8
	updateBitF  updateBit
}

type updateBit func(df *dataFrame, bit bool) updateBit

const PreambleValue = 155

func dataF(df *dataFrame, bit bool) updateBit {
	idx := df.dataBit / 8
	df.Data[idx] = df.Data[idx] << 1
	if bit {
		df.Data[idx] = df.Data[idx] | 1
	}
	df.dataBit++
	if df.dataBit == df.dataBits {
		df.updateBitF = checksumF
		df.Stage = Checksum
	}
	return df.updateBitF
}

func doneF(df *dataFrame, _ bool) updateBit {
	return df.updateBitF
}

func checksumF(df *dataFrame, bit bool) updateBit {
	df.Checksum = df.Checksum << 1
	if bit {
		df.Checksum = df.Checksum | 1
	}
	df.checksumBit++
	if df.checksumBit == 32 {
		df.updateBitF = doneF
		df.Stage = Done
	}
	return df.updateBitF
}

func sizeF(df *dataFrame, bit bool) updateBit {
	df.Size = df.Size << 1
	if bit {
		df.Size = df.Size | 1
	}
	df.sizeBit++
	if df.sizeBit == 8 {
		df.Data = make([]byte, df.Size)
		df.dataBits = uint16(df.Size * 8)
		df.updateBitF = dataF
		df.Stage = Data
	}
	return df.updateBitF
}

func preambleF(df *dataFrame, bit bool) updateBit {
	df.Preamble = df.Preamble << 1
	if bit {
		df.Preamble = df.Preamble | 1
	}
	if df.Preamble == PreambleValue {
		df.updateBitF = sizeF
		df.Stage = Size
	}
	return df.updateBitF
}

func NewDataFrame() (r dataFrame) {
	r = dataFrame{
		Preamble:   0,
		Size:       0,
		Data:       nil,
		Checksum:   0,
		sizeBit:    0,
		dataBit:    0,
		updateBitF: preambleF,
		Stage:      Preamble,
	}
	return r
}

func BuildDataFrame(data []byte) dataFrame {
	checksum := crc32.ChecksumIEEE(data)
	df := NewDataFrame()
	df.Preamble = PreambleValue
	df.Size = byte(len(data))
	df.Data = make([]byte, len(data))
	copy(df.Data, data)
	df.Checksum = checksum
	df.Stage = Done
	return df
}

func (r *dataFrame) ReadBit(bit bool) bool {
	r.updateBitF(r, bit)
	if r.Stage == Done {
		return true
	} else {
		return false
	}
}

type BitWriter func(bool)

func (r *dataFrame) WriteFrame(f BitWriter) {
	size := r.Size + 6 // preamble + size byte + data + checksum
	dst := make([]byte, size)
	dst[0] = r.Preamble
	dst[1] = r.Size
	copy(dst[2:], r.Data)
	dst[size-4] = byte((r.Checksum & 0xFF000000) >> 24)
	dst[size-3] = byte((r.Checksum & 0xFF0000) >> 16)
	dst[size-2] = byte((r.Checksum & 0xFF00) >> 8)
	dst[size-1] = byte(r.Checksum & 0xFF)

	for _, v := range dst {
		for i := 7; i >= 0; i-- {
			f(v&byte(1<<uint(i)) > 0)
		}
	}
}
