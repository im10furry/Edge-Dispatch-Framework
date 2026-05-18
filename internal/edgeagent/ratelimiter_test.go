package edgeagent

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestNewBandwidthLimiter_ZeroMbps(t *testing.T) {
	t.Parallel()

	bl := NewBandwidthLimiter(0)
	if bl == nil {
		t.Fatal("expected non-nil BandwidthLimiter for 0 Mbps")
	}

	ctx := context.Background()
	r := bl.LimitReader(ctx, bytes.NewReader([]byte("x")))
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if err == nil {
		t.Fatal("expected error from zero-rate limiter (burst=0)")
	}
}

func TestNewBandwidthLimiter_10Mbps(t *testing.T) {
	t.Parallel()

	bl := NewBandwidthLimiter(10)
	if bl == nil {
		t.Fatal("expected non-nil BandwidthLimiter for 10 Mbps")
	}

	ctx := context.Background()
	data := []byte("hello world")
	r := bl.LimitReader(ctx, bytes.NewReader(data))
	buf := make([]byte, len(data))
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != len(data) {
		t.Errorf("read %d bytes, want %d", n, len(data))
	}
	if !bytes.Equal(buf[:n], data) {
		t.Errorf("content mismatch: got %q, want %q", buf[:n], data)
	}
}

func TestSetBandwidth(t *testing.T) {
	t.Parallel()

	bl := NewBandwidthLimiter(10)

	ctx := context.Background()
	data := []byte("hello")
	r := bl.LimitReader(ctx, bytes.NewReader(data))
	buf := make([]byte, 10)
	_, err := r.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error before SetBandwidth: %v", err)
	}

	bl.SetBandwidth(0)

	r2 := bl.LimitReader(ctx, bytes.NewReader([]byte("x")))
	buf2 := make([]byte, 1)
	_, err = r2.Read(buf2)
	if err == nil {
		t.Fatal("expected error after SetBandwidth(0)")
	}
}

func TestLimitReader_Throttled(t *testing.T) {
	t.Parallel()

	bl := NewBandwidthLimiter(1)

	dataSize := 200 * 1024
	data := make([]byte, dataSize)

	ctx := context.Background()
	r := bl.LimitReader(ctx, bytes.NewReader(data))

	start := time.Now()
	buf := make([]byte, 32*1024)
	n, err := io.CopyBuffer(io.Discard, r, buf)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != int64(dataSize) {
		t.Errorf("copied %d bytes, want %d", n, dataSize)
	}

	if elapsed < 100*time.Millisecond {
		t.Errorf("expected throttling (≥100ms), got %v", elapsed)
	}
}
