package render

import _ "embed"

// rawTestAudio is shared raw PCM test input (s16le, 2ch, 48kHz) used by the
// clip / flac / sox tests in this package.
//
//go:embed test_data/sweep_30_20000_s16le_2ch_48000k.raw
var rawTestAudio []byte
