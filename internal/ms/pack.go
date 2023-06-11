package ms

import (
	"bytes"
	"io"
	"strings"
	"time"
)

// RecordFunc is used as a callback when packing sample data.
type RecordFunc func(*Record) error

// NewEmptyRecord returns a Record pointer with the base required settings.
func NewEmptyRecord(reclen, factor, multi int) *Record {

	rec := Record{
		RecordHeader: RecordHeader{
			DataQualityIndicator:         'D',
			ReservedByte:                 ' ',
			SampleRateFactor:             int16(factor),
			SampleRateMultiplier:         int16(multi),
			NumberOfBlockettesThatFollow: 2,
			BeginningOfData:              64,
			FirstBlockette:               48,
		},
		B1000: Blockette1000{
			WordOrder:    uint8(BigEndian),
			RecordLength: uint8(reclen),
		},
	}

	return &rec
}

// EmptyRecord returns a Record pointer with the base required settings based on the current Record.
func (r *Record) EmptyRecord() *Record {
	return NewEmptyRecord(int(r.B1000.RecordLength), int(r.SampleRateFactor), int(r.SampleRateMultiplier))
}

// SetQualityIndication updated the record quality byte.
func (r *Record) SetQualityIndication(quality byte) {
	r.RecordHeader.DataQualityIndicator = quality
}

// SetBigEndian forces the Record to be big endian.
func (r *Record) SetBigEndian() {
	r.B1000.WordOrder = uint8(BigEndian)
}

// SetLittleEndian forces the Record to be little endian.
func (r *Record) SetLittleEndian() {
	r.B1000.WordOrder = uint8(LittleEndian)
}

// SetWordOrder forces the Record to be the given word order.
func (r *Record) SetWordOrder(order WordOrder) {
	r.B1000.WordOrder = uint8(order)
}

// IsBigEndian queries whether the Record is big endian.
func (r *Record) IsBigEndian() bool {
	return WordOrder(r.B1000.WordOrder) == BigEndian
}

// IsLittleEndian queries whether the Record is little endian.
func (r *Record) IsLittleEndian() bool {
	return WordOrder(r.B1000.WordOrder) == LittleEndian
}

// SetSampleRateFactor sets the Record rate factor, this will be samples per second if possitive, or seconds per sample if negative.
func (r *Record) SetSampleRateFactor(factor int) {
	r.RecordHeader.SampleRateFactor = int16(factor)
}

// SetSampleRateMultiplier sets the Record rate multiplier, this is used for sample rate factors that are not integers.
func (r *Record) SetSampleRateMultiplier(mult int) {
	r.RecordHeader.SampleRateMultiplier = int16(mult)
}

// PackASCII takes a string slice and packs it into miniseed Records which are passed to a callback function.
func (r *Record) PackASCII(start time.Time, raw []string, fn RecordFunc) error {

	samples := append([]string(nil), raw...)
	size := (r.BlockSize() - int(r.BeginningOfData))
	data := []byte(strings.Join(samples, "\n"))
	blocks := make([][]byte, 0, (len(samples)+size-1)/size)
	for size < len(data) {
		data, blocks = data[size:], append(blocks, data[0:size:size])
	}
	blocks = append(blocks, data)

	for _, b := range blocks {
		rec := Record{
			RecordHeader: r.RecordHeader,
			B1000:        r.B1000,
			B1001:        r.B1001,
			Data:         b,
		}

		rec.RecordHeader.RecordStartTime = NewBTime(start)
		rec.RecordHeader.NumberOfSamples = uint16(len(b))
		rec.B1000.Encoding = uint8(EncodingASCII)

		if err := fn(&rec); err != nil {
			return err
		}
	}

	return nil
}

// PackInt32 takes an int32 slice and packs it into miniseed Records which are passed to a callback function.
func (r *Record) PackInt32(start time.Time, raw []int32, fn RecordFunc) error {

	samples := append([]int32(nil), raw...)
	size := (r.BlockSize() - int(r.BeginningOfData)) / 4
	blocks := make([][]int32, 0, (len(samples)+size-1)/size)
	for size < len(samples) {
		samples, blocks = samples[size:], append(blocks, samples[0:size:size])
	}
	blocks = append(blocks, samples)

	var count int
	for _, b := range blocks {
		block := make([]byte, size*4)

		var ptr int
		for _, v := range b {
			copy(block[ptr:ptr+4], encodeInt32(v, uint8(r.ByteOrder())))
			ptr += 4
		}

		rec := Record{
			RecordHeader: r.RecordHeader,
			B1000:        r.B1000,
			B1001:        r.B1001,
			Data:         block,
		}

		offset := start.Add(time.Duration(count) * r.SamplePeriod())
		btime := NewBTime(offset)

		rec.RecordHeader.RecordStartTime = btime
		rec.RecordHeader.NumberOfSamples = uint16(len(b))
		rec.B1000.Encoding = uint8(EncodingInt32)
		rec.B1001.MicroSec = int8(offset.Sub(btime.Time()) / time.Microsecond)

		if err := fn(&rec); err != nil {
			return err
		}

		count += len(b)
	}

	return nil
}

// PackFloat32 takes an float32 slice and packs it into miniseed Records which are passed to a callback function.
func (r *Record) PackFloat32(start time.Time, raw []float32, fn RecordFunc) error {

	samples := append([]float32(nil), raw...)
	size := (r.BlockSize() - int(r.BeginningOfData)) / 4
	blocks := make([][]float32, 0, (len(samples)+size-1)/size)
	for size < len(samples) {
		samples, blocks = samples[size:], append(blocks, samples[0:size:size])
	}
	blocks = append(blocks, samples)

	var count int
	for _, b := range blocks {

		block := make([]byte, size*4)

		var ptr int
		for _, v := range b {
			copy(block[ptr:ptr+4], encodeFloat32(v, uint8(r.ByteOrder())))
			ptr += 4
		}

		rec := Record{
			RecordHeader: r.RecordHeader,
			B1000:        r.B1000,
			B1001:        r.B1001,
			Data:         block,
		}

		offset := start.Add(time.Duration(count) * r.SamplePeriod())
		btime := NewBTime(offset)

		rec.RecordHeader.RecordStartTime = btime
		rec.RecordHeader.NumberOfSamples = uint16(len(b))
		rec.B1000.Encoding = uint8(EncodingIEEEFloat)
		rec.B1001.MicroSec = int8(offset.Sub(btime.Time()) / time.Microsecond)

		if err := fn(&rec); err != nil {
			return err
		}

		count += len(b)
	}

	return nil
}

// PackFloat64 takes an float64 slice and packs it into miniseed Records which are passed to a callback function.
func (r *Record) PackFloat64(start time.Time, raw []float64, fn RecordFunc) error {

	samples := append([]float64(nil), raw...)
	size := (r.BlockSize() - int(r.BeginningOfData)) / 8
	blocks := make([][]float64, 0, (len(samples)+size-1)/size)
	for size < len(samples) {
		samples, blocks = samples[size:], append(blocks, samples[0:size:size])
	}
	blocks = append(blocks, samples)

	var count int
	for _, b := range blocks {

		block := make([]byte, size*8)

		var ptr int
		for _, v := range b {
			copy(block[ptr:ptr+8], encodeFloat64(v, uint8(r.ByteOrder())))
			ptr += 8
		}

		rec := Record{
			RecordHeader: r.RecordHeader,
			B1000:        r.B1000,
			B1001:        r.B1001,
			Data:         block,
		}

		offset := start.Add(time.Duration(count) * r.SamplePeriod())
		btime := NewBTime(offset)

		rec.RecordHeader.RecordStartTime = btime
		rec.RecordHeader.NumberOfSamples = uint16(len(b))
		rec.B1000.Encoding = uint8(EncodingIEEEDouble)
		rec.B1001.MicroSec = int8(offset.Sub(btime.Time()) / time.Microsecond)

		if err := fn(&rec); err != nil {
			return err
		}

		count += len(b)
	}

	return nil
}

// PackFloatSteim1 takes an int32 slice and packs it into miniseed Records, using Steim1 compression, which are passed to a callback function.
func (r *Record) PackSteim1(start time.Time, prev int32, raw []int32, fn RecordFunc) error {

	samples := append([]int32(nil), raw...)
	frames := (r.BlockSize() - int(r.BeginningOfData)) / 64

	var count int
	if err := packSteim(1, frames, prev, samples, func(buf []byte, index uint16, frames uint8) error {
		rec := Record{
			RecordHeader: r.RecordHeader,
			B1000:        r.B1000,
			B1001:        r.B1001,
			Data:         buf,
		}

		offset := start.Add(time.Duration(count) * r.SamplePeriod())
		btime := NewBTime(offset)

		rec.RecordHeader.RecordStartTime = btime
		rec.RecordHeader.NumberOfSamples = index
		rec.B1000.Encoding = uint8(EncodingSTEIM1)
		rec.B1000.WordOrder = uint8(BigEndian)
		rec.B1001.MicroSec = int8(offset.Sub(btime.Time()) / time.Microsecond)
		rec.B1001.FrameCount = frames

		if err := fn(&rec); err != nil {
			return err
		}

		count += int(index)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// PackFloatSteim2 takes an int32 slice and packs it into miniseed Records, using Steim2 compression, which are passed to a callback function.
func (r *Record) PackSteim2(start time.Time, prev int32, raw []int32, fn func(*Record) error) error {

	samples := append([]int32(nil), raw...)
	frames := (r.BlockSize() - int(r.BeginningOfData)) / 64

	var count int
	if err := packSteim(2, frames, prev, samples, func(buf []byte, index uint16, frames uint8) error {

		rec := Record{
			RecordHeader: r.RecordHeader,
			B1000:        r.B1000,
			B1001:        r.B1001,
			Data:         buf,
		}

		offset := start.Add(time.Duration(count) * r.SamplePeriod())
		btime := NewBTime(offset)

		rec.RecordHeader.RecordStartTime = btime
		rec.RecordHeader.NumberOfSamples = index
		rec.B1000.Encoding = uint8(EncodingSTEIM2)
		rec.B1000.WordOrder = uint8(BigEndian)
		rec.B1001.MicroSec = int8(offset.Sub(btime.Time()) / time.Microsecond)
		rec.B1001.FrameCount = frames

		if err := fn(&rec); err != nil {
			return err
		}

		count += int(index)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// Marshal converts a Record into a miniseed format byte slice.
func (r *Record) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	if err := r.Encode(&buf); err != nil {
		return nil, err
	}
	// build a byte array and copy running buffer
	blk := make([]byte, r.BlockSize())
	copy(blk, buf.Bytes())
	return blk, nil
}

// Encode writes a miniseed formatted byte slice into the given Writer.
func (r *Record) Encode(wr io.Writer) error {

	// encode the header into the buffer
	if err := r.RecordHeader.Encode(wr); err != nil {
		return err
	}

	// where the first blockette will be
	offset := int(r.FirstBlockette)

	// add any space between the header and the first blockette
	if n := offset - RecordHeaderSize; n > 0 {
		if _, err := wr.Write(make([]byte, n)); err != nil {
			return err
		}
	}

	// where the next blockette will be if present
	offset += BlocketteHeaderSize + Blockette1000Size

	b1000 := BlocketteHeader{
		BlocketteType: 1000,
		NextBlockette: func() uint16 {
			if r.RecordHeader.NumberOfBlockettesThatFollow > 1 {
				return uint16(offset)
			}
			return 0
		}(),
	}
	if err := b1000.Encode(wr); err != nil {
		return err
	}
	if err := r.B1000.Encode(wr); err != nil {
		return err
	}

	if r.RecordHeader.NumberOfBlockettesThatFollow > 1 {
		// where the next blockette will be if present
		offset += BlocketteHeaderSize + Blockette1001Size

		b1001 := BlocketteHeader{
			BlocketteType: 1001,
		}
		if err := b1001.Encode(wr); err != nil {
			return err
		}
		if err := r.B1001.Encode(wr); err != nil {
			return err
		}
	}

	// add any space between the blockettes and the data
	if n := int(r.BeginningOfData) - offset; n > 0 {
		if _, err := wr.Write(make([]byte, n)); err != nil {
			return err
		}
	}

	// write the actual encoded data
	if _, err := wr.Write(r.Data); err != nil {
		return err
	}

	return nil
}
