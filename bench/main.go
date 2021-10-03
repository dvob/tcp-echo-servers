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

func main() {
	var (
		connections = 1
		requestSize = 1024
		duration    = time.Second * 1
	)
	flag.IntVar(&connections, "c", connections, "number of parallel connections")
	flag.IntVar(&requestSize, "s", requestSize, "request size in bytes")
	flag.DurationVar(&duration, "d", duration, "how long we run the test")
	flag.Parse()

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(duration))

	var wg sync.WaitGroup
	requests := make([]int, connections)

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func(i int) {
			var err error
			requests[i], err = test(ctx, requestSize)
			if err != nil {
				log.Println(err)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	total := 0
	min := requests[0]
	max := 0
	for _, req := range requests {
		if req < min {
			min = req
		}
		if req > max {
			max = req
		}
		total += req
	}
	fmt.Printf("%.2f req/s\n", float64(total)/duration.Seconds())
	fmt.Printf("reqest per connection: min=%d, max=%d, avg=%d\n", min, max, total/connections)
}

func test(ctx context.Context, size int) (int, error) {
	requests := 0
	conn, err := net.Dial("tcp", "127.0.0.1:1234")
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
