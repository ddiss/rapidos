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
		Name:  "example",
		Descr: "Simple annotated example",
		// Builder specifies the u-root builder type. This can be "bb"
		// (single busybox style binary) or "binary" (separate files for
		// each of the listed Pkgs).
		Builder: "bb",
		Inventory: rapidos.Inventory{
			// The Init package will be executed immediately when
			// the image is booted.
			Init:  "gitlab.com/rapidos/rapidos/inits/example/uinit",
			// Additional go packages can be listed in Pkgs
			Pkgs: []string{
				// The following pkgs aren't strictly needed,
				// but provide a nice interactive shell to play
				// with once Init has completed...
				"github.com/u-root/u-root/cmds/exp/rush",
				"github.com/u-root/u-root/cmds/core/ls",
				"github.com/u-root/u-root/cmds/core/pwd",
				"github.com/u-root/u-root/cmds/core/cat",
				"github.com/u-root/u-root/cmds/exp/modprobe",
				"github.com/u-root/u-root/cmds/core/dmesg",
				"github.com/u-root/u-root/cmds/core/mount",
				"github.com/u-root/u-root/cmds/core/df",
				"github.com/u-root/u-root/cmds/core/mkdir",
				"github.com/u-root/u-root/cmds/core/shutdown",
			},
			// Kmods contains a list of kernel modules that should
			// be placed in the initramfs image. modules.dep
			// dependencies will be automatically added.
			Kmods: []string{"zram", "lzo"},
			// Bins specifies which binaries from the host system
			// should be included in the image.
			Bins:  []string{"mkfs.xfs"},
			// Additional miscellaneous files can be listed below.
			Files: []string{},
		},
		// VMResources are passed through to qemu when the image is
		// booted via "rapidos -boot".
		VMResources: rapidos.Resources{
			Network: false,
			CPUs:    2,
			Memory:  "512M",
		},
	}

	// AddManifest() registers this manifest when it is imported by the
	// main application in rapidos.go.
	rapidos.AddManifest(manifest)
}
