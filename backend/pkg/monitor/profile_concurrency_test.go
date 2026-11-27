package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

type profileBlockingReadClient struct {
	active, maximum atomic.Int32
	started         chan time.Duration
	release         <-chan struct{}
}

func (*profileBlockingReadClient) URL(string, map[string]string) *url.URL {
	return &url.URL{Scheme: "http", Host: "prometheus"}
}

func (c *profileBlockingReadClient) Do(ctx context.Context, _ *http.Request) (*http.Response, []byte, error) {
	active := c.active.Add(1)
	defer c.active.Add(-1)
	for maximum := c.maximum.Load(); active > maximum; maximum = c.maximum.Load() {
		if c.maximum.CompareAndSwap(maximum, active) {
			break
		}
	}
	deadline, _ := ctx.Deadline()
	c.started <- time.Until(deadline)
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-c.release:
	}
	response := &prompb.ReadResponse{Results: []*prompb.QueryResult{{}}}
	body, err := response.Marshal()
	return &http.Response{StatusCode: http.StatusOK}, snappy.Encode(nil, body), err
}

func TestProfileRemoteReadsShareProcessConcurrencyLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const requests = 16
		release := make(chan struct{})
		remote := &profileBlockingReadClient{started: make(chan time.Duration, requests), release: release}
		results := make(chan error, requests)
		var workers sync.WaitGroup
		for range requests {
			workers.Go(func() {
				p := &PrometheusClient{client: remote}
				_, err := p.readProfileChunk(t.Context(), &prompb.Query{})
				results <- err
			})
		}
		synctest.Wait()
		active, maximum := remote.active.Load(), remote.maximum.Load()
		close(release)
		workers.Wait()
		close(results)
		for err := range results {
			if err != nil {
				t.Error(err)
			}
		}
		if active != profileReadConcurrency || maximum != profileReadConcurrency || remote.maximum.Load() != profileReadConcurrency {
			t.Fatalf("process-wide Remote Read limit exceeded or underused: blocked=%d, maximum=%d, final maximum=%d",
				active, maximum, remote.maximum.Load())
		}
	})
}

func TestProfileRemoteReadSlotWaitHonorsCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if err := profileReadSlots.Acquire(t.Context(), profileReadConcurrency); err != nil {
			t.Fatal(err)
		}
		defer profileReadSlots.Release(profileReadConcurrency)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		remote := &profileBlockingReadClient{started: make(chan time.Duration, 1)}
		p := &PrometheusClient{client: remote}
		result := make(chan error, 1)
		go func() {
			_, err := p.readProfileChunk(ctx, &prompb.Query{})
			result <- err
		}()
		synctest.Wait()
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) || len(remote.started) != 0 {
			t.Fatalf("canceled semaphore wait reached Remote Read or lost cancellation: %v", err)
		}
	})
}

func TestProfileRemoteReadTimeoutStartsAfterSlotAcquisition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		if err := profileReadSlots.Acquire(t.Context(), profileReadConcurrency); err != nil {
			t.Fatal(err)
		}
		release := make(chan struct{})
		close(release)
		remote := &profileBlockingReadClient{started: make(chan time.Duration, 1), release: release}
		p := &PrometheusClient{client: remote}
		ctx, cancel := context.WithTimeout(t.Context(), profileQueryTimeout)
		defer cancel()
		result := make(chan error, 1)
		go func() {
			_, err := p.readProfileChunk(ctx, &prompb.Query{})
			result <- err
		}()
		synctest.Wait()
		time.Sleep(queryTimeout + time.Second)
		profileReadSlots.Release(profileReadConcurrency)
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if remaining := <-remote.started; remaining != queryTimeout {
			t.Fatalf("slot wait consumed shard timeout: remaining=%v, want=%v", remaining, queryTimeout)
		}
	})
}
