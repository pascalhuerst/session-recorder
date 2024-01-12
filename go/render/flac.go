package render

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/pkg/errors"
)

func Flac(raw io.Reader) (*bytes.Buffer, error) {
	var (
		sampleRate uint32        = 48000
		nChannels  int           = 2
		bps        uint8         = 16
		ret        *bytes.Buffer = new(bytes.Buffer)
	)

	info := &meta.StreamInfo{
		//		BlockSizeMin:  16,    // adjusted by encoder.
		//		BlockSizeMax:  65535, // adjusted by encoder.
		SampleRate:    sampleRate,
		NChannels:     uint8(nChannels),
		BitsPerSample: bps,
	}

	enc, err := flac.NewEncoder(ret, info)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer enc.Close()

	// Number of samples per channel and block.
	const nsamplesPerChannel = 16
	nsamplesPerBlock := nChannels * nsamplesPerChannel

	subframes := make([]*frame.Subframe, nChannels)
	for i := range subframes {
		subframe := &frame.Subframe{
			Samples: make([]int32, nsamplesPerChannel),
		}

		subframes[i] = subframe
	}

	for frameNum := 0; ; frameNum++ {
		buf := make([]int16, nsamplesPerBlock)

		err := binary.Read(raw, binary.LittleEndian, buf)
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, errors.WithStack(err)
		}

		for _, subframe := range subframes {
			subHdr := frame.SubHeader{
				Pred:   frame.PredVerbatim,
				Order:  0,
				Wasted: 0,
			}
			subframe.SubHeader = subHdr
			subframe.NSamples = len(buf) / nChannels
			subframe.Samples = subframe.Samples[:subframe.NSamples]
		}

		for i, sample := range buf {
			subframe := subframes[i%nChannels]
			subframe.Samples[i/nChannels] = int32(sample)
		}

		for _, subframe := range subframes {
			sample := subframe.Samples[0]
			constant := true

			for _, s := range subframe.Samples[1:] {
				if sample != s {
					constant = false
				}
			}

			if constant {
				subframe.SubHeader.Pred = frame.PredConstant
			}
		}

		hdr := frame.Header{
			HasFixedBlockSize: false,
			BlockSize:         uint16(nsamplesPerChannel),
			SampleRate:        uint32(sampleRate),
			Channels:          frame.ChannelsLR,
			BitsPerSample:     uint8(bps),
		}

		f := &frame.Frame{
			Header:    hdr,
			Subframes: subframes,
		}

		if err := enc.WriteFrame(f); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return ret, nil
}
