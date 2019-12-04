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
		Name:  "etcd",
		Descr: "Distributed key-value store",
		// Use u-root binary builder so that pkgs with vendor subdirs
		// are handled correctly.
		Builder: "binary",
		Inventory: rapidos.Inventory{
			Init:  "gitlab.com/rapidos/rapidos/inits/etcd/uinit",
			Pkgs: []string{
				"go.etcd.io/etcd",
			},
			Kmods: []string{"zram", "lzo", "lzo-rle"},
			Bins:  []string{"mkfs.xfs"},
			Files: []string{},
		},
		VMResources: rapidos.Resources{
			Network: true,
			CPUs:    2,
			Memory:  "1024M",
		},
	}

	rapidos.AddManifest(manifest)
}
