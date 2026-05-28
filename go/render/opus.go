package render

import (
	"crypto/rand"
	"encoding/binary"
	"io"

	"github.com/kazzmir/opus-go/ogg"
	"github.com/kazzmir/opus-go/opus"
	"github.com/pkg/errors"
)

// Opus encodes raw PCM (s16le, 2ch, 48kHz) read from src into an Ogg/Opus
// stream written incrementally to dst. It replaces the previous sox-based OGG
// (Vorbis) output; the container extension and audio/ogg MIME type are
// unchanged, only the codec differs. Memory use is bounded (one Opus frame at a
// time) regardless of stream length, so it is safe for multi-hour sessions.
func Opus(dst io.Writer, src io.Reader) error {
	const (
		sr        = sampleRate  // 48000
		ch        = numChannels // 2
		frameSize = 960         // 20ms @ 48kHz
		bitrate   = 96000       // VBR target, ample for stereo playback previews
	)

	enc, err := opus.NewEncoder(sr, ch, opus.ApplicationAudio)
	if err != nil {
		return errors.WithStack(err)
	}
	defer enc.Close()

	if err := enc.SetBitrate(bitrate); err != nil {
		return errors.WithStack(err)
	}
	if err := enc.SetVBR(true); err != nil {
		return errors.WithStack(err)
	}
	if err := enc.SetComplexity(10); err != nil {
		return errors.WithStack(err)
	}

	lookahead, err := enc.Lookahead()
	if err != nil {
		return errors.WithStack(err)
	}

	pw := ogg.NewPacketWriter(dst, randomSerial())

	head := ogg.OpusHead{
		Version:              1,
		Channels:             ch,
		PreSkip:              uint16(lookahead),
		InputSampleRate:      sr,
		ChannelMappingFamily: 0,
	}
	headPkt, err := ogg.BuildOpusHeadPacket(head)
	if err != nil {
		return errors.WithStack(err)
	}
	tagsPkt, err := ogg.BuildOpusTagsPacket(ogg.OpusTags{Vendor: "session-recorder"})
	if err != nil {
		return errors.WithStack(err)
	}
	if err := pw.WritePacket(headPkt, 0, true, false); err != nil {
		return errors.WithStack(err)
	}
	if err := pw.WritePacket(tagsPkt, 0, false, false); err != nil {
		return errors.WithStack(err)
	}

	frameBytes := frameSize * ch * 2
	rbuf := make([]byte, frameBytes)
	pcm := make([]int16, frameSize*ch)
	packet := make([]byte, 4000)
	var totalPerCh uint64

	for {
		n, rerr := io.ReadFull(src, rbuf)
		if n == 0 {
			if rerr == io.EOF || rerr == io.ErrUnexpectedEOF {
				break
			}
			return errors.WithStack(rerr)
		}

		validSamples := n / 2
		for i := range pcm {
			if i < validSamples {
				pcm[i] = int16(binary.LittleEndian.Uint16(rbuf[i*2:]))
			} else {
				pcm[i] = 0 // zero-pad the final short frame
			}
		}

		isLast := rerr == io.EOF || rerr == io.ErrUnexpectedEOF

		nBytes, err := enc.Encode(pcm, frameSize, packet)
		if err != nil {
			return errors.WithStack(err)
		}

		totalPerCh += uint64(validSamples / ch)
		granule := uint64(head.PreSkip) + totalPerCh
		if err := pw.WritePacket(packet[:nBytes], granule, false, isLast); err != nil {
			return errors.WithStack(err)
		}

		if isLast {
			break
		}
	}

	if err := pw.Flush(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func randomSerial() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 1
	}
	return binary.LittleEndian.Uint32(b[:])
}
