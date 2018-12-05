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

package example

import (
	"gitlab.com/rapidos/rapidos/internal/pkg/rapidos"
)

func init() {
	manifest := rapidos.Manifest{
		Name:  "lio-local",
		Descr: "LIO iSCSI target",
		Inventory: rapidos.Inventory{
			Init:  "gitlab.com/rapidos/rapidos/inits/lio_local/uinit",
			Pkgs: []string{
				// The following pkgs aren't strictly needed,
				// but provide a nice interactive shell to play
				// with once Init has completed...
				"github.com/u-root/u-root/xcmds/rush",
				"github.com/u-root/u-root/cmds/ls",
				"github.com/u-root/u-root/cmds/pwd",
				"github.com/u-root/u-root/cmds/cat",
				"github.com/u-root/u-root/cmds/dmesg",
				"github.com/u-root/u-root/cmds/df",
				"github.com/u-root/u-root/cmds/echo",
				"github.com/u-root/u-root/cmds/mkdir",
				"github.com/u-root/u-root/cmds/shutdown",
			},
			Kmods: []string{"zram", "lzo", "iscsi_target_mod",
				"target_core_mod", "target_core_iblock"},
			Bins:  []string{},
			Files: []string{},
		},
		VMResources: rapidos.Resources{
			Network: true,
			CPUs:    2,
			Memory:  "2048",
		},
	}

	rapidos.AddManifest(manifest)
}
