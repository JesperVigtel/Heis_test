// Package bcast implements UDP broadcast messaging for arbitrary Go types.
// It uses JSON encoding and sends/receives on a fixed UDP port.
package bcast

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"time"
)

const broadcastAddr = "255.255.255.255"

// Transmitter broadcasts values received on ch to the given UDP port.
func Transmitter[T any](port int, ch <-chan T) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", broadcastAddr, port))
	if err != nil {
		panic(fmt.Sprintf("bcast Transmitter: ResolveUDPAddr: %v", err))
	}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		panic(fmt.Sprintf("bcast Transmitter: DialUDP: %v", err))
	}
	defer conn.Close()

	for msg := range ch {
		b, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("bcast Transmitter: marshal error: %v\n", err)
			continue
		}
		_, err = conn.Write(b)
		if err != nil {
			fmt.Printf("bcast Transmitter: write error: %v\n", err)
		}
	}
}

// Receiver listens for UDP broadcasts on the given port and sends received
// values to ch.
func Receiver[T any](port int, ch chan<- T) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("bcast Receiver: ResolveUDPAddr: %v", err))
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		panic(fmt.Sprintf("bcast Receiver: ListenUDP: %v", err))
	}
	defer conn.Close()

	var buf [65536]byte
	for {
		n, _, err := conn.ReadFromUDP(buf[:])
		if err != nil {
			fmt.Printf("bcast Receiver: read error: %v\n", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var msg T
		if err := json.Unmarshal(buf[:n], &msg); err != nil {
			fmt.Printf("bcast Receiver: unmarshal error (type %v): %v\n", reflect.TypeOf(msg), err)
			continue
		}
		ch <- msg
	}
}
