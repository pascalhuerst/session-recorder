package main

import (
	"encoding/binary"
	"fmt"
	"io"

	"net/http"
	"os"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func main() {
	// Replace the URL with your actual presigned URL
	url := "http://127.0.0.1:9000/session-recorder/22a2a258-bc03-4836-a94a-d322d5e9aaaa/sessions/0f6e7eee-d4fe-4975-9641-29a33dd1a4d9/data.raw?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=rqn5q9WNvH9n5XOrVRrq%2F20240111%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20240111T232358Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=8609f5a4e2427dd7712ccff56588441e923009dbad8023f7f20bccfc9ae1a571"

	// Make an HTTP GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode != http.StatusOK {
		fmt.Println("Error: Unexpected status code", response.StatusCode)
		return
	}

	defer response.Body.Close()

	if err := wav2flac(response.Body); err != nil {
		log.Fatal().Err(err).Msg("Cannot convert wav to flac")
	}

}

func wav2flac(in io.ReadCloser) error {
	sampleRate := 48000
	nchannels := 2
	bps := 16

	// Create FLAC encoder.
	flacPath := "data.flac"
	w, err := os.Create(flacPath)
	if err != nil {
		return errors.WithStack(err)
	}

	info := &meta.StreamInfo{
		//		BlockSizeMin:  16,    // adjusted by encoder.
		//		BlockSizeMax:  65535, // adjusted by encoder.
		SampleRate:    uint32(sampleRate),
		NChannels:     uint8(nchannels),
		BitsPerSample: uint8(bps),
	}

	enc, err := flac.NewEncoder(w, info)
	if err != nil {
		return errors.WithStack(err)
	}
	defer enc.Close()

	// Number of samples per channel and block.
	const nsamplesPerChannel = 16
	nsamplesPerBlock := nchannels * nsamplesPerChannel

	subframes := make([]*frame.Subframe, nchannels)
	for i := range subframes {
		subframe := &frame.Subframe{
			Samples: make([]int32, nsamplesPerChannel),
		}

		subframes[i] = subframe
	}

	for frameNum := 0; ; frameNum++ {
		buf := make([]int16, nsamplesPerBlock)

		err := binary.Read(in, binary.LittleEndian, buf)
		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.WithStack(err)
		}

		for _, subframe := range subframes {
			subHdr := frame.SubHeader{
				Pred:   frame.PredVerbatim,
				Order:  0,
				Wasted: 0,
			}
			subframe.SubHeader = subHdr
			subframe.NSamples = len(buf) / nchannels
			subframe.Samples = subframe.Samples[:subframe.NSamples]
		}

		for i, sample := range buf {
			subframe := subframes[i%nchannels]
			subframe.Samples[i/nchannels] = int32(sample)
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
			return errors.WithStack(err)
		}
	}

	return nil
}
