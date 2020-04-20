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
	"os"
	"os/exec"

	"github.com/u-root/u-root/pkg/mount"

	"gitlab.com/rapidos/rapidos/inits/uinit_common"
)

const zramDisksize = "2G"

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

	_, err = mount.Mount(zramDev, "/root", "xfs", "", 0)
	if err != nil {
		log.Fatalf("mount failed: %v", err)
	}

	cmd = exec.Command("minio", "server", "/root")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("minio failed: %v", err)
	}
}
