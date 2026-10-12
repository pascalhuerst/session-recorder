package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pascalhuerst/session-recorder/broadcast"
	"github.com/pascalhuerst/session-recorder/grpc"
	cspb "github.com/pascalhuerst/session-recorder/protocols/go/chunksink"
	cmpb "github.com/pascalhuerst/session-recorder/protocols/go/common"
	sspb "github.com/pascalhuerst/session-recorder/protocols/go/sessionsource"
	"github.com/pascalhuerst/session-recorder/storage"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

/**
 * Test Plan: a recorder's status reaches StreamRecorders subscribers unchanged
 *
 * The status carries audio fingerprints the backend does not read itself
 * (chromaprintFingerprint, chromaprintSpeedFingerprints, chromaprintSpeedCount,
 * peakConstellation - fields 6 to 9). These tests run the real servers and
 * handlers on loopback: a ChunkSink client plays the recorder, SessionSource
 * clients play the subscribers.
 *
 * Scenario: Live relay over gRPC
 *   Given two subscribers on StreamRecorders
 *   When the recorder sets a status with all four fingerprint fields
 *   Then both subscribers receive exactly that status
 *   And the same holds for the status that follows it
 *
 * Scenario: A subscriber that connects later
 *   Given the recorder has set its status, and further RPCs have been served
 *   When a subscriber connects
 *   Then the first thing it receives for that recorder is that status, intact
 *
 * Scenario: gRPC-Web (the web UI's transport), binary and text framing
 *   Given the recorder has set its status
 *   When a gRPC-Web client opens StreamRecorders
 *   Then it receives the cached status, and the next live one, intact
 *
 * Scenario: Fields newer than the backend's generated code
 *   When the recorder sets a status with a field the backend does not know
 *   Then subscribers still receive it
 *
 * The statuses used are as large as the recorder can make them (see
 * largestStatus), so the scenarios also cover the message size.
 */

const relayTimeout = 5 * time.Second

// fakeRecorderStorage is the part of the storage the status path touches:
// the list of known recorders.
type fakeRecorderStorage struct {
	storage.Storage
	mu        sync.Mutex
	recorders map[uuid.UUID]storage.Recorder
}

func (f *fakeRecorderStorage) EnsureRecorderExists(_ context.Context, recorderID uuid.UUID, recorderName string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.recorders == nil {
		f.recorders = make(map[uuid.UUID]storage.Recorder)
	}
	f.recorders[recorderID] = storage.Recorder{ID: recorderID, Name: recorderName}
}

func (f *fakeRecorderStorage) GetRecorders() map[uuid.UUID]storage.Recorder {
	f.mu.Lock()
	defer f.mu.Unlock()
	recorders := make(map[uuid.UUID]storage.Recorder, len(f.recorders))
	for id, recorder := range f.recorders {
		recorders[id] = recorder
	}
	return recorders
}

func (f *fakeRecorderStorage) RegisterOnSessionStateChangedCallback(storage.OnSessionStateChangedCb) error {
	return nil
}

func (f *fakeRecorderStorage) RegisterOnAudioChunkCallback(storage.OnAudioChunkCb) error {
	return nil
}

// relayFixture is the backend's status path wired up the way main() does it.
type relayFixture struct {
	t           *testing.T
	broadcaster *broadcast.RecorderBroadcaster
	recorder    cspb.ChunkSinkClient
	source      sspb.SessionSourceClient
	grpcWebURL  string
}

func newRelayFixture(t *testing.T) *relayFixture {
	t.Helper()

	fake := &fakeRecorderStorage{}
	// Same per-subscriber buffer as main().
	broadcaster := broadcast.NewRecorderBroadcaster(10)

	chunkSinkHandler := NewChunkSinkHandler(fake, broadcaster)
	chunkSinkServer := grpc.NewChunkSinkServer(&grpc.ChunkSinkServerConfig{
		Name:               "test",
		OnRecorderStatusCB: chunkSinkHandler.setRecorderStatus,
	})
	sessionSourceHandler := NewSessionSourceHandler(fake, chunkSinkServer, broadcaster,
		broadcast.NewSessionBroadcaster(10), broadcast.NewAudioBroadcaster(10), nil, nil)
	sessionSourceServer := grpc.NewSessionSourceServer(&grpc.SessionSourceServerConfig{
		Name:              "test",
		StreamRecordersCB: sessionSourceHandler.streamRecorders,
	})

	chunkSinkGrpc, _, chunkSinkPort, err := grpc.StartProtocolServer(chunkSinkServer, nil, "chunksink-test", 0)
	if err != nil {
		t.Fatalf("start chunk sink server: %v", err)
	}
	t.Cleanup(chunkSinkGrpc.Stop)

	sessionSourceGrpc, _, sessionSourcePort, err := grpc.StartProtocolServer(sessionSourceServer, nil, "sessionsource-test", 0)
	if err != nil {
		t.Fatalf("start session source server: %v", err)
	}
	t.Cleanup(sessionSourceGrpc.Stop)

	grpcWeb := httptest.NewServer(grpc.NewGrpcWebHandler(sessionSourceGrpc))
	t.Cleanup(func() {
		grpcWeb.CloseClientConnections()
		grpcWeb.Close()
	})

	return &relayFixture{
		t:           t,
		broadcaster: broadcaster,
		recorder:    cspb.NewChunkSinkClient(dialLoopback(t, chunkSinkPort)),
		source:      sspb.NewSessionSourceClient(dialLoopback(t, sessionSourcePort)),
		grpcWebURL:  grpcWeb.URL,
	}
}

func dialLoopback(t *testing.T, port uint16) *ggrpc.ClientConn {
	t.Helper()
	conn, err := ggrpc.NewClient(fmt.Sprintf("127.0.0.1:%d", port), ggrpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial port %d: %v", port, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// setStatus plays the recorder. It sends a copy, so the caller's status stays
// what the test compares against.
func (x *relayFixture) setStatus(status *cmpb.RecorderStatus) {
	x.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), relayTimeout)
	defer cancel()
	response, err := x.recorder.SetRecorderStatus(ctx, proto.Clone(status).(*cmpb.RecorderStatus))
	if err != nil {
		x.t.Fatalf("SetRecorderStatus: %v", err)
	}
	if !response.GetSuccess() {
		x.t.Fatalf("SetRecorderStatus: not successful: %s", response.GetErrorMessage())
	}
}

// subscribe opens StreamRecorders over gRPC and returns once the backend has
// registered the subscription, so a status set afterwards is a live one.
func (x *relayFixture) subscribe() sspb.SessionSource_StreamRecordersClient {
	x.t.Helper()
	before := x.broadcaster.SubscriberCount()
	ctx, cancel := context.WithTimeout(context.Background(), relayTimeout)
	x.t.Cleanup(cancel)
	stream, err := x.source.StreamRecorders(ctx, &sspb.StreamRecordersRequest{})
	if err != nil {
		x.t.Fatalf("StreamRecorders: %v", err)
	}
	x.waitForSubscribers(before + 1)
	return stream
}

func (x *relayFixture) waitForSubscribers(n int) {
	x.t.Helper()
	deadline := time.Now().Add(relayTimeout)
	for x.broadcaster.SubscriberCount() < n {
		if time.Now().After(deadline) {
			x.t.Fatalf("backend has %d subscribers, want %d", x.broadcaster.SubscriberCount(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

// expectStatus reads from next until the status of want's recorder comes by
// and fails unless it is exactly want. Other recorders' updates are skipped.
func expectStatus(t *testing.T, next func() (*sspb.Recorder, error), want *cmpb.RecorderStatus) {
	t.Helper()
	for {
		recorder, err := next()
		if err != nil {
			t.Fatalf("waiting for status of recorder %s: %v", want.GetRecorderID(), err)
		}
		if recorder.GetRecorderID() != want.GetRecorderID() {
			continue
		}
		assertSameStatus(t, recorder.GetStatus(), want)
		return
	}
}

func assertSameStatus(t *testing.T, got, want *cmpb.RecorderStatus) {
	t.Helper()
	if got == nil {
		t.Fatalf("recorder update carries no status")
	}
	if got.GetChromaprintFingerprint() != want.GetChromaprintFingerprint() {
		t.Errorf("chromaprintFingerprint (6): got %d chars, want the %d sent",
			len(got.GetChromaprintFingerprint()), len(want.GetChromaprintFingerprint()))
	}
	gotSpeeds, wantSpeeds := got.GetChromaprintSpeedFingerprints(), want.GetChromaprintSpeedFingerprints()
	if len(gotSpeeds) != len(wantSpeeds) {
		t.Errorf("chromaprintSpeedFingerprints (7): got %d entries, want %d", len(gotSpeeds), len(wantSpeeds))
	} else {
		for i := range wantSpeeds {
			if !proto.Equal(gotSpeeds[i], wantSpeeds[i]) {
				t.Errorf("chromaprintSpeedFingerprints (7): entry %d (speed %v) differs from the one sent (speed %v)",
					i, gotSpeeds[i].GetSpeedPercent(), wantSpeeds[i].GetSpeedPercent())
			}
		}
	}
	if got.GetChromaprintSpeedCount() != want.GetChromaprintSpeedCount() {
		t.Errorf("chromaprintSpeedCount (8): got %d, want %d", got.GetChromaprintSpeedCount(), want.GetChromaprintSpeedCount())
	}
	if !bytes.Equal(got.GetPeakConstellation(), want.GetPeakConstellation()) {
		t.Errorf("peakConstellation (9): got %d bytes, want the %d sent",
			len(got.GetPeakConstellation()), len(want.GetPeakConstellation()))
	}
	// Everything else, including fields unknown to the generated code.
	if !proto.Equal(got, want) {
		t.Errorf("status differs from the one sent:\n got  %d bytes\n want %d bytes", proto.Size(got), proto.Size(want))
	}
}

// Sizes the recorder can reach: rust/recorder allows at most 64 speed
// variants (fingerprint.rs MAX_SPEED_VARIANTS); a Chromaprint of its 25 s
// window is about 200 values, under 1100 base64 characters; a constellation
// of 25 s holds at most a few hundred peaks, 5 bytes each - 1000 here.
const (
	largestSpeedVariants      = 64
	largestFingerprintChars   = 1100
	largestConstellationPeaks = 1000
)

// largestStatus is a status with all four fingerprint fields at the sizes
// above. Different seeds give different content of the same size.
func largestStatus(recorderID uuid.UUID, seed int) *cmpb.RecorderStatus {
	fingerprint := func(n int) string {
		const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
		var b strings.Builder
		for i := 0; i < largestFingerprintChars; i++ {
			b.WriteByte(alphabet[(i*7+n*13+seed*31)%len(alphabet)])
		}
		return b.String()
	}

	speeds := make([]*cmpb.SpeedFingerprint, 0, largestSpeedVariants)
	for i := 1; i <= largestSpeedVariants/2; i++ {
		speeds = append(speeds,
			&cmpb.SpeedFingerprint{SpeedPercent: float64(i) * 0.25, Fingerprint: fingerprint(2 * i)},
			&cmpb.SpeedFingerprint{SpeedPercent: float64(-i) * 0.25, Fingerprint: fingerprint(2*i + 1)},
		)
	}

	// "SIC1", count, frame numbers (uint32 LE), bins (uint8). Bins run over
	// every byte value, so a transport that is not 8-bit clean would show.
	constellation := []byte("SIC1")
	constellation = binary.LittleEndian.AppendUint32(constellation, largestConstellationPeaks)
	for i := 0; i < largestConstellationPeaks; i++ {
		constellation = binary.LittleEndian.AppendUint32(constellation, uint32(i+seed))
	}
	for i := 0; i < largestConstellationPeaks; i++ {
		constellation = append(constellation, byte(i+seed))
	}

	return &cmpb.RecorderStatus{
		RecorderID:                   recorderID.String(),
		RecorderName:                 "relay test",
		SignalStatus:                 cmpb.SignalStatus_SIGNAL,
		PeakPercent:                  42.5,
		Clipping:                     true,
		ChromaprintFingerprint:       fingerprint(0),
		ChromaprintSpeedFingerprints: speeds,
		ChromaprintSpeedCount:        largestSpeedVariants,
		PeakConstellation:            constellation,
	}
}

func TestRecorderStatusFingerprintsReachLiveSubscribers(t *testing.T) {
	x := newRelayFixture(t)
	recorderID := uuid.New()

	first, second := x.subscribe(), x.subscribe()

	status := largestStatus(recorderID, 1)
	t.Logf("status size on the wire: %d bytes", proto.Size(status))
	x.setStatus(status)
	expectStatus(t, first.Recv, status)
	expectStatus(t, second.Recv, status)

	// The next window: a new fingerprint, the speed list starting over.
	next := largestStatus(recorderID, 2)
	next.ChromaprintSpeedFingerprints = next.ChromaprintSpeedFingerprints[:3]
	x.setStatus(next)
	expectStatus(t, first.Recv, next)
	expectStatus(t, second.Recv, next)
}

func TestRecorderStatusFingerprintsReachLateSubscriberFromCache(t *testing.T) {
	x := newRelayFixture(t)
	recorderID := uuid.New()

	status := largestStatus(recorderID, 1)
	x.setStatus(status)
	// More requests of the same size on the same connection: the cached
	// status must not share memory with a request buffer that is reused.
	for seed := 2; seed < 6; seed++ {
		x.setStatus(largestStatus(uuid.New(), seed))
	}

	late := x.subscribe()
	expectStatus(t, late.Recv, status)
}

func TestRecorderStatusFingerprintsOverGrpcWeb(t *testing.T) {
	for _, contentType := range []string{"application/grpc-web+proto", "application/grpc-web-text"} {
		t.Run(contentType, func(t *testing.T) {
			x := newRelayFixture(t)
			recorderID := uuid.New()

			cached := largestStatus(recorderID, 1)
			x.setStatus(cached)

			next := x.subscribeGrpcWeb(contentType)
			expectStatus(t, next, cached)

			live := largestStatus(recorderID, 2)
			x.setStatus(live)
			expectStatus(t, next, live)
		})
	}
}

func TestRecorderStatusFieldsUnknownToTheBackendAreRelayed(t *testing.T) {
	x := newRelayFixture(t)
	recorderID := uuid.New()

	// A field number RecorderStatus does not have (yet).
	const futureField = 1000
	var unknown []byte
	unknown = protowire.AppendTag(unknown, futureField, protowire.BytesType)
	unknown = protowire.AppendBytes(unknown, []byte("from a newer recorder"))

	status := largestStatus(recorderID, 1)
	status.ProtoReflect().SetUnknown(unknown)

	live := x.subscribe()
	x.setStatus(status)
	expectStatus(t, live.Recv, status)

	late := x.subscribe()
	expectStatus(t, late.Recv, status)
}

// subscribeGrpcWeb opens StreamRecorders the way a browser does - a plain
// HTTP/1.1 POST to the gRPC-Web handler - and returns a function that reads
// the next message from the response. "application/grpc-web-text" is the
// base64 framing the web UI's GrpcWebFetchTransport uses by default.
// A recorder with all live analysis switched off (or one from before it
// existed) sends none of the fingerprint fields. That is the plain case, not a
// degraded one: its status is relayed as it is, live and from the cache, and
// nothing is filled in for it.
func TestRecorderWithoutLiveAnalysisIsRelayedAsIs(t *testing.T) {
	x := newRelayFixture(t)
	recorderID := uuid.New()
	live := x.subscribe()

	plain := &cmpb.RecorderStatus{
		RecorderID:   recorderID.String(),
		RecorderName: "No analysis",
		SignalStatus: cmpb.SignalStatus_SIGNAL,
		PeakPercent:  42.5,
	}
	x.setStatus(plain)
	expectStatus(t, live.Recv, plain)

	late := x.subscribe()
	got, err := late.Recv()
	if err != nil {
		t.Fatalf("late subscriber: %v", err)
	}
	status := got.GetStatus()
	assertSameStatus(t, status, plain)
	if status.GetChromaprintFingerprint() != "" || len(status.GetChromaprintSpeedFingerprints()) != 0 ||
		len(status.GetPeakConstellation()) != 0 || status.GetTempoBpm() != 0 ||
		status.GetChromaprintEnabled() || status.GetPeakConstellationEnabled() || status.GetTempoEnabled() {
		t.Errorf("the backend invented analysis data for a recorder that sends none: %v", status)
	}
}

// What the recorder says it is set up for, the window number and the tempo
// travel like the fingerprints: unchanged.
func TestRecorderStatusAnalysisFlagsWindowAndTempoAreRelayed(t *testing.T) {
	x := newRelayFixture(t)
	recorderID := uuid.New()
	live := x.subscribe()

	// Only the constellation and the tempo switched on; the window's first status.
	status := &cmpb.RecorderStatus{
		RecorderID:               recorderID.String(),
		RecorderName:             "Constellation and tempo",
		SignalStatus:             cmpb.SignalStatus_SIGNAL,
		PeakConstellationEnabled: true,
		TempoEnabled:             true,
		FingerprintWindow:        17,
		TempoBpm:                 141,
		TempoProbability:         0.58,
	}
	x.setStatus(status)
	expectStatus(t, live.Recv, status)
	expectStatus(t, x.subscribe().Recv, status)
}

func (x *relayFixture) subscribeGrpcWeb(contentType string) func() (*sspb.Recorder, error) {
	x.t.Helper()
	text := strings.HasPrefix(contentType, "application/grpc-web-text")

	// One length-prefixed message: an empty StreamRecordersRequest.
	body := []byte{0, 0, 0, 0, 0}
	if text {
		body = []byte(base64.StdEncoding.EncodeToString(body))
	}

	before := x.broadcaster.SubscriberCount()
	ctx, cancel := context.WithTimeout(context.Background(), relayTimeout)
	x.t.Cleanup(cancel)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		x.grpcWebURL+"/sessionsource.SessionSource/StreamRecorders", bytes.NewReader(body))
	if err != nil {
		x.t.Fatalf("build gRPC-Web request: %v", err)
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", contentType)
	request.Header.Set("X-Grpc-Web", "1")

	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		x.t.Fatalf("gRPC-Web request: %v", err)
	}
	x.t.Cleanup(func() { response.Body.Close() })
	if response.StatusCode != http.StatusOK {
		x.t.Fatalf("gRPC-Web request: HTTP %s", response.Status)
	}
	if got := response.Header.Get("Content-Type"); got != contentType {
		x.t.Fatalf("gRPC-Web response Content-Type = %q, want %q", got, contentType)
	}
	x.waitForSubscribers(before + 1)

	var frames io.Reader = response.Body
	if text {
		frames = &base64QuadReader{source: response.Body}
	}

	return func() (*sspb.Recorder, error) {
		// A frame is a flag byte, a big-endian uint32 length and the payload.
		var header [5]byte
		if _, err := io.ReadFull(frames, header[:]); err != nil {
			return nil, fmt.Errorf("read gRPC-Web frame header: %w", err)
		}
		payload := make([]byte, binary.BigEndian.Uint32(header[1:]))
		if _, err := io.ReadFull(frames, payload); err != nil {
			return nil, fmt.Errorf("read gRPC-Web frame of %d bytes: %w", len(payload), err)
		}
		if header[0]&0x80 != 0 {
			return nil, fmt.Errorf("stream ended with trailers: %q", payload)
		}
		recorder := &sspb.Recorder{}
		if err := proto.Unmarshal(payload, recorder); err != nil {
			return nil, fmt.Errorf("decode gRPC-Web frame of %d bytes: %w", len(payload), err)
		}
		return recorder, nil
	}
}

// base64QuadReader decodes a grpc-web-text body. The server base64-encodes
// each flush on its own, so padding can appear anywhere in the stream; four
// characters at a time always decode correctly.
type base64QuadReader struct {
	source  io.Reader
	decoded []byte
}

func (r *base64QuadReader) Read(p []byte) (int, error) {
	for len(r.decoded) == 0 {
		var quad [4]byte
		if _, err := io.ReadFull(r.source, quad[:]); err != nil {
			return 0, err
		}
		decoded, err := base64.StdEncoding.DecodeString(string(quad[:]))
		if err != nil {
			return 0, err
		}
		r.decoded = decoded
	}
	n := copy(p, r.decoded)
	r.decoded = r.decoded[n:]
	return n, nil
}
