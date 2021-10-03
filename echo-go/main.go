package main

import (
	"fmt"
	"io"
	"log"
	"net"
)

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ln, err := net.Listen("tcp", "127.0.0.1:1234")
	if err != nil {
		return err
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("accept failed: %w", err)
		}
		go func(c net.Conn) {
			//log.Printf("new connection from %s", c.RemoteAddr())
			_, err := io.Copy(c, c)
			if err != nil {
				log.Printf("failed to transfer byes to %s: %s", c.RemoteAddr(), err)
				return
			}
			conn.Close()
			//log.Printf("transferred %d to %s", n, c.RemoteAddr())
		}(conn)
	}
	return nil
}
