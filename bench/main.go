package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const TargetAddr = "127.0.0.1:1234"

func main() {
	var (
		connections     = 1
		requestSize     = 1024
		duration        = time.Second * 1
		requestsPerConn = 0
	)
	flag.IntVar(&connections, "c", connections, "number of parallel connections")
	flag.IntVar(&requestSize, "s", requestSize, "request size in bytes")
	flag.IntVar(&requestsPerConn, "r", requestsPerConn, "how many requests do we send through one connection. if zero only one connection is used for all requests")
	flag.DurationVar(&duration, "d", duration, "how long we run the test")
	flag.Parse()

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(duration))

	var wg sync.WaitGroup
	requests := make([]int, connections)

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(i int) {
			var err error
			requests[i], err = test(ctx, requestSize, requestsPerConn)
			if err != nil {
				log.Println(err)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	total := 0
	min := float64(requests[0]) / duration.Seconds()
	max := 0.0
	for _, request := range requests {
		req := float64(request) / duration.Seconds()
		if req < min {
			min = req
		}
		if req > max {
			max = req
		}
		total += request
	}
	fmt.Printf("total: %.2f req/s\n", float64(total)/duration.Seconds())
	fmt.Printf("per connection:\n  min %.2f req/s\n  max %.2f req/s\n  avg %.2f req/s\n", min, max, float64(total)/float64(connections)/duration.Seconds())
}

// test sends and reads bytes in chunks of size to the target until the context
// is canceled. After requestsPerConn requests the connection is closed an
// reopened. If requestPerConn is 0 all bytes are sent through the same
// connection.
func test(ctx context.Context, size int, requestsPerConn int) (int, error) {
	var (
		dialer   net.Dialer
		conn     net.Conn
		requests int
	)

	conn, err := dialer.DialContext(ctx, "tcp", TargetAddr)
	if err != nil {
		return requests, err
	}
	defer conn.Close()

	buf := make([]byte, size)

	for {
		select {
		case <-ctx.Done():
			return requests, nil
		default:
		}

		// reopen connection
		if requestsPerConn != 0 && requests%requestsPerConn == 0 {
			conn.Close()
			conn, err = dialer.DialContext(ctx, "tcp", TargetAddr)
			if err != nil {
				return requests, err
			}
		}

		n, err := conn.Write(buf)
		if err != nil {
			return requests, err
		}
		n, err = conn.Read(buf)
		if err != nil {
			return requests, fmt.Errorf("failed to read bytes: %w", err)
		}
		if n != size {
			return requests, fmt.Errorf("did not receive all bytes")
		}
		requests++
	}

	return requests, nil
}
