// Copyright (C) SUSE LLC 2019, all rights reserved.
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
	"os"
	"os/exec"
	"text/template"

	"github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/kmodule"

	"gitlab.com/rapidos/rapidos/inits/uinit_common"
)

const (
	zramDisksize = "2G"
	conf = `
	[global]

	[{{.Share}}]
		comment = cifsd share
		path = /root
		read only = no
`
)

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

	cmd = exec.Command("chmod", "0777", "/root")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("chmod failed: %v", err)
	}

	err = kmodule.Probe("cifsd", "")
	if err != nil {
		log.Fatalf("failed to load cifsd kmod: %v", err)
	}

	err = os.MkdirAll("/etc/cifs", 0755)
	if err != nil {
		log.Fatal(err)
	}

	cifsdToolsSrc := uinit_common.GetDirPath(c, "CIFSD_TOOLS_SRC")

	pathOld := os.Getenv("PATH")
	// modify PATH so that FindBins looks in the user source dir
	// XXX add auto PATH helper
	err = os.Setenv("PATH",
		pathOld + ":" + cifsdToolsSrc + "/cifsadmin/.libs:" + cifsdToolsSrc + "/cifsd/.libs")
	err = os.Setenv("LD_LIBRARY_PATH", cifsdToolsSrc + "/lib/.libs")

	cifsOpts := uinit_common.GetCifsOpts(c)

	cmd = exec.Command("cifsadmin", "-a", cifsOpts.User, "-p", cifsOpts.Pw)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("cifsadmin failed: %v", err)
	}

	// XXX using templates adds ~1M to init - do it in cut!!
	tmpl, err := template.New("smb.conf").Parse(conf)
	if err != nil {
		log.Fatalf("tmpl failed: %v", err)
	}

	f, err := os.Create("/etc/cifs/smb.conf")
	if err != nil {
		log.Fatalf("failed to write conf %v\n", err)
	}
	err = tmpl.Execute(f, cifsOpts)
	f.Close()
	if err != nil {
		log.Fatalf("failed to write conf %v\n", err)
	}

	cmd = exec.Command("cifsd")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("cifsd failed: %v", err)
	}
	log.Print("cifsd loaded and running\n")
}
