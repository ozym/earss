package ms

import (
	"fmt"

	"encoding/binary"
)

const (
	steimValuesPerFrame = 15

	steim1SpecialMask  = 0
	steim1ByteMask     = 1
	steim1HalfWordMask = 2
	steim1FullWordMask = 3

	steim2SpecialMask = 0
	steim2ByteMask    = 1
	steim2FrontMask   = 2
	steim2BackMask    = 3
)

type steimFrame [64]byte

func (f *steimFrame) Ctrl() uint32 {
	return binary.BigEndian.Uint32(f[0:])
}
func (f *steimFrame) SetCtrl(ctrl uint32) {
	binary.BigEndian.PutUint32(f[0:], ctrl)
}
func (f *steimFrame) PushCtrl(mask uint32) {
	f.SetCtrl((f.Ctrl() << 2) | mask)
}
func (f *steimFrame) SetFirst(val int32) {
	binary.BigEndian.PutUint32(f[4:], uint32(val))
}
func (f *steimFrame) SetLast(val int32) {
	binary.BigEndian.PutUint32(f[8:], uint32(val))
}
func (f *steimFrame) SetByte(i, j int, val int32) {
	f[4+i*4+j] = byte(val)
}
func (f *steimFrame) SetHalf(i, j int, val int32) {
	binary.BigEndian.PutUint16(f[4+i*4+2*j:], uint16(val))
}
func (f *steimFrame) SetFull(i int, val int32) {
	binary.BigEndian.PutUint32(f[4+i*4:], uint32(val))
}
func (f *steimFrame) Pack(i, bits, n int, m1, m2 uint32, diff [7]int32) {
	var val uint32
	for i := 0; i < n && i < 7; i++ {
		val = (val << bits) | (uint32(diff[i]) & m1)
	}
	val |= (m2 << 30)
	binary.BigEndian.PutUint32(f[4+i*4:], uint32(val))
}
func (f *steimFrame) Encode() []byte {
	blk := make([]byte, 64)
	copy(blk, f[:])
	return blk
}

func minPackBits(diff int32) int32 {
	switch {
	case diff >= -8 && diff < 8:
		return 4
	case diff >= -16 && diff < 16:
		return 5
	case (diff >= -32 && diff < 32):
		return 6
	case (diff >= -128 && diff < 128):
		return 8
	case (diff >= -512 && diff < 512):
		return 10
	case (diff >= -16384 && diff < 16384):
		return 15
	case (diff >= -32768 && diff < 32768):
		return 16
	case (diff >= -536870912 && diff < 536870912):
		return 30
	default:
		return 32
	}
}

func encodeSteim(version int, nf int, d0 int32, data []int32) ([]byte, int, int) {
	switch version {
	case 1:
		return encodeSteim1(nf, d0, data)
	case 2:
		return encodeSteim2(nf, d0, data)
	default:
		return nil, 0, 0
	}
}

func packSteim(version int, nf int, d0 int32, raw []int32, fn func([]byte, uint16, uint8) error) error {

	// make a copy to avoid hidden caller problems
	data := append([]int32(nil), raw...)

	for len(data) > 0 {
		res, ns, fs := encodeSteim(version, nf, d0, data)
		if fs < 0 {
			return fmt.Errorf("unable to represent difference in <= 30 bits")
		}
		if ns == 0 || fs == 0 {
			break
		}
		if err := fn(res, uint16(ns), uint8(fs)); err != nil {
			return err
		}
		d0, data = data[ns-1], data[ns:]
	}

	return nil
}

func encodeSteim1(nf int, d0 int32, data []int32) ([]byte, int, int) {
	// running counts
	var fn, wn, pn int

	// running differences
	var diff, minbits [4]int32

	// make space for expected frames
	fr := make([]steimFrame, nf)
	ns, pr := len(data), len(data)

	// calculate initial difference and minbits buffers /
	diff[0] = d0
	minbits[0] = minPackBits(diff[0])
	for i := 1; i < 4 && i < ns; i++ {
		diff[i] = data[i] - data[i-1]
		minbits[i] = minPackBits(diff[i])
	}

	// first and current last values
	fr[0].SetFirst(data[0])
	fr[0].PushCtrl(steim1SpecialMask)
	wn++

	fr[0].SetLast(data[ns-1])
	fr[0].PushCtrl(steim1SpecialMask)
	wn++

	for pr > 0 {
		var pp int

		var mask uint32
		switch {
		case (pr >= 4 && (minbits[0] <= 8) && (minbits[1] <= 8) && (minbits[2] <= 8) && (minbits[3] <= 8)):
			mask = steim1ByteMask
			for j := 0; j < 4; j++ {
				fr[fn].SetByte(wn, j, diff[j])
			}
			pp = 4
		case (pr >= 2 && (minbits[0] <= 16) && (minbits[1] <= 16)):
			mask = steim1HalfWordMask
			for j := 0; j < 2; j++ {
				fr[fn].SetHalf(wn, j, diff[j])
			}
			pp = 2
		default:
			mask = steim1FullWordMask
			fr[fn].SetFull(wn, diff[0])
			pp = 1
		}

		pn = pn + pp
		pr = pr - pp

		// push marker and update last value
		fr[fn].PushCtrl(mask)
		fr[0].SetLast(data[pn-1])

		// Check for full frame or full block
		if wn = wn + 1; wn == steimValuesPerFrame {
			// reset output index to beginning of frame
			wn = 0
			// if block is full, output block and reinitialize
			if fn = fn + 1; fn == nf {
				break
			}
			fr[fn].SetCtrl(0)
		}

		// shift and re-fill difference and minbits buffers
		for i := pp; i < 4; i++ {
			// shift remaining buffer entries
			diff[i-pp] = diff[i]
			minbits[i-pp] = minbits[i]
		}
		for i, j := 4-pp, pn+(4-pp); i < 4 && j < ns; i, j = i+1, j+1 {
			// re-fill entries
			diff[i] = data[j] - data[j-1]
			minbits[i] = minPackBits(diff[i])
		}
	}

	if wn < steimValuesPerFrame && fn < nf {
		for ; wn < steimValuesPerFrame; wn++ {
			fr[fn].PushCtrl(steim1SpecialMask)
			fr[fn].SetFull(wn, 0)
		}
		fn++
	}

	// convert frames into a byte slice
	var res []byte
	for i := 0; i < fn; i++ {
		res = append(res, fr[i].Encode()...)
	}

	return res, pn, fn
}

func encodeSteim2(nf int, d0 int32, data []int32) ([]byte, int, int) {
	// running counts
	var fn, wn, pn int

	// running differences
	var diff, minbits [7]int32

	// make space for expected frames
	fr := make([]steimFrame, nf)
	ns, pr := len(data), len(data)

	// calculate initial difference and minbits buffers /
	diff[0] = d0
	minbits[0] = minPackBits(diff[0])
	for i := 1; i < 7 && i < ns; i++ {
		diff[i] = data[i] - data[i-1]
		minbits[i] = minPackBits(diff[i])
	}

	// first and current last values
	fr[0].SetFirst(data[0])
	fr[0].PushCtrl(steim2SpecialMask)
	wn++

	fr[0].SetLast(data[ns-1])
	fr[0].PushCtrl(steim2SpecialMask)
	wn++

	for pr > 0 {
		var pp int

		var mask uint32
		switch {
		case pr >= 7 && (minbits[0] <= 4) && (minbits[1] <= 4) && (minbits[2] <= 4) && (minbits[3] <= 4) && (minbits[4] <= 4) && (minbits[5] <= 4) && (minbits[6] <= 4):
			mask = steim2BackMask
			fr[fn].Pack(wn, 4, 7, 0x0000000f, 02, diff)
			pp = 7
		case pr >= 6 && (minbits[0] <= 5) && (minbits[1] <= 5) && (minbits[2] <= 5) && (minbits[3] <= 5) && (minbits[4] <= 5) && (minbits[5] <= 5):
			mask = steim2BackMask
			fr[fn].Pack(wn, 5, 6, 0x0000001f, 01, diff)
			pp = 6
		case pr >= 5 && (minbits[0] <= 6) && (minbits[1] <= 6) && (minbits[2] <= 6) && (minbits[3] <= 6) && (minbits[4] <= 6):
			mask = steim2BackMask
			fr[fn].Pack(wn, 6, 5, 0x0000003f, 00, diff)
			pp = 5
		case pr >= 4 && (minbits[0] <= 8) && (minbits[1] <= 8) && (minbits[2] <= 8) && (minbits[3] <= 8):
			mask = steim2ByteMask
			for j := 0; j < 4; j++ {
				fr[fn].SetByte(wn, j, diff[j])
			}
			pp = 4
		case pr >= 3 && (minbits[0] <= 10) && (minbits[1] <= 10) && (minbits[2] <= 10):
			mask = steim2FrontMask
			fr[fn].Pack(wn, 10, 3, 0x000003ff, 03, diff)
			pp = 3
		case pr >= 2 && (minbits[0] <= 15) && (minbits[1] <= 15):
			mask = steim2FrontMask
			fr[fn].Pack(wn, 15, 2, 0x00007fff, 02, diff)
			pp = 2
		case pr >= 1 && (minbits[0] <= 30):
			mask = steim2FrontMask
			fr[fn].Pack(wn, 30, 1, 0x3fffffff, 01, diff)
			pp = 1
		default:
			return nil, 0, -1
		}

		pn = pn + pp
		pr = pr - pp

		// push marker and update last value
		fr[fn].PushCtrl(mask)
		fr[0].SetLast(data[pn-1])

		// Check for full frame or full block
		if wn = wn + 1; wn == steimValuesPerFrame {
			// reset output index to beginning of frame
			wn = 0
			// if block is full, output block and reinitialize
			if fn = fn + 1; fn == nf {
				break
			}
			fr[fn].SetCtrl(0)
		}

		// shift and re-fill difference and minbits buffers
		for i := pp; i < 7; i++ {
			// shift remaining buffer entries
			diff[i-pp] = diff[i]
			minbits[i-pp] = minbits[i]
		}
		for i, j := 7-pp, pn+(7-pp); i < 7 && j < ns; i, j = i+1, j+1 {
			// re-fill entries
			diff[i] = data[j] - data[j-1]
			minbits[i] = minPackBits(diff[i])
		}
	}

	if wn < steimValuesPerFrame && fn < nf {
		for ; wn < steimValuesPerFrame; wn++ {
			fr[fn].PushCtrl(steim2SpecialMask)
			fr[fn].SetFull(wn, 0)
		}
		fn++
	}

	// convert frames into a byte slice
	var res []byte
	for i := 0; i < nf; i++ {
		res = append(res, fr[i].Encode()...)
	}

	return res, pn, fn
}
