package main

import (
	"bytes"
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

	// prepare requesters
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

	// run requesters
	var (
		wg  sync.WaitGroup
		ctx = context.Background()
	)
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

	// print results
	result := ResultFromRequesters(requesters, duration)

	fmt.Println(result.Text())
	if showGraph {
		printTerminalGraph(result)
	}

}

type Result struct {
	Duration               time.Duration
	ConnectionsTotal       int
	RequestsTotal          int
	RequestsPerSecond      float64
	RequestDurationTotal   time.Duration
	RequestDurationAverage time.Duration
	RequestDurationMin     time.Duration
	RequestDurationMax     time.Duration
	RequestDurationP95     time.Duration
	RequestDurationP99     time.Duration
	Durations              []time.Duration
}

func ResultFromRequesters(requesters []*Requester, duration time.Duration) *Result {
	result := &Result{
		RequestDurationMin: time.Hour,
		Duration:           duration,
	}

	for _, requester := range requesters {
		result.RequestsTotal += len(requester.requests)

		result.ConnectionsTotal += len(requester.connections)

		for _, request := range requester.requests {
			if request.Duration < result.RequestDurationMin {
				result.RequestDurationMin = request.Duration
			}
			if request.Duration > result.RequestDurationMax {
				result.RequestDurationMax = request.Duration
			}
			result.RequestDurationTotal += request.Duration
			result.Durations = append(result.Durations, request.Duration)
		}
	}

	result.RequestsPerSecond = float64(result.RequestsTotal) / result.Duration.Seconds()
	result.RequestDurationAverage = result.RequestDurationTotal / time.Duration(result.RequestsTotal)

	sort.Slice(result.Durations, func(i, j int) bool { return result.Durations[i] < result.Durations[j] })
	indexP95 := percentile(len(result.Durations), 0.95)
	indexP99 := percentile(len(result.Durations), 0.99)
	result.RequestDurationP95 = result.Durations[indexP95]
	result.RequestDurationP99 = result.Durations[indexP99]

	return result
}

func (r *Result) Text() string {
	out := &bytes.Buffer{}

	fmt.Fprintf(out, "total connections: %d\n", r.ConnectionsTotal)
	fmt.Fprintf(out, "requests:\n")
	fmt.Fprintf(out, "  total %d\n", r.RequestsTotal)
	fmt.Fprintf(out, "  throughput %.2f req/s\n", r.RequestsPerSecond)
	fmt.Fprintf(out, "request duration:\n")
	fmt.Fprintf(out, "  avg %s\n", r.RequestDurationAverage)
	fmt.Fprintf(out, "  min %s\n", r.RequestDurationMin)
	fmt.Fprintf(out, "  max %s\n", r.RequestDurationMax)
	fmt.Fprintf(out, "  p95 %s\n", r.RequestDurationP95)
	fmt.Fprintf(out, "  p99 %s\n", r.RequestDurationP99)
	return out.String()
}

func printTerminalGraph(r *Result) {
	chart := goterm.NewLineChart(100, 20)
	data := &goterm.DataTable{}
	data.AddColumn("request")
	data.AddColumn("duration")

	for i, duration := range r.Durations {
		data.AddRow(float64(i), duration.Seconds())
	}

	goterm.Println(chart.Draw(data))
	goterm.Flush()
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
