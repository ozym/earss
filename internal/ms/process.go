package ms

import (
	"bytes"
	"io"
	"os"
	"time"
)

// ProcessFn is a function to apply after each miniseed block has been decoded.
type ProcessFn func(src string, start time.Time, delta time.Duration, samples ...float64) error

// Process reads miniseed blocks of an expected blksize, these are decoded and passed
// to the given function. If the function returns an error it is immediately returned
// by the parent Process function.
func Process(rd io.Reader, blksize int, fn ProcessFn) error {

	buf := make([]byte, blksize)
	for {
		// read a full block, otherwise an error or the end of file.
		if _, err := io.ReadFull(rd, buf); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		msr, err := NewRecord(buf)
		if err != nil {
			return err
		}

		// decode the data samples into floats
		samples, err := msr.Float64s()
		if err != nil {
			return err
		}

		// pass the results to the process function
		if err := fn(msr.SrcName(false), msr.StartTime(), msr.SamplePeriod(), samples...); err != nil {
			return err
		}
	}
}

// ProcessFile reads a file for miniseed blocks of an expected blksize,
// these are decoded and passed to the given function. If the function returns
// an error it is immediately returned by the parent Process function.
func ProcessFile(path string, blksize int, fn ProcessFn) error {

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := Process(file, blksize, fn); err != nil {
		return err
	}

	return nil
}

// ProcessBytes reads a byte slice for miniseed blocks of an expected blksize,
// these are decoded and passed to the given function. If the function returns
// an error it is immediately returned by the parent Process function.
func ProcessBytes(data []byte, blksize int, fn ProcessFn) error {
	return Process(bytes.NewBuffer(data), blksize, fn)
}
