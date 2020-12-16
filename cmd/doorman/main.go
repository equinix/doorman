package main

import (
	"flag"
	"os"

	"github.com/equinix/doorman"
	"github.com/packethost/pkg/log"
)

var (
	serve bool

	authenticate bool
	openvpnFile  string

	disconnect   bool
	client       string
	connectingIP string
)

func init() {
	// serve flags
	flag.BoolVar(&serve, "s", false, "Serve the VPN gRPC service on the specified IP and port")
	flag.Parse()
}

func main() {
	l, err := log.Init("github.com/equinix/doorman")
	if err != nil {
		panic(err)
	}
	defer l.Close()

	if serve {
		// TODO: turn this into an actual daemon
		doorman.ServeVPN(l)
	} else {
		flag.Usage()
		os.Exit(1)
	}
	os.Exit(0)
}
