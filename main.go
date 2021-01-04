package main

// const targetIP = "8.8.8.8"

// var ListenAddr = "0.0.0.0"

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	// Stolen from https://godoc.org/golang.org/x/net/internal/iana,
	// can't import "internal" packages
	ProtocolICMP = 1
)

var ListenAddr = "0.0.0.0"

func Ping(addr string) (*net.IPAddr, time.Duration, error) {
	// Start listening for icmp replies
	packetConnection, err := icmp.ListenPacket("ip4:icmp", ListenAddr)
	if err != nil {
		return nil, 0, err
	}
	defer packetConnection.Close()

	// Resolve any DNS (if used) and get the real IP of the target
	dst, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		panic(err)
	}

	// Make a new ICMP message
	// The Identifier and Sequence Number can be used by the client
	// to match the reply with the request that caused the reply.
	// In practice, most Linux systems use a unique identifier for every ping process,
	// and sequence number is an increasing number within that process.
	// Windows uses a fixed identifier, which varies between Windows versions, and a sequence number that is only reset at boot time.
	// https://en.wikipedia.org/wiki/Ping_(networking_utility)#:~:text=The%20echo%20request%20(%22ping%22,is%20an%20ICMP%2FICMP6%20message.&text=The%20Identifier%20and%20Sequence%20Number,increasing%20number%20within%20that%20process.
	icmpMessage := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Body: &icmp.Echo{
			ID: os.Getpid(), Seq: 1,
			Data: []byte(""),
		},
	}
	message, err := icmpMessage.Marshal(nil)
	if err != nil {
		return dst, 0, err
	}

	// Send it
	start := time.Now()
	n, err := packetConnection.WriteTo(message, dst)
	if err != nil {
		return dst, 0, err
	} else if n != len(message) {
		return dst, 0, fmt.Errorf("got %v; want %v", n, len(message))
	}

	// Wait for a reply
	reply := make([]byte, 1500)
	err = packetConnection.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return dst, 0, err
	}
	n, peer, err := packetConnection.ReadFrom(reply)
	if err != nil {
		return dst, 0, err
	}
	duration := time.Since(start)

	// Pack it up boys, we're done here
	rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return dst, 0, err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return dst, duration, nil
	default:
		return dst, 0, fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
}

func main() {
	p := func(addr string) {
		dst, dur, err := Ping(addr)
		if err != nil {
			log.Printf("Ping %s (%s): %s\n", addr, dst, err)
			return
		}
		log.Printf("Ping %s (%s): %s\n", addr, dst, dur)
	}
	// p("127.0.0.1")
	// p("172.27.0.1")
	p("google.com")
	// p("reddit.com")
	// p("www.gp.se")

	//for {
	//    p("google.com")
	//    time.Sleep(1 * time.Second)
	//
}
