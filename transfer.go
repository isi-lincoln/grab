package grab

import (
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/isi-lincoln/grab/bps"
)

type transfer struct {
	n     int64 // must be 64bit aligned on 386
	ctx   context.Context
	gauge bps.Gauge
	lim   RateLimiter
	w     io.Writer
	r     io.Reader
	b     []byte
}

func newTransfer(ctx context.Context, lim RateLimiter, dst io.Writer, src io.Reader, buf []byte) *transfer {
	return &transfer{
		ctx:   ctx,
		gauge: bps.NewSMA(6), // five second moving average sampling every second
		lim:   lim,
		w:     dst,
		r:     src,
		b:     buf,
	}
}

// copy behaves similarly to io.CopyBuffer except that it checks for cancelation
// of the given context.Context, reports progress in a thread-safe manner and
// tracks the transfer rate.
func (c *transfer) copy() (written int64, err error) {
	// maintain a bps gauge in another goroutine
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	go bps.Watch(ctx, c.gauge, c.N, time.Second)

	// start the transfer
	if c.b == nil {
		c.b = make([]byte, 64*1024)
	}
	for {
		select {
		case <-c.ctx.Done():
			err = c.ctx.Err()
			return
		default:
			// keep working
		}

		nr, er := io.Copy(c.w, c.r)
		if er != nil {
			if er != io.EOF {
				err = er
			}
			return
		}
		//nr, er := c.r.Read(c.b)
		if nr > 0 {
			//nw, ew := c.w.Write(c.b[0:nr])
			written += int64(nw)
			atomic.StoreInt64(&c.n, written)
		}
	}
	return
}

// N returns the number of bytes transferred.
func (c *transfer) N() (n int64) {
	if c == nil {
		return 0
	}
	n = atomic.LoadInt64(&c.n)
	return
}

// BPS returns the current bytes per second transfer rate using a simple moving
// average.
func (c *transfer) BPS() (bps float64) {
	if c == nil || c.gauge == nil {
		return 0
	}
	return c.gauge.BPS()
}
