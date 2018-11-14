// Copyright (C) SUSE LINUX GmbH 2018, all rights reserved.
//
// This program is free software: you can redistribute it and/or modify it under
// the terms of the GNU General Public License as published by the Free Software
// Foundation, version 3.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE. See the GNU General Public License for more
// details.

package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/u-root/u-root/pkg/mount"

	"gitlab.com/rapidos/rapidos/inits/uinit_common"
)

const zramDisksize = "2G"

type State int

const (
	unknown State = iota
	etcdStarting
	etcdStarted
	etcdExited
)

func startEtcd(ch chan<- State, dataDir string) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	etcdArgs := []string{"--name", hostname, "--data-dir", "/root",
		"--initial-cluster-token", "rapidos-etcd-cluster",
		"--initial-cluster-state", "new"}
	// TODO support CA and --initial-cluster node list configuration in
	// rapidos.conf

	// Add client + peer listen URLs for each non-loopback addr
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("failed to get network addresses: %v", err)
	}
	cliAddr := ""
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		// FIXME support https and IPv6!
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue // skip loopback and non-V4 addresses
		}

		etcdArgs = append(etcdArgs, "--listen-peer-urls",
			"http://"+ipnet.IP.String()+":2380",
			"--initial-advertise-peer-urls",
			"http://"+ipnet.IP.String()+":2380")

		cliAddr = ipnet.IP.String() + ":2379"
		etcdArgs = append(etcdArgs,
			"--listen-client-urls", "http://"+cliAddr,
			"--advertise-client-urls", "http://"+cliAddr)
	}
	if cliAddr == "" {
		log.Fatalf("no non-loopback v4 addresses present\n")
	}

	cmd := exec.Command("etcd", etcdArgs...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Start()
	if err != nil {
		log.Fatalf("etcd failed to start: %v", err)
	}

	// wait for etcd to start listening. timeout handled by caller
	for {
		conn, _ := net.Dial("tcp", cliAddr)
		if conn != nil {
			conn.Close()
			break
		}
		time.Sleep(time.Duration(100 * time.Millisecond))
	}
	ch <- etcdStarted

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("etcd failed: %v", err)
	}

	ch <- etcdExited
}

func main() {
	zramDev := uinit_common.ProvisionZram(zramDisksize)

	c, err := uinit_common.ReadConfGob()
	if err != nil {
		log.Fatalf("failed to parse conf %v\n", err)
	}
	uinit_common.EnableDynDebug(c)

	cmd := exec.Command("mkfs.xfs", zramDev)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("mkfs failed: %v", err)
	}

	err = mount.Mount(zramDev, "/root", "xfs", "", 0)
	if err != nil {
		log.Fatalf("mount failed: %v", err)
	}

	// start etcd in the background and track its state via channel
	etcdChan := make(chan State)
	go func() {
		startEtcd(etcdChan, "/root")
	}()

	var state State = etcdStarting
	stateTransTimeout := time.Duration(10 * time.Second)
	stateTransTimer := time.NewTimer(stateTransTimeout)
	for {
		select {
		case state = <-etcdChan:
			log.Printf("etcd service state: %v\n", state)
			if state == etcdStarted {
				if !stateTransTimer.Stop() {
					<-stateTransTimer.C
				}
				// stateTransTimer.Reset(stateTransTimeout)
				// XXX subsequent services depending on etcd
				// could be started here.
			} else {
				log.Fatalf("unexpected etcd state: %v",
					state)
			}
		case <-stateTransTimer.C:
			log.Fatalf("State %v transition timeout!", state)
		}
	}
}
