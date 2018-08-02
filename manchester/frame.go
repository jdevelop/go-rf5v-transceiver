package manchester

import (
	"hash/crc32"
)

// Stage of the packet transformation.
type Stage string

const (
	// Preamble stage - expecting the preamble.
	Preamble Stage = "preamble"
	// Size stage - expecting size bytes.
	Size Stage = "size"
	// Data stage - reading data bytes.
	Data Stage = "data"
	// Checksum stage - the checksum is expected.
	Checksum Stage = "checksum"
	// Done - the packet is read successfully.
	Done Stage = "done"

	// PreambleValue the byte value to read from ether.
	PreambleValue = uint32(3078352847)
)

type (
	// DataFrame defines the data frame reader and data container.
	DataFrame struct {
		// Preamble the value of the current Preamble.
		Preamble uint32
		// Size the number of bytes in the data frame.
		Size byte
		// Data the data array.
		Data []byte
		// Checksum the checksum of the data slice.
		Checksum uint32

		Stage Stage

		sizeBit     uint8
		dataBit     uint16
		dataBits    uint16
		checksumBit uint8
		updateBitF  updateBit
	}

	updateBit func(df *DataFrame, bit bool) updateBit

	// BitWriter is the function to use for writing the signal. Could be used as transport.
	BitWriter func(bool)
)

var preambleValueBytes = []byte{
	byte((PreambleValue & 0xFF000000) >> 24),
	byte((PreambleValue & 0xFF0000) >> 16),
	byte((PreambleValue & 0xFF00) >> 8),
	byte(PreambleValue & 0xFF),
}

func dataF(df *DataFrame, bit bool) updateBit {
	idx := df.dataBit / 8
	if idx >= uint16(df.Size) {
		df.Reset()
	} else {
		df.Data[idx] = df.Data[idx] << 1
		if bit {
			df.Data[idx] = df.Data[idx] | 1
		}
		df.dataBit++
		if df.dataBit == df.dataBits {
			df.updateBitF = checksumF
			df.Stage = Checksum
		}
	}
	return df.updateBitF
}

func doneF(df *DataFrame, _ bool) updateBit {
	return df.updateBitF
}

func checksumF(df *DataFrame, bit bool) updateBit {
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

func sizeF(df *DataFrame, bit bool) updateBit {
	df.Size = df.Size << 1
	if bit {
		df.Size = df.Size | 1
	}
	df.sizeBit++
	if df.sizeBit == 8 {
		if df.Size <= 0 {
			df.Reset()
		} else {
			df.Data = make([]byte, df.Size)
			df.dataBits = uint16(df.Size * 8)
			df.updateBitF = dataF
			df.Stage = Data
		}
	}
	return df.updateBitF
}

func preambleF(df *DataFrame, bit bool) updateBit {
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

// NewDataFrame creates the data frame reader instance.
func NewDataFrame() *DataFrame {
	r := DataFrame{}
	r.Reset()
	return &r
}

// Reset resets the data frame reader to initial state.
func (df *DataFrame) Reset() {
	df.Preamble = 0
	df.Size = 0
	df.Data = nil
	df.Checksum = 0
	df.sizeBit = 0
	df.dataBit = 0
	df.dataBits = 0
	df.checksumBit = 0
	df.updateBitF = preambleF
	df.Stage = Preamble
}

// BuildDataFrame builds the data frame from the byte slice.
func BuildDataFrame(data []byte) *DataFrame {
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

// IsValid verifies that the data frame content matches the checksum.
func (df *DataFrame) IsValid() bool {
	checksum := crc32.ChecksumIEEE(df.Data)
	return checksum == df.Checksum
}

// ReadBit the callback function to update the state of the bit operation. Returns true when the data frame read is complete.
func (df *DataFrame) ReadBit(bit bool) bool {
	df.updateBitF(df, bit)
	return df.Stage == Done
}

// WriteFrame writes the data frame using the passed writer function as a transport.
func (df *DataFrame) WriteFrame(f BitWriter) {
	size := df.Size + 9 // preamble + size byte + data + checksum
	dst := make([]byte, size)
	copy(dst[0:], preambleValueBytes)
	dst[4] = df.Size
	copy(dst[5:], df.Data)
	dst[size-4] = byte((df.Checksum & 0xFF000000) >> 24)
	dst[size-3] = byte((df.Checksum & 0xFF0000) >> 16)
	dst[size-2] = byte((df.Checksum & 0xFF00) >> 8)
	dst[size-1] = byte(df.Checksum & 0xFF)

	for _, v := range dst {
		for i := 7; i >= 0; i-- {
			f(v&byte(1<<uint(i)) > 0)
		}
	}
}
