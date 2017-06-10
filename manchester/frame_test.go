package manchester

import (
	"github.com/stretchr/testify/assert"
	"hash/crc32"
	"testing"
)

func feedByte(f *dataFrame, src byte) {
	for i := 7; i > -1; i-- {
		f.ReadBit(src&byte(1<<uint(i)) > 0)
	}
}

var payload = []byte{'H', 'E', 'L', 'L', 'O'}

func TestFrameCreateRead(t *testing.T) {
	src := BuildDataFrame([]byte{'T', 'E', 'S', 'T'})
	dst := NewDataFrame()
	src.WriteFrame(func(bit bool) {
		dst.ReadBit(bit)
	})
	assert.Equal(t, src.Preamble, dst.Preamble)
	assert.Equal(t, src.Stage, dst.Stage)
	assert.Equal(t, src.Data, dst.Data)
	assert.Equal(t, src.Size, dst.Size)
	assert.Equal(t, src.Checksum, dst.Checksum)
}

func TestFrame(t *testing.T) {
	f := NewDataFrame()
	// read feed random noise
	feedByte(&f, 10)
	feedByte(&f, 0xfa)
	assert.Equal(t, f.Stage, Preamble)
	// feed preamble
	feedByte(&f, 155)
	assert.Equal(t, f.Stage, Size)
	// allocate 5 bytes of data
	feedByte(&f, 5)
	assert.Equal(t, f.Stage, Data)
	assert.Equal(t, byte(5), f.Size)
	feedByte(&f, 'H')
	feedByte(&f, 'E')
	feedByte(&f, 'L')
	feedByte(&f, 'L')
	feedByte(&f, 'O')
	assert.Equal(t, f.Stage, Checksum)
	assert.Equal(t, payload, f.Data)
	chks := crc32.ChecksumIEEE(payload)
	feedByte(&f, byte((chks&0xFF000000)>>24))
	feedByte(&f, byte((chks&0x00FF0000)>>16))
	feedByte(&f, byte((chks&0x0000FF00)>>8))
	feedByte(&f, byte(chks&0x000000FF))
	assert.Equal(t, f.Stage, Done)
	assert.Equal(t, chks, f.Checksum)
}