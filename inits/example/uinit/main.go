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
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/u-root/u-root/pkg/mount"

	"gitlab.com/rapidos/rapidos/inits/uinit_common"
)

const zramDisksize = "100M"

func main() {
	// kernel modules should be loaded prior to EnableDynDebug()
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

	fmt.Printf("\nRapidos scratch VM running. Have a lot of fun...\n")
	// init will exec u-root DefaultShell following uinit completion...
}
