// SPDX-FileCopyrightText: 2020 Ethel Morgan
//
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethulhu/helix/flag"
	"github.com/ethulhu/helix/flags"
	"github.com/ethulhu/helix/logger"
	"github.com/ethulhu/helix/media"
	"github.com/ethulhu/helix/netutil"
	"github.com/ethulhu/helix/upnp"
	"github.com/ethulhu/helix/upnpav/connectionmanager"
	"github.com/ethulhu/helix/upnpav/contentdirectory"
	"github.com/ethulhu/helix/upnpav/contentdirectory/jackalope"

	jackalopeDB "go.eth.moe/jackalope"
)

var (
	udn            = flag.Custom("udn", "", "UDN to broadcast (if unset, will generate one)", flags.UDN)
	friendlyName   = flag.Custom("friendly-name", "", "human-readable name to broadcast (if unset, will generate one)", flags.FriendlyName)
	iface          = flag.Custom("interface", "", "interface to listen on (will try to find a Private IPv4 if unset)", flags.NetInterface)
	notifyInterval = flag.Duration("notify-interval", time.Duration(30*time.Second), "interval between SSDP advertisements")

	basePath      = flag.Custom("path", "", "path to serve", flag.RequiredString)
	jackalopePath = flag.Custom("jackalope-path", "", "path to Jackalope db", flag.RequiredString)

	disableMetadataCache = flag.Bool("disable-metadata-cache", false, "disable the metadata cache")
)

func main() {
	flag.Parse()

	friendlyName := (*friendlyName).(string)
	iface := (*iface).(*net.Interface)
	udn := (*udn).(string)

	basePath := (*basePath).(string)
	jackalopePath := (*jackalopePath).(string)

	ctx := context.Background()
	log, _ := logger.FromContext(ctx)

	ip, err := netutil.SuitableIP(iface)
	if err != nil {
		name := "ALL"
		if iface != nil {
			name = iface.Name
		}

		log.AddField("interface", name)
		log.WithError(err).Fatal("could not find suitable serving IP")
	}
	addr := &net.TCPAddr{
		IP: ip,
	}

	httpConn, err := net.Listen("tcp", addr.String())
	if err != nil {
		log.AddField("listener", addr)
		log.WithError(err).Fatal("could not create HTTP listener")
	}
	defer httpConn.Close()

	device := &upnp.Device{
		Name:             friendlyName,
		UDN:              udn,
		DeviceType:       contentdirectory.DeviceType,
		Manufacturer:     "Eth Morgan",
		ManufacturerURL:  "https://ethulhu.co.uk",
		ModelDescription: "Helix with Jackalope",
		ModelName:        "helix-directory-jackalope",
		ModelNumber:      "420",
		ModelURL:         "https://ethulhu.co.uk",
		SerialNumber:     "00000000",
	}

	metadataCache := media.NewMetadataCache()
	if *disableMetadataCache {
		metadataCache = media.NoOpCache{}
	}

	jackalopeDB, err := jackalopeDB.Open(jackalopePath)
	if err != nil {
		log.WithError(err).Fatal("could not open Jackalope DB")
	}

	cd, err := jackalope.NewContentDirectory(basePath, fmt.Sprintf("http://%v/objects/", httpConn.Addr()), metadataCache, jackalopeDB)
	if err != nil {
		log.WithError(err).Fatal("could not create ContentDirectory object")
	}

	device.Handle(contentdirectory.Version1, contentdirectory.ServiceID, contentdirectory.SCPD, contentdirectory.SOAPHandler{cd})
	device.Handle(connectionmanager.Version1, connectionmanager.ServiceID, connectionmanager.SCPD, nil)

	mux := http.NewServeMux()
	mux.Handle("/objects/", http.StripPrefix("/objects/", http.FileServer(http.Dir(basePath))))
	mux.Handle("/upnp/", http.StripPrefix("/upnp", device.HTTPHandler("/upnp/")))

	httpServer := &http.Server{Handler: mux}
	go func() {
		log := log.WithField("http.listener", httpConn.Addr())
		log.Info("serving HTTP")
		err := httpServer.Serve(httpConn)
		if !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("could not serve HTTP")
		}
	}()

	url := fmt.Sprintf("http://%v/upnp/", httpConn.Addr())
	go func() {
		if err := upnp.BroadcastDevice(ctx, device, url, iface, *notifyInterval); err != nil {
			log.WithError(err).Fatal("could not serve SSDP")
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	upnp.NotifyByeBye(ctx, device, url, iface)
	httpServer.Close()
}
