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
	"path/filepath"
	"syscall"
	"time"

	"github.com/ethulhu/helix/flag"
	"github.com/ethulhu/helix/flags"
	"github.com/ethulhu/helix/logger"
	"github.com/ethulhu/helix/media"
	"github.com/ethulhu/helix/netutil"
	"github.com/ethulhu/helix/upnp"
	"github.com/ethulhu/helix/upnpav"
	"github.com/ethulhu/helix/upnpav/connectionmanager"
	"github.com/ethulhu/helix/upnpav/contentdirectory"
	"github.com/ethulhu/helix/upnpav/contentdirectory/fileserver"
	"github.com/fsnotify/fsnotify"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	udn            = flag.Custom("udn", "", "UDN to broadcast (if unset, will generate one)", flags.UDN)
	friendlyName   = flag.Custom("friendly-name", "", "human-readable name to broadcast (if unset, will generate one)", flags.FriendlyName)
	iface          = flag.Custom("interface", "", "interface to listen on (will try to find a Private IPv4 if unset)", flags.NetInterface)
	notifyInterval = flag.Duration("notify-interval", time.Duration(30*time.Second), "interval between SSDP advertisements")
	port           = flag.Int("port", 8080, "port to listen on")

	logLevel             = flag.String("log-level", "info", "log level (debug, info, warning, error)")
	basePath             = flag.Custom("path", "", "path to serve", flag.RequiredString)
	disableMetadataCache = flag.Bool("disable-metadata-cache", false, "disable the metadata cache")
	_                    = flag.Bool("version", false, "show version and exit")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "helix %s (commit %s, built at %s)\n\n", version, commit, date)
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.CommandLine.PrintDefaults()
	}

	for _, arg := range os.Args[1:] {
		if arg == "-version" || arg == "--version" {
			fmt.Printf("helix %s (commit %s, built at %s)\n", version, commit, date)
			os.Exit(0)
		}
	}

	flag.Parse()

	basePath := (*basePath).(string)
	friendlyName := (*friendlyName).(string)
	iface := (*iface).(*net.Interface)
	udn := (*udn).(string)

	ctx := context.Background()
	log, _ := logger.FromContext(ctx)

	if err := logger.SetLevel(*logLevel); err != nil {
		log.WithError(err).Fatal("could not set log level")
	}

	log.WithField("version", version).WithField("commit", commit).WithField("date", date).Info("starting helix")

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
		IP:   ip,
		Port: *port,
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
		ModelDescription: "Helix",
		ModelName:        "Helix",
		ModelNumber:      "42",
		ModelURL:         "https://ethulhu.co.uk",
		SerialNumber:     "00000000",
	}
	device.SetBootID(uint(time.Now().Unix()))

	metadataCache := media.NewMetadataCache()
	if *disableMetadataCache {
		metadataCache = media.NoOpCache{}
	}

	cd, err := fileserver.NewContentDirectory(basePath, fmt.Sprintf("http://%v/objects/", httpConn.Addr()), metadataCache)
	if err != nil {
		log.WithError(err).Fatal("could not create ContentDirectory object")
	}

	device.Handle(contentdirectory.Version1, contentdirectory.ServiceID, contentdirectory.SCPD, contentdirectory.SOAPHandler{Interface: cd})
	device.Handle(connectionmanager.Version1, connectionmanager.ServiceID, connectionmanager.SCPD, nil)

	mux := http.NewServeMux()

	objectsHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("getContentFeatures.dlna.org") != "" {
			w.Header().Set("contentFeatures.dlna.org", upnpav.ContentFeatures)
		}
		log.AddField("http.Method", r.Method)
		log.Debug("serving objects at " + r.URL.String())
		http.StripPrefix("/objects/", http.FileServer(http.Dir(basePath))).ServeHTTP(w, r)
	}

	mux.HandleFunc("/objects/", objectsHandler)
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

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Fatal("could not create filesystem watcher")
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.WithField("event", event.String()).Debug("got filesystem event")
				cd.IncrementSystemUpdateID()
				if err := upnp.SendUpdateNotification(ctx, device, url, iface); err != nil {
					log.WithError(err).Warning("could not send update notification")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.WithError(err).Warning("filesystem watcher error")
			}
		}
	}()

	if err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		log.WithError(err).Fatal("could not watch path")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	upnp.NotifyByeBye(ctx, device, url, iface)
	httpServer.Close()
}
