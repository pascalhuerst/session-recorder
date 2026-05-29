// Command waveform_png renders an audiowaveform ".dat" file to a PNG, for
// eyeballing the preview rendering used by render.WaveformPNG.
//
//	go run ./cmd/waveform_png -in /tmp/waveform.dat -out /tmp/waveform.png
//	go run ./cmd/waveform_png -width 800 -height 80 /tmp/waveform.dat /tmp/out.png
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pascalhuerst/session-recorder/render"
)

func main() {
	in := flag.String("in", "", "input audiowaveform .dat file")
	out := flag.String("out", "", "output .png file")
	width := flag.Int("width", 600, "PNG width in pixels")
	height := flag.Int("height", 80, "PNG height in pixels")
	flag.Parse()

	// Allow positional args too: waveform_png <in.dat> <out.png>
	if args := flag.Args(); len(args) >= 1 && *in == "" {
		*in = args[0]
	}
	if args := flag.Args(); len(args) >= 2 && *out == "" {
		*out = args[1]
	}

	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: waveform_png [-width N] [-height N] -in <input.dat> -out <output.png>")
		fmt.Fprintln(os.Stderr, "   or: waveform_png [-width N] [-height N] <input.dat> <output.png>")
		os.Exit(2)
	}

	datFile, err := os.Open(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot open %s: %v\n", *in, err)
		os.Exit(1)
	}
	defer datFile.Close()

	png, err := render.WaveformPNG(datFile, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot render waveform: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, png.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "cannot write %s: %v\n", *out, err)
		os.Exit(1)
	}

	fmt.Printf("wrote %dx%d PNG (%d bytes) to %s\n", *width, *height, png.Len(), *out)
}
