package decoder

import "encoding/binary"

type View struct {
	buffer []byte
	cursor int
}

var enc = binary.LittleEndian

func (v *View) Uint32(at int) uint32 {
	return enc.Uint32(v.buffer[at:])
}
