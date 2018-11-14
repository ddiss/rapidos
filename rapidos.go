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
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/rapidos/rapidos/internal/pkg/rapidos"

	// inits registered via AddManifest() callback
	_ "gitlab.com/rapidos/rapidos/inits/etcd"
	_ "gitlab.com/rapidos/rapidos/inits/example"
	_ "gitlab.com/rapidos/rapidos/inits/lio_local"
	_ "gitlab.com/rapidos/rapidos/inits/minio"
)

func listInits(title string) {
	fmt.Print(title)
	cb := func(m rapidos.Manifest) {
		fmt.Printf("  %s\n\t%s\n", m.Name, m.Descr)
	}
	rapidos.IterateManifests(cb)
}

func usage() {
	fmt.Printf("Usage: %s [options]\n", filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	listInits("Available inits:\n")
}

type cliParams struct {
	list        bool
	debug       bool
	confPath    string
	imgPath     string
	confOverlay map[string]string
	bootVM      bool
	cutInitName string
	qemuPidDir  string
}

// string "get" callback for -C <key>=<val>. Not sure what to return.
func (params *cliParams) String() string {
	return ""
}

// "set" callback for -C <key>=<val>
func (params *cliParams) Set(value string) error {
	kv := strings.Split(value, "=")
	if len(kv) != 2 {
		return fmt.Errorf("not in <KEY>=<val> format\n", value)
	}
	params.confOverlay[kv[0]] = kv[1]
	return nil
}

func main() {
	// XXX: binary is under /tmp/go-build when run via "go run"!
	rdir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalf("failed to determine rapidos dir: %v", err)
	}

	params := new(cliParams)
	flag.Usage = usage
	flag.BoolVar(&params.list, "list", false, "List available inits")
	flag.BoolVar(&params.debug, "debug", false, "Log debug messages")
	flag.StringVar(&params.confPath, "conf",
		path.Join(rdir, "rapidos.conf"),
		"rapidos.conf config file `path`")
	params.confOverlay = make(map[string]string)
	flag.StringVar(&params.imgPath, "img",
		path.Join(rdir, "imgs", "rapidos-img.cpio"),
		"Initramfs image `path`")
	flag.Var(params, "C",
		"<KEY>=<val> overlay for rapidos.conf. Can be given multiple times")
	flag.StringVar(&params.cutInitName, "cut", "",
		"Cut an image with the provided `init`")
	flag.BoolVar(&params.bootVM, "boot", true, "Boot the initramfs image")
	flag.StringVar(&params.qemuPidDir, "pid-dir",
		path.Join(rdir, "imgs"),
		"Directory `path` for QEMU PID files")

	flag.Parse()

	if len(flag.Args()) != 0 {
		fmt.Printf("Error: unsupported trailing parameter(s)\n")
		usage()
		return
	}

	if params.list {
		listInits("Available inits:\n")
		return
	}

	if params.cutInitName == "" {
		if !params.bootVM {
			fmt.Printf("-cut <img>, -boot, or -list parameter required\n")
			usage()
			return
		}
		_, err = os.Stat(params.imgPath)
		if err != nil && os.IsNotExist(err) {
			fmt.Printf("-cut required: img path %s doesn't exist\n",
				params.imgPath)
			usage()
			return
		}
	}

	conf, err := rapidos.ParseConf(params.confPath, params.confOverlay,
		params.debug)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	if params.cutInitName != "" {
		m := rapidos.LookupManifest(params.cutInitName)
		if m == nil {
			fmt.Printf("Failed to lookup manifest: %s\n",
				params.cutInitName)
			usage()
			return
		}

		err = rapidos.Cut(conf, m, rdir, params.imgPath)
		if err != nil {
			log.Fatalf("failed cut image: %v", err)
		}
	}

	if params.bootVM {
		// QEMU blocks in boot() until shutdown, unless run with -daemonize
		err = rapidos.Boot(conf, params.imgPath, params.qemuPidDir)
		if err != nil {
			log.Fatalf("failed to boot VM: %v", err)
		}
	}
}
