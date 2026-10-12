package render

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"io"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/pkg/errors"
)

// PCMSummary is what a FLAC header states about the audio as a whole: how many
// inter-channel samples it holds, and the MD5 of the unencoded samples. For
// s16le input that MD5 is simply the hash of the raw bytes.
//
// Both go into the STREAMINFO block at the very start of the stream, so they
// have to be known before the first frame is written whenever the destination
// cannot be seeked back into - which is how every FLAC here is written (piped
// into an upload, or through a buffered file writer). Without them a player
// cannot show a duration or seek properly, and nothing can verify the audio.
type PCMSummary struct {
	Frames uint64
	MD5    [md5.Size]byte
}

// SummarizePCM reads raw PCM (s16le, 2ch) from src to its end and returns its
// summary. A trailing partial frame is ignored, exactly as Flac ignores it.
func SummarizePCM(src io.Reader) (PCMSummary, error) {
	hash := md5.New()
	buf := make([]byte, 64*1024) // a multiple of bytesPerFrame
	var summary PCMSummary
	for {
		n, err := io.ReadFull(src, buf)
		whole := n - n%bytesPerFrame
		hash.Write(buf[:whole])
		summary.Frames += uint64(whole / bytesPerFrame)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return PCMSummary{}, errors.WithStack(err)
		}
	}
	hash.Sum(summary.MD5[:0])
	return summary, nil
}

// Flac encodes src like the package-level Flac, and writes this summary into
// the stream header. It fails if src turns out not to be the audio the
// summary was taken from - a header that contradicts its own stream would be
// worse than one that says nothing.
func (s PCMSummary) Flac(dst io.Writer, src io.Reader) error {
	return encodeFlac(dst, src, &s)
}

// Flac encodes raw PCM (s16le, 2ch, 48kHz) read from src into a FLAC stream
// written incrementally to dst. Frames are emitted one block at a time, so
// memory use is bounded regardless of stream length (safe for multi-hour
// sessions).
//
// Each block is compressed: the stereo representation, predictor and Rice
// parameters are chosen per block (see flac_predict.go), falling back to
// storing the samples verbatim where prediction does not pay off.
//
// The header carries neither a total length nor an MD5 this way; use
// PCMSummary.Flac where the input can be read twice.
func Flac(dst io.Writer, src io.Reader) error {
	return encodeFlac(dst, src, nil)
}

func encodeFlac(dst io.Writer, src io.Reader, summary *PCMSummary) error {
	const (
		sr  uint32 = 48000
		bps uint8  = 16
		// Number of inter-channel samples per block. 4096 is the usual FLAC
		// block size; every block carries its own frame header and CRCs, so
		// very small blocks inflate the file and make it slow to decode.
		nsamplesPerChannel = 4096
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
	if summary != nil {
		info.NSamples = summary.Frames
		info.MD5sum = summary.MD5
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
	predict := newPredictor(nsamplesPerChannel)
	mid := make([]int32, nsamplesPerChannel)
	side := make([]int32, nsamplesPerChannel)

	br := bufio.NewReaderSize(src, 64*1024)
	rbuf := make([]byte, nsamplesPerChannel*numChannels*2)
	encoded := PCMSummary{}
	encodedHash := md5.New()

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
		encoded.Frames += uint64(nFrames)
		encodedHash.Write(rbuf[:nFrames*bytesPerFrame])

		for ch, subframe := range subframes {
			subframe.NSamples = nFrames
			subframe.Samples = subframe.Samples[:nFrames]
			for i := range nFrames {
				s := int16(binary.LittleEndian.Uint16(rbuf[(i*numChannels+ch)*2:]))
				subframe.Samples[i] = int32(s)
			}
		}

		// The subframes keep the plain left/right samples: the encoder applies
		// the chosen stereo representation itself while writing the frame.
		plan := predict.planStereo(subframes[0].Samples, subframes[1].Samples, mid, side, uint(bps))
		for ch, subframe := range subframes {
			plan.subframes[ch].apply(subframe)
		}

		f := &frame.Frame{
			Header: frame.Header{
				HasFixedBlockSize: false,
				BlockSize:         uint16(nFrames),
				SampleRate:        sr,
				Channels:          plan.channels,
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

	if summary != nil {
		encodedHash.Sum(encoded.MD5[:0])
		if encoded != *summary {
			return errors.Errorf("flac: encoded %d frames that do not match the %d-frame summary in the header", encoded.Frames, summary.Frames)
		}
	}

	return nil
}
