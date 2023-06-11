package earss

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	BufferLength = 16384
	HeaderLength = 16
	MaxChannels  = 3
	DataLength   = 16368
	DataValues   = 8184
)

// GainSample accounts for the gain ranging
var GainSample = [8]int{128, 64, 32, 16, 8, 4, 2, 1}

// GainSystem accounts for the channel gain setting
var GainSystem = [8]int{1, 2, 4, 8, 16, 32, 64, 128}

func decodeSample(value int16) int {
	mag := int(value&4095) * GainSample[int((value>>12)&7)]
	if value < 0 {
		return -mag
	}
	return mag
}

type Record struct {
	StartTime        time.Time
	Instrument       int
	TapeNumber       int
	PreEventSeconds  int
	NumberOfChannels int
	BufferType       int
	BufferNumber     int
	LastTrigger      bool
	SampleRate       int
	TimeCorrection   int
	Gain             [MaxChannels]int
	Samples          [DataValues]int
}

func (r Record) String() string {
	var sb strings.Builder
	sb.WriteString(r.StartTime.Format(time.RFC3339Nano))
	sb.WriteString(fmt.Sprintf(" %d", r.Instrument))
	sb.WriteString(fmt.Sprintf(" %d", r.TapeNumber))
	sb.WriteString(fmt.Sprintf(" %d", r.PreEventSeconds))
	sb.WriteString(fmt.Sprintf(" %d", r.NumberOfChannels))
	sb.WriteString(fmt.Sprintf(" %d", r.BufferType))
	sb.WriteString(fmt.Sprintf(" %d", r.BufferNumber))
	sb.WriteString(fmt.Sprintf(" %v", r.LastTrigger))
	sb.WriteString(fmt.Sprintf(" %d", r.SampleRate))
	sb.WriteString(fmt.Sprintf(" %d", r.TimeCorrection))
	sb.WriteString(fmt.Sprintf(" %d %d %d", r.Gain[0], r.Gain[1], r.Gain[2]))
	sb.WriteString("\n")
	for i := 0; i < DataValues; i += r.NumberOfChannels {
		for j := 0; j < r.NumberOfChannels; j++ {
			sb.WriteString(fmt.Sprintf(" %v", r.Samples[i+j]))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (r *Record) Decode(data []byte) error {
	if len(data) != BufferLength {
		return fmt.Errorf("invalid length")
	}

	header := data[BufferLength-HeaderLength:]

	year := int(header[5])
	switch {
	case year < 50:
		year += 2000
	default:
		year += 1900
	}
	month := int(header[6])
	day := int(header[7])
	hour := int(header[10])
	minute := int(header[11])
	second := int(header[12])
	nano := 10000000 * int(header[13])

	r.StartTime = time.Date(year, time.Month(month), day, hour, minute, second, nano, time.UTC)
	r.SampleRate = int(math.Pow(2.0, float64((header[0]&48)/16)) * 25.0)
	r.LastTrigger = ((header[0] & 128) / 128) > 0

	r.BufferType = int(header[0] & 15)
	r.BufferNumber = int(header[1]) + 1
	r.NumberOfChannels = int(header[2]&3) + 1
	r.PreEventSeconds = int(header[4])
	r.Instrument = int(header[14])
	r.TapeNumber = int(header[15])

	r.Gain[0] = int((header[2] & 112) / 16)
	r.Gain[1] = int((header[2]&128)/128 + (header[3]&3)*2)
	r.Gain[2] = int((header[3] & 28) / 4)

	r.TimeCorrection = int(binary.LittleEndian.Uint16(header[8:10]))

	for i := 0; i < DataValues; i += r.NumberOfChannels {
		for j := 0; j < r.NumberOfChannels; j++ {
			value := int16(binary.LittleEndian.Uint16(data[(i+j)*2 : (i+j+1)*2]))

			r.Samples[i+j] = decodeSample(value)
		}
	}

	return nil
}

func Decode(data []byte) ([]Record, error) {
	var records []Record
	for i := 0; i < len(data)/BufferLength; i++ {
		var record Record
		if err := record.Decode(data[i*BufferLength : (i+1)*BufferLength]); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}
