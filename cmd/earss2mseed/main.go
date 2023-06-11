package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/ozym/earss/internal/earss"
	"github.com/ozym/earss/internal/ms"
)

type Settings struct {
	verbose bool
	blksize int

	station    string
	network    string
	location   string
	channel    string
	components string
}

func (s Settings) Blksize() int {
	return int(math.Log(float64(s.blksize)) / math.Log(2))
}

func (s Settings) Channel(offset int) string {
	if offset < len(s.components) {
		return s.channel + string(s.components[offset])
	}
	return s.channel
}

func main() {

	var settings Settings

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Convert EARSS formatted data into miniSeed\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  %s [options] <files...>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "General Options:\n")
		fmt.Fprintf(os.Stderr, "\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	flag.BoolVar(&settings.verbose, "verbose", false, "make noise.")
	flag.StringVar(&settings.network, "network", "XX", "miniseed network code")
	flag.StringVar(&settings.station, "station", "XXXX", "miniseed station code")
	flag.StringVar(&settings.location, "location", "XX", "miniseed location code prefix")
	flag.StringVar(&settings.channel, "channel", "EH", "miniseed channel code prefix")
	flag.StringVar(&settings.components, "components", "ZNE", "miniseed channel code suffix")
	flag.IntVar(&settings.blksize, "blksize", 512, "miniseed block size in bytes")

	flag.Parse()

	for _, f := range flag.Args() {
		if settings.verbose {
			log.Printf("converting file %s", f)
		}
		data, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		records, err := earss.Decode(data)
		if err != nil {
			log.Fatal(err)
		}

		if settings.verbose {
			log.Printf("read %d records from %s", len(records), f)
		}

		var counter int
		for _, record := range records {
			for channel := 0; channel < record.NumberOfChannels; channel++ {
				var samples []int32
				for i := 0; i < earss.DataValues; i += record.NumberOfChannels {
					samples = append(samples, int32(record.Samples[i+channel]))
				}
				rec := ms.NewEmptyRecord(settings.Blksize(), record.SampleRate, 1)
				start := record.StartTime.Add(-time.Second * time.Duration(record.PreEventSeconds))
				copy(rec.NetworkIdentifier[:], []byte(settings.network))
				copy(rec.StationIdentifier[:], []byte(settings.station))
				copy(rec.LocationIdentifier[:], []byte(settings.location))
				copy(rec.ChannelIdentifier[:], []byte(settings.Channel(channel)))
				rec.TimeCorrection = int32(100 * record.TimeCorrection)
				if err := rec.PackSteim2(start, 0, samples, func(msr *ms.Record) error {
					copy(msr.SequenceNumber[:], []byte(fmt.Sprintf("%06d", counter+1)))
					counter++
					return msr.Encode(os.Stdout)
				}); err != nil {
					log.Fatal(err)
				}
			}
		}
		if settings.verbose {
			log.Printf("packed %d blocks from %s", counter, f)
		}
	}

	if settings.verbose {
		log.Println("conversion complete.")
	}
}
