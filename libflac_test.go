// Copyright 2015 Cocoon Alarm Ltd.
//
// See LICENSE file for terms and conditions.

package libflac

import (
	"io"
	"os"
	"testing"

	"github.com/jbert/testify/assert"
)

func TestDecode(t *testing.T) {
	a := assert.New(t)

	d, err := NewDecoder("data/nonexistent.flac")
	a.Equal(d, (*Decoder)(nil), "decoder is nil")
	a.NotNil(err, "err is not nil")

	d, err = NewDecoder("data/sine24-00.flac")

	a.Equal(err, nil, "err is nil")
	a.Equal(d.Channels, 1, "channels is 1")
	a.Equal(d.Depth, 24, "depth is 24")
	a.Equal(d.Rate, 48000, "depth is 48000")

	samples := 0

	f, err := d.ReadFrame()

	a.Equal(err, nil, "err is nil")
	a.Equal(f.Channels, 1, "channels is 1")
	a.Equal(f.Depth, 24, "depth is 24")
	a.Equal(f.Rate, 48000, "depth is 48000")

	samples = samples + len(f.Buffer)

	for {
		f, err := d.ReadFrame()

		if err == nil || err == io.EOF {
			if f != nil {
				samples = samples + len(f.Buffer)
			}
		} else {
			a.Equal(err, nil, "error reported")
			break
		}

		if err == io.EOF {
			break
		}
	}

	a.Equal(samples, 200000, "all samples read")
	d.Close()
}

func TestDecodeReader(t *testing.T) {
	a := assert.New(t)

	reader, _ := os.Open("data/nonexistent.flac")

	d, err := NewDecoderReader(reader)

	a.Equal(d, (*Decoder)(nil), "decoder is nil")
	a.Error(err)

	reader, _ = os.Open("data/sine24-00.flac")

	d, err = NewDecoderReader(reader)

	a.Equal(err, nil, "err is nil")
	a.Equal(d.Channels, 1, "channels is 1")
	a.Equal(d.Depth, 24, "depth is 24")
	a.Equal(d.Rate, 48000, "depth is 48000")

	samples := 0

	f, err := d.ReadFrame()

	a.Equal(err, nil, "err is nil")
	a.Equal(f.Channels, 1, "channels is 1")
	a.Equal(f.Depth, 24, "depth is 24")
	a.Equal(f.Rate, 48000, "depth is 48000")

	samples = samples + len(f.Buffer)

	for {
		f, err := d.ReadFrame()

		if err == nil || err == io.EOF {
			if f != nil {
				samples = samples + len(f.Buffer)
			}
		} else {
			a.Equal(err, nil, "error reported")
			break
		}

		if err == io.EOF {
			break
		}
	}

	a.Equal(samples, 200000, "all samples read")
	d.Close()
}

func TestEncode(t *testing.T) {
	a := assert.New(t)

	e, err := NewEncoder("not-existdir/foo.flac", 2, 24, 48000)
	a.Equal(e, (*Encoder)(nil), "encoder is nil")
	a.NotNil(err, "err is not nil")

	fileName := "data/test.flac"

	e, err = NewEncoder(fileName, 2, 24, 48000)

	a.Equal(err, nil, "err is nil")

	f := Frame{Channels: 1, Depth: 24, Rate: 48000}

	err = e.WriteFrame(f)

	a.Error(err, "channels mismatch")

	f.Channels = 2
	f.Buffer = make([]int32, 2*100)

	err = e.WriteFrame(f)

	a.Equal(err, nil, "frame encoded")

	e.Close()

	os.Remove(fileName)
}

func TestRoundTrip(t *testing.T) {
	a := assert.New(t)

	inputFile := "data/sine24-00.flac"
	outputFile := "data/test.flac"

	d, err := NewDecoder(inputFile)

	a.Equal(err, nil, "err is nil")

	e, err := NewEncoder(outputFile, d.Channels, d.Depth, d.Rate)

	samples := 0

	for {
		f, err := d.ReadFrame()
		if err == nil || err == io.EOF {
			if f != nil {
				_ = e.WriteFrame(*f)
				samples = samples + len(f.Buffer)
			}
		} else {
			a.Equal(err, nil, "error reported")
			break
		}

		if err == io.EOF {
			break
		}
	}

	a.Equal(samples, 200000, "all samples read")
	d.Close()
	e.Close()

	os.Remove(outputFile)
}

func TestRoundTripStereo(t *testing.T) {
	a := assert.New(t)

	inputFile := "data/sine16-12.flac"
	outputFile := "data/test.flac"

	d, err := NewDecoder(inputFile)

	a.Equal(err, nil, "err is nil")

	e, err := NewEncoder(outputFile, d.Channels, d.Depth, d.Rate)

	samples := 0

	for {
		f, err := d.ReadFrame()
		if err == nil || err == io.EOF {
			if f != nil {
				_ = e.WriteFrame(*f)
				samples = samples + len(f.Buffer)
			}
		} else {
			a.Equal(err, nil, "error reported")
			break
		}

		if err == io.EOF {
			break
		}
	}

	a.Equal(samples, 400000, "all samples read")
	d.Close()
	e.Close()

	os.Remove(outputFile)
}

func TestRoundTripReaderWriter(t *testing.T) {
	a := assert.New(t)

	inputFile := "data/sine24-00.flac"
	outputFile := "data/test.flac"

	reader, _ := os.Open(inputFile)

	d, err := NewDecoderReader(reader)

	a.Equal(err, nil, "err is nil")

	writer, _ := os.Create(outputFile)

	e, err := NewEncoderWriter(writer, d.Channels, d.Depth, d.Rate)

	samples := 0

	for {
		f, err := d.ReadFrame()
		if err == nil || err == io.EOF {
			if f != nil {
				_ = e.WriteFrame(*f)
				samples = samples + len(f.Buffer)
			}
		} else {
			a.Equal(err, nil, "error reported")
			break
		}

		if err == io.EOF {
			break
		}
	}

	a.Equal(samples, 200000, "all samples read")
	d.Close()
	e.Close()

	os.Remove(outputFile)
}
