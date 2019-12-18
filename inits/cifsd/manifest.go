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

package example

import (
	"os"

	"gitlab.com/rapidos/rapidos/internal/pkg/rapidos"
)

func init() {
	manifest := rapidos.Manifest{
		Name:  "cifsd",
		Descr: "In-kernel SMB server",
		Builder: "bb",
		Inventory: rapidos.Inventory{
			Init:  "gitlab.com/rapidos/rapidos/inits/cifsd/uinit",
			Pkgs: []string{
				// The following pkgs aren't strictly needed,
				// but provide a nice interactive shell to play
				// with once Init has completed...
				"github.com/u-root/u-root/cmds/exp/rush",
				"github.com/u-root/u-root/cmds/core/chmod",
				"github.com/u-root/u-root/cmds/core/ls",
				"github.com/u-root/u-root/cmds/core/strace",
				"github.com/u-root/u-root/cmds/core/pwd",
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/exp/modprobe",
				"github.com/u-root/u-root/cmds/core/dmesg",
				"github.com/u-root/u-root/cmds/core/mount",
				"github.com/u-root/u-root/cmds/core/df",
				"github.com/u-root/u-root/cmds/core/mkdir",
				"github.com/u-root/u-root/cmds/core/shutdown",
				"github.com/u-root/u-root/cmds/core/ps",
				"github.com/u-root/u-root/cmds/core/ip",
			},
			Kmods: []string{"zram", "lzo", "lzo-rle", "cifsd"},
			Bins:  []string{"mkfs.xfs", "cifsd", "cifsadmin"},
			Files: []string{},
		},

		// use a callback for processing CIFSD_TOOLS_SRC
		InventoryCB: InventoryCB,

		VMResources: rapidos.Resources{
			Network: true,
			CPUs:    2,
			Memory:  "1024M",
		},
	}

	rapidos.AddManifest(manifest)
}

func InventoryCB(conf rapidos.RapidosConf, inv *rapidos.Inventory) error {
	cifsdToolsSrc, err := conf.GetDirPath("CIFSD_TOOLS_SRC")
	if err != nil {
		return err
	}

	// modify PATH so that FindBins looks in the user source dir
	err = os.Setenv("LD_LIBRARY_PATH", cifsdToolsSrc + "/lib/.libs")
	pathOld := os.Getenv("PATH")
	err = os.Setenv("PATH",
		pathOld + ":" + cifsdToolsSrc + "/cifsadmin/.libs:" +
		cifsdToolsSrc + "/cifsd/.libs")
	return err
}
