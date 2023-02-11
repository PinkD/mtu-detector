package main

import (
	"context"
	"log"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	ListenAddrStr string `short:"l" long:"listen" description:"server mode, listen address" required:"false"`
	RemoteAddr    string `short:"r" long:"remote" description:"client mode, remote address" required:"false"`
	MTU           string `short:"m" long:"mtu" description:"single value will only overwrite the max, two values should be concat with '-', default is 1024-1500" required:"false"`
	Count         uint32 `short:"c" long:"count" description:"client only, send count for every test, avoid packet loss, default is 3" required:"false" default:"3"`
	Timeout       uint32 `short:"w" long:"timeout" description:"client only, wait timeout in second, default is 3" required:"false" default:"3"`
}

func parseMTU(mtu string) uint32 {
	u, err := strconv.ParseUint(mtu, 10, 32)
	if err != nil {
		panic(err)
	}
	return uint32(u)
}

var pool sync.Pool
var minMTU = uint32(1024)
var maxMTU = uint32(1500)

func main() {
	var args, err = flags.ParseArgs(&opts, os.Args[1:])
	if err != nil {
		return
	}
	if len(args) != 0 {
		log.Printf("unknown args: %v", args)
	}
	if len(opts.ListenAddrStr) != 0 && len(opts.RemoteAddr) != 0 {
		log.Print("cannot set listen and remote address at the same time")
		os.Exit(1)
	}
	if len(opts.MTU) != 0 {
		max := opts.MTU
		ss := strings.Split(opts.MTU, "-")
		if len(ss) == 2 {
			minMTU = parseMTU(ss[0])
			max = ss[1]
		}
		maxMTU = parseMTU(max)
	}
	pool.New = func() any {
		return new(Payload)
	}
	switch {
	case len(opts.ListenAddrStr) != 0:
		server()
	case len(opts.RemoteAddr) != 0:
		client()
	default:
		log.Print("listen or remote address is required")
		os.Exit(1)
	}
}

func avg(a, b uint32) uint32 {
	c := a + b
	if c%2 == 0 {
		return c / 2
	}
	// +1 to make mtu always closer to max
	return c/2 + 1
}

func client() {
	serverAddr, err := netip.ParseAddrPort(opts.RemoteAddr)
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(serverAddr))
	if err != nil {
		panic(err)
	}
	err = setDF(conn)
	if err != nil {
		panic(err)
	}
	mtuChan := make(chan uint32)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		mtu := minMTU
		buff := make([]byte, maxMTU)
		for {
			n, err := conn.Read(buff)
			if err != nil {
				if err != net.ErrClosed {
					return
				}
				log.Printf("read packet: %s", err)
			}
			log.Printf("response received, size is: %d", n)
			p := pool.Get().(*Payload)
			p.Decode(buff)
			if p.Current == 0 {
				log.Print("zero mtu, the packet may be corrupted")
				continue
			}
			log.Printf("packet from %s, mtu is: %d", serverAddr, p.Current)
			if mtu != p.Current {
				mtuChan <- p.Current
				mtu = p.Current
			}
		}
	}()
	count := opts.Count
	go func() {
		defer wg.Done()
		min := minMTU
		max := maxMTU
		mtu := avg(max, min)
		ctx := context.TODO()
		timeout := time.Duration(opts.Timeout) * time.Second
	Next:
		for {
			for i := uint32(1); i <= count; i++ {
				p := &Payload{
					Index:   i,
					Total:   count,
					Current: mtu,
					Min:     min,
					Max:     max,
				}
				log.Printf("trying mtu %d", mtu)
				n, err := conn.Write(p.Encode())
				if err != nil {
					if isMsgSizeErr(err) {
						// mtu reach interface limit
						log.Printf("mtu %d reaches interface limit", mtu)
						max = mtu
						mtu = avg(max, min)
						if mtu == max {
							break
						}
						continue Next
					}
					log.Printf("send packet with mtu %d: %s", mtu, err)
				}
				basicHeaderSize := udpHeaderSize + ipHeaderSize
				if uint32(n)+basicHeaderSize != mtu {
					log.Printf("send packet with mtu %d, but write size %d(+%dheader) not equal",
						mtu, n, basicHeaderSize)
				}
			}
			tctx, cancel := context.WithTimeout(ctx, timeout)
			select {
			case min = <-mtuChan:
				cancel()
			case <-tctx.Done():
				// timeout, too large, set max to mtu
				max = mtu
				cancel()
			}
			if max-min <= 1 {
				_ = conn.Close()
				log.Printf("mtu is: %d, data size is: %d", min, min-basicHeaderSize)
				os.Exit(0)
			}
			mtu = avg(max, min)
		}
	}()
	wg.Wait()
}

func server() {
	listenAddr, err := net.ResolveUDPAddr("udp", opts.ListenAddrStr)
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		panic(err)
	}
	err = setDF(conn)
	if err != nil {
		panic(err)
	}
	log.Printf("listen on %s", listenAddr)
	buff := make([]byte, maxMTU)
	for {
		n, addr, err := conn.ReadFrom(buff)
		if err != nil {
			panic(err)
		}
		p := pool.Get().(*Payload)
		p.Decode(buff)
		if p.Current == 0 {
			log.Print("zero mtu, the packet may be corrupted")
			continue
		}
		log.Printf("packet from %s, mtu is: %d", addr, p.Current)
		nn, err := conn.WriteTo(buff[:n], addr)
		if err != nil {
			if !isMsgSizeErr(err) {
				panic(err)
			}
			log.Printf("reach limit for mtu %d %s", p.Current, err)
			continue
		}
		if nn != n {
			log.Printf("send packet size %d, but only send %d", n, nn)
		}
	}
}
