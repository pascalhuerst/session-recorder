module github.com/pascalhuerst/session-recorder

go 1.23.0

toolchain go1.24.3

require (
	github.com/bogem/id3v2 v1.2.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/godbus/dbus/v5 v5.1.0
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/holoplot/go-avahi v1.0.1
	github.com/mattn/go-colorable v0.1.13
	github.com/mattn/go-isatty v0.0.20
	github.com/mewkiz/flac v1.0.10
	github.com/minio/minio-go/v7 v7.0.66
	github.com/pascalhuerst/session-recorder/protocols/go v0.0.0-20240420153411-21708cdb8a48
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.31.0
	golang.org/x/sync v0.12.0
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/icza/bitio v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.6 // indirect
	github.com/mewkiz/pkg v0.0.0-20230226050401-4010bf0fec14 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace github.com/pascalhuerst/session-recorder/protocols/go => ../protocols/go
