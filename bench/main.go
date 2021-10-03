package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/buger/goterm"
)

const DefaultTargetAddr = "127.0.0.1:1234"

func main() {
	var (
		connections     = 1
		requestSize     = 1024
		duration        = time.Second * 1
		requestsPerConn = 0
		target          string
		showGraph       = false
	)
	flag.IntVar(&connections, "c", connections, "number of parallel connections")
	flag.IntVar(&requestSize, "s", requestSize, "request jize in bytes")
	flag.IntVar(&requestsPerConn, "r", requestsPerConn, "how many requests do we send through one connection. if zero only one connection is used for all requests")
	flag.StringVar(&target, "t", DefaultTargetAddr, "target address")
	flag.DurationVar(&duration, "d", duration, "how long we run the test")
	flag.BoolVar(&showGraph, "g", showGraph, "show a graph with response times")
	flag.Parse()

	ctx := context.Background()

	var wg sync.WaitGroup
	requesters := make([]*Requester, connections)
	for i, _ := range requesters {
		requesters[i] = &Requester{
			id:              i,
			size:            requestSize,
			requestsPerConn: requestsPerConn,
			target:          target,
			done:            make(chan struct{}),
		}
	}

	for _, requester := range requesters {
		wg.Add(1)
		go func(r *Requester) {
			err := r.Run(ctx)
			if err != nil {
				log.Println(err)
			}
			wg.Done()
		}(requester)
	}
	time.Sleep(duration)
	for _, r := range requesters {
		r.Stop()
	}
	wg.Wait()
	var (
		totalRequestDuration time.Duration
		minDuration          = time.Hour
		maxDuration          time.Duration
	)
	totalRequests := 0
	totalConnections := 0
	requestDurations := []time.Duration{}
	min := float64(len(requesters[0].requests)) / duration.Seconds()
	max := 0.0
	for _, requester := range requesters {
		requests := len(requester.requests)
		req := float64(requests) / duration.Seconds()
		if req < min {
			min = req
		}
		if req > max {
			max = req
		}
		for _, request := range requester.requests {
			if request.Duration < minDuration {
				minDuration = request.Duration
			}
			if request.Duration > maxDuration {
				maxDuration = request.Duration
			}
			totalRequestDuration += request.Duration
			requestDurations = append(requestDurations, request.Duration)
		}
		totalRequests += requests
		totalConnections += len(requester.connections)
	}

	sort.Slice(requestDurations, func(i, j int) bool { return requestDurations[i] < requestDurations[j] })

	fmt.Printf("total connections: %d\n", totalConnections)
	fmt.Printf("requests:\n  total %d\n  throughput %.2f req/s\n", totalRequests, float64(totalRequests)/duration.Seconds())
	fmt.Printf("per connection:\n  min %.2f req/s\n  max %.2f req/s\n  avg %.2f req/s\n", min, max, float64(totalRequests)/float64(connections)/duration.Seconds())
	fmt.Printf("request duration:\n  min %s\n  max %s\n  avg %s\n", minDuration, maxDuration, totalRequestDuration/time.Duration(totalRequests))
	indexP95 := percentile(len(requestDurations), 0.95)
	indexP99 := percentile(len(requestDurations), 0.99)
	fmt.Printf("  p95 %s\n  p99 %s\n", requestDurations[indexP95], requestDurations[indexP99])
	fmt.Printf("min %s  %d\n", requestDurations[0], len(requestDurations))

	if showGraph {
		chart := goterm.NewLineChart(100, 20)
		data := &goterm.DataTable{}
		data.AddColumn("request")
		data.AddColumn("duration")

		for i, dur := range requestDurations {
			data.AddRow(float64(i), dur.Seconds())
		}

		goterm.Println(chart.Draw(data))
		goterm.Flush()
	}
}

func percentile(i int, p float64) int {
	return int(math.Round(float64(i)*p+0.5)) - 1
}

type Request struct {
	Duration time.Duration
}

type Requester struct {
	id              int
	dialer          net.Dialer
	target          string
	size            int
	requestsPerConn int
	conn            net.Conn
	// indexes to requests
	connections []int
	requests    []Request
	done        chan struct{}
}

func (r *Requester) Stop() {
	close(r.done)
}

// Run sends requests to target until context is canceled
func (r *Requester) Run(ctx context.Context) error {
	buf := make([]byte, r.size)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.done:
			r.conn.Close()
			return nil
		default:
		}
		if len(r.requests) == 0 || r.requestsPerConn != 0 && len(r.requests)%r.requestsPerConn == 0 {
			err := r.establishConn(ctx)
			if err != nil {
				return fmt.Errorf("failed to establish connection (%d): %w", r.id, err)
			}
		}

		start := time.Now()
		n, err := r.conn.Write(buf)
		if err != nil {
			return fmt.Errorf("failed to write (%d): %w", r.id, err)
		}
		received := 0
		for {
			n, err = r.conn.Read(buf)
			if err != nil {
				return fmt.Errorf("failed to read bytes (%d): %w", r.id, err)
			}
			received += n
			if received == r.size {
				break
			}
		}
		end := time.Now()
		r.requests = append(r.requests, Request{end.Sub(start)})
	}
}

func (r *Requester) establishConn(ctx context.Context) error {
	var err error
	if r.conn != nil {
		_ = r.conn.Close()
	}
	r.connections = append(r.connections, len(r.requests))
	r.conn, err = r.dialer.DialContext(ctx, "tcp", r.target)
	return err
}
