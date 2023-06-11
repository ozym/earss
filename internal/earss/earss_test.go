package earss

import (
	"os"
	"testing"
)

func TestEarss_Reading(t *testing.T) {

	first := [3]int{-3, -2412, -10}
	last := [3]int{-2924, -8, -19}

	data, err := os.ReadFile("testdata/lylm0313.dat")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; (i < len(data)/BufferLength) && i < len(first) && i < len(last); i++ {
		var record Record
		if err := record.Decode(data[i*BufferLength : (i+1)*BufferLength]); err != nil {
			t.Fatal(err)
		}
		if s := record.Instrument; s != 106 {
			t.Errorf("invalid instrument id for record %d, expected %d but got %d", i+1, 106, s)
		}
		if s := record.SampleRate; s != 100 {
			t.Errorf("invalid sample rate for record %d, expected %d but got %d", i+1, 100, s)
		}
		if s := record.TimeCorrection; s != 54 {
			t.Errorf("invalid time correction for record %d, expected %d but got %d", i+1, 54, s)
		}
		if s := record.Samples[0]; s != first[i] {
			t.Errorf("invalid first sample for record %d, expected %d but got %d", i+1, first[i], s)
		}
		if s := record.Samples[len(record.Samples)-1]; s != last[i] {
			t.Errorf("invalid last sample for record %d, expected %d but got %d", i+1, last[i], s)
		}
	}
}
