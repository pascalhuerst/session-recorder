package render

import (
	"bytes"
	"context"
	"crypto/md5"
	_ "embed"
	"io"
	"testing"
)

func mkHash(r io.Reader) ([]byte, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, r); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

//go:embed test_data/sweep_30_20000_s16le_2ch_48000k.raw
var rawTestAudio []byte

//go:embed test_data/sweep_30_20000_s16le_2ch_48000k.png
var resultWaveform []byte

func TestCreateOverview(t *testing.T) {
	type args struct {
		raw    io.Reader
		zoom   int
		width  int
		height int
	}
	tests := []struct {
		name    string
		args    args
		want    *bytes.Buffer
		wantErr bool
	}{
		{
			name: "sweep_30_20000_s16le_2ch_48000k",
			args: args{
				raw:    bytes.NewReader(rawTestAudio),
				zoom:   256,
				width:  1024,
				height: 256,
			},
			want: bytes.NewBuffer(resultWaveform),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateOverview(context.Background(), tt.args.raw, tt.args.zoom, tt.args.width, tt.args.height)
			if err != nil {
				t.Errorf("CreateOverview() error = %v, wantErr %v", err, tt.wantErr)
			}

			if got.Len() != tt.want.Len() {
				t.Errorf("CreateOverview() length mismatch: got = %v bytes, want %v bytes", got.Len(), tt.want.Len())
			}

			gotHash, err := mkHash(got)
			if err != nil {
				t.Errorf("CreateOverview() error = %v, wantErr %v", err, tt.wantErr)
			}

			wantHash, err := mkHash(tt.want)
			if err != nil {
				t.Errorf("CreateOverview() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !bytes.Equal(gotHash, wantHash) {
				t.Errorf("CreateOverview() got = %v, want %v", gotHash, wantHash)
			}
		})
	}
}
