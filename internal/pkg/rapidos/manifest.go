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

package rapidos

import (
	"log"
)

type Manifest struct {
	// Name of init back-end
	Name string
	// short description of what this image does
	Descr string
	// init package runs immediately following boot..
	// The stock u-root init process is responsible for invoking the "uinit"
	// provided with a given manifest
	Init string
	// Additional go packages to install in the initramfs
	Pkgs []string
	// kernel modules required by this init
	Kmods []string
	// Binaries to locate via PATH (and sbin). The initramfs destination
	// will match the local source, with ldd dependencies also pulled in.
	Bins []string
	// files to include in the initramfs image. ldd dependencies will be
	// automatically pulled in alongside binaries.
	// Files will be placed in the same path as the local source by default.
	// The initramfs destination path can be explicitly specified via:
	// <local source>:<initramfs dest>
	// TODO use u-root/cmds/which to locate bins under PATH (+sbin)
	Files []string
	// u-root builder type. "bb" (default), "binary" or "source"
	Builder string

	// VMResources are different from the rest of the Manifest in that they are
	// considered at VM boot time.
	VMResources Resources
}

var (
	manifs = make(map[string]Manifest)
)

// XXX this is called by manifest init() functions, so should panic on error.
func AddManifest(m Manifest) {
	if len(m.Name) == 0 {
		log.Fatal("invalid manifest with empty name")
	}

	name := m.Name
	if m.VMResources.CPUs < 1 {
		log.Fatalf("%s: invalid manifest CPU resource (%d)",
			name, m.VMResources.CPUs)
	}

	if m.Builder == "" {
		// u-root busybox style build by default
		m.Builder = "bb"
	}

	err := ValidateMemStr(m.VMResources.Memory)
	if err != nil {
		log.Fatalf("%s: invalid manifest memory resource (%s): %v",
			name, m.VMResources.Memory, err)
	}

	if _, ok := manifs[name]; ok {
		log.Fatalf("%s: manifest already present", name)
	}
	manifs[name] = m
}

func LookupManifest(name string) *Manifest {
	if m, ok := manifs[name]; ok {
		return &m
	}
	return nil
}

func IterateManifests(cb func(m Manifest)) {
	for _, m := range manifs {
		cb(m)
	}
}
