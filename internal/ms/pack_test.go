package ms

import (
	"os"
	"testing"
	"time"
)

func TestRecord_Repack(t *testing.T) {

	files := []string{
		"NZ.AUCT.40.BTT.mseed",
		"NZ.CHIT.40.BTT.mseed",
	}

	for _, k := range files {
		t.Run("repack: "+k, func(t *testing.T) {
			raw, err := os.ReadFile("testdata/" + k)
			if err != nil {
				t.Fatal(err)
			}
			var rec Record
			if err := rec.Unpack(raw); err != nil {
				t.Fatal(err)
			}

			data, err := rec.Int32s()
			if err != nil {
				t.Fatal(err)
			}

			var count int
			var packed []int32
			var times []time.Time

			if err := rec.PackInt32(rec.StartTime(), data, func(msr *Record) error {
				values, err := msr.Int32s()
				if err != nil {
					t.Fatal(err)
				}
				for i := range values {
					times = append(times, msr.StartTime().Add(time.Duration(i)*msr.SamplePeriod()))
				}
				packed = append(packed, values...)
				count += msr.SampleCount()
				return nil
			}); err != nil {
				t.Fatal(err)
			}

			if len(packed) != len(data) {
				t.Fatalf("invalid unpacked sample count, expected %d got %d", len(data), len(packed))
			}

			for i := 0; i < len(data); i++ {
				if packed[i] == data[i] {
					continue
				}
				t.Fatalf("invalid unpacked sample %d, expected %d got %d", i, data[i], packed[i])
			}
			for i := 0; i < len(data); i++ {
				if at := rec.StartTime().Add(time.Duration(i) * rec.SamplePeriod()); !at.Equal(times[i]) {
					t.Errorf("invalid sample time %d, expected %s got %s", i, at.String(), times[i].String())
				}
			}

			if count != rec.SampleCount() {
				t.Errorf("invalid sample count, expected %d got %d", rec.SampleCount(), count)
			}
		})
	}
}
