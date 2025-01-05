package runner

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Runner struct {
	Rate     int
	Workers  int
	jobsChan chan string
	wg       sync.WaitGroup
}

func New(rate, workers int) *Runner {
	if workers <= 0 {
		workers = 1
	}
	return &Runner{
		Rate:     rate,
		Workers:  workers,
		jobsChan: make(chan string, workers),
	}
}

func (r *Runner) Run(ctx context.Context, urls []string, fn func(string) error) error {
	var delay time.Duration
	if r.Rate > 0 {
		delay = time.Duration(time.Second.Nanoseconds() / int64(r.Rate))
	}

	// Create fixed-size worker pool
	for i := 0; i < r.Workers; i++ {
		r.wg.Add(1)
		go r.worker(ctx, delay, fn)
	}

	go func() {
		for _, url := range urls {
			select {
			case <-ctx.Done():
				return
			case r.jobsChan <- url:
				// No delay when rate is 0 (unlimited)
			}
		}
		close(r.jobsChan)
	}()

	r.wg.Wait()
	return nil
}

func (r *Runner) worker(ctx context.Context, delay time.Duration, fn func(string) error) {
	defer r.wg.Done()

	var ticker *time.Ticker
	if delay > 0 {
		ticker = time.NewTicker(delay)
		defer ticker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case url, ok := <-r.jobsChan:
			if !ok {
				return
			}
			if err := fn(url); err != nil {
				fmt.Printf("[ERROR] Processing URL %s: %v\n", url, err)
			}
			if delay > 0 {
				<-ticker.C
			}
		}
	}
}
