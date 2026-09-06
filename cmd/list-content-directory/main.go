// SPDX-FileCopyrightText: 2020 Ethel Morgan
// SPDX-FileCopyrightText: 2026 TorrPlay
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ethulhu/helix/upnp"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/contentdirectory"
)

var (
	object = flag.String("object", "0", "object to list (0 means root)")
	server = flag.String("server", "", "name of server to list")

	startingIndex  = flag.Uint("starting-index", 0, "starting index to enumerate children")
	requestedCount = flag.Uint("requested-count", 0, "requested number of entries (0 means all)")

	ifaceName = flag.String("interface", "", "network interface to discover on (optional)")
)

func main() {
	flag.Parse()

	if *server == "" {
		log.Fatal("must set -server")
	}

	var iface *net.Interface
	if *ifaceName != "" {
		var err error
		iface, err = net.InterfaceByName(*ifaceName)
		if err != nil {
			log.Fatalf("could not find interface %s: %v", *ifaceName, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	devices, _, err := upnp.DiscoverDevices(ctx, contentdirectory.Version1, iface)
	cancel()
	if err != nil {
		log.Fatalf("could not discover ContentDirectory clients: %v", err)
	}

	var directory contentdirectory.Interface
	for _, device := range devices {
		if client, ok := device.SOAPInterface(contentdirectory.Version1); ok && device.UDN == *server {
			directory = contentdirectory.NewClient(client)
			break
		}
	}
	if directory == nil {
		log.Fatalf("could not find ContentDirectory server %v", *server)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	didl, _, err := directory.BrowseChildren(ctx, upnpav.ObjectID(*object), *startingIndex, *requestedCount, nil)
	if err != nil {
		log.Fatalf("could not list ContentDirectory root: %v", err)
	}

	for _, collection := range didl.Containers {
		fmt.Printf("%+v\n", collection)
	}
	for _, item := range didl.Items {
		fmt.Printf("%+v\n", item)
	}
}
