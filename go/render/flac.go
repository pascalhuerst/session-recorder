package render

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/pkg/errors"
)

// Flac encodes raw PCM (s16le, 2ch, 48kHz) read from src into a FLAC stream
// written incrementally to dst. Frames are emitted one block at a time, so
// memory use is bounded regardless of stream length (safe for multi-hour
// sessions).
func Flac(dst io.Writer, src io.Reader) error {
	const (
		sr  uint32 = 48000
		bps uint8  = 16
		// Number of inter-channel samples per block.
		nsamplesPerChannel = 16
	)

	// The encoder only back-patches BlockSizeMin/Max in Close() when the writer
	// is an io.WriteSeeker; we stream to a pipe/file, so they must be set up
	// front or the header carries an invalid block size of 0. Frames use a fixed
	// block size of nsamplesPerChannel (a short final block may be smaller).
	info := &meta.StreamInfo{
		BlockSizeMin:  nsamplesPerChannel,
		BlockSizeMax:  nsamplesPerChannel,
		SampleRate:    sr,
		NChannels:     numChannels,
		BitsPerSample: bps,
	}

	enc, err := flac.NewEncoder(dst, info)
	if err != nil {
		return errors.WithStack(err)
	}
	defer enc.Close()

	subframes := make([]*frame.Subframe, numChannels)
	for i := range subframes {
		subframes[i] = &frame.Subframe{Samples: make([]int32, nsamplesPerChannel)}
	}

	br := bufio.NewReaderSize(src, 64*1024)
	rbuf := make([]byte, nsamplesPerChannel*numChannels*2)

	for {
		n, rerr := io.ReadFull(br, rbuf)
		if n == 0 {
			if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
				break
			}
			return errors.WithStack(rerr)
		}

		// Whole frames only (the raw stream is frame-aligned: n%bytesPerFrame==0).
		nFrames := n / bytesPerFrame

		for ch, subframe := range subframes {
			subframe.SubHeader = frame.SubHeader{Pred: frame.PredVerbatim}
			subframe.NSamples = nFrames
			subframe.Samples = subframe.Samples[:nFrames]
			for i := range nFrames {
				s := int16(binary.LittleEndian.Uint16(rbuf[(i*numChannels+ch)*2:]))
				subframe.Samples[i] = int32(s)
			}

			constant := true
			for _, s := range subframe.Samples[1:] {
				if s != subframe.Samples[0] {
					constant = false
					break
				}
			}
			if constant {
				subframe.SubHeader.Pred = frame.PredConstant
			}
		}

		f := &frame.Frame{
			Header: frame.Header{
				HasFixedBlockSize: false,
				BlockSize:         uint16(nFrames),
				SampleRate:        sr,
				Channels:          frame.ChannelsLR,
				BitsPerSample:     bps,
			},
			Subframes: subframes,
		}
		if err := enc.WriteFrame(f); err != nil {
			return errors.WithStack(err)
		}

		if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
			break
		}
	}

	return nil
}
