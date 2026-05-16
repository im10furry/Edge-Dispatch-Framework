package edgeagent

import (
	"context"
	"io"
	"sync"

	"golang.org/x/time/rate"
)

type BandwidthLimiter struct {
	limiter   *rate.Limiter
	burstSize int
	mu        sync.Mutex
}

func NewBandwidthLimiter(bandwidthMbps int) *BandwidthLimiter {
	bytesPerSec := bandwidthMbps * 1024 * 1024 / 8
	return &BandwidthLimiter{
		limiter:   rate.NewLimiter(rate.Limit(bytesPerSec), bytesPerSec),
		burstSize: bytesPerSec,
	}
}

type limitedReader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

func (lr *limitedReader) Read(p []byte) (n int, err error) {
	n, err = lr.r.Read(p)
	if n > 0 {
		if limErr := lr.limiter.WaitN(lr.ctx, n); limErr != nil {
			return n, limErr
		}
	}
	return n, err
}

func (bl *BandwidthLimiter) LimitReader(ctx context.Context, r io.Reader) io.Reader {
	return &limitedReader{r: r, limiter: bl.limiter, ctx: ctx}
}

func (bl *BandwidthLimiter) SetBandwidth(bandwidthMbps int) {
	bl.mu.Lock()
	defer bl.mu.Unlock()
	bytesPerSec := bandwidthMbps * 1024 * 1024 / 8
	bl.limiter.SetLimit(rate.Limit(bytesPerSec))
	bl.limiter.SetBurst(bytesPerSec)
}
