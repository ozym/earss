package ms

import (
	"encoding/binary"
	"math"
)

func encodeInt32(d int32, order uint8) []byte {
	var buf [4]byte
	switch order {
	case 0:
		binary.LittleEndian.PutUint32(buf[:], uint32(d))
	default:
		binary.BigEndian.PutUint32(buf[:], uint32(d))
	}
	return buf[:]
}

func encodeFloat32(f float32, order uint8) []byte {
	var buf [4]byte
	switch order {
	case 0:
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(f))
	default:
		binary.BigEndian.PutUint32(buf[:], math.Float32bits(f))
	}
	return buf[:]
}

func encodeFloat64(f float64, order uint8) []byte {
	var buf [8]byte
	switch order {
	case 0:
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
	default:
		binary.BigEndian.PutUint64(buf[:], math.Float64bits(f))
	}
	return buf[:]
}
