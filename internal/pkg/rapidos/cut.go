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
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/u-root/u-root/pkg/cpio"
	"github.com/u-root/u-root/pkg/golang"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/uroot/builder"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
)

func Cut(conf *RapidosConf, m *Manifest, rdir string,
	imgPath string) error {
	var files []string
	var err error
	logger := log.New(os.Stderr, "", log.LstdFlags)

	if len(m.Kmods) > 0 {
		files, err = FindKmods(conf, m.Kmods)
		if err != nil {
			return err
		}
	}

	if len(m.Bins) > 0 {
		bins, err := FindBins(m.Bins,
			false) // ignoreMissing=false
		if err != nil {
			return err
		}
		files = append(files, bins...)
	}

	if len(m.Files) > 0 {
		files = append(files, m.Files...)
	}

	// u-root's base "init" is responsible for invoking the manifest
	// specific "uinit", and subsequently interactive shell (rush)
	pkgs := append(m.Pkgs, "github.com/u-root/u-root/cmds/init", m.Init)

	env := golang.Default()
	env.CgoEnabled = false

	var b builder.Builder
	switch m.Builder {
	case "bb":
		b = builder.BBBuilder{}
	case "binary":
		b = builder.BinaryBuilder{}
	default:
		return fmt.Errorf("unsupported builder type %s", m.Builder)
	}

	archiver, err := initramfs.GetArchiver("cpio")
	if err != nil {
		return err
	}

	tmpDir, err := ioutil.TempDir("", "rapidos")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// XXX write a subset of conf based on manifest?
	confGob, err := conf.GenGob()
	if err != nil {
		return err
	}

	// XXX OpenWriter truncates to zero, but the resource xattrs need to be
	// dropped, so delete unconditionally
	err = os.Remove(imgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	w, err := archiver.OpenWriter(logger, imgPath, "", "")
	if err != nil {
		return err
	}

	// similar to the existing default, but drops resolv.conf, etc.
	base := cpio.ArchiveFromRecords([]cpio.Record{
		cpio.Directory("etc", 0755),
		cpio.Directory("dev", 0755),
		cpio.Directory("tmp", 0777),
		cpio.Directory("ubin", 0755),
		cpio.Directory("usr", 0755),
		cpio.Directory("usr/lib", 0755),
		cpio.Directory("var/log", 0777),
		cpio.Directory("lib64", 0755),
		cpio.Directory("bin", 0755),
		cpio.CharDev("dev/console", 0600, 5, 1),
		cpio.CharDev("dev/tty", 0666, 5, 0),
		cpio.CharDev("dev/null", 0666, 1, 3),
		cpio.CharDev("dev/port", 0640, 1, 4),
		cpio.CharDev("dev/urandom", 0666, 1, 9),

		cpio.StaticFile("rapidos.conf.bin", confGob.String(), 0600),
	})

	opts := uroot.Opts{
		TempDir: tmpDir,
		Env:     env,
		Commands: []uroot.Commands{
			{
				Builder:  b,
				Packages: pkgs,
			},
		},
		ExtraFiles: files,
		OutputFile: w,
		// TODO: use a manifest specific initcmd, rather than relying
		// on the init->uinit functionality?
		InitCmd:      "init",
		DefaultShell: "/bbin/rush",
		BaseArchive:  base.Reader(),
	}

	if conf.Debug {
		log.Printf("uroot opts: %+v\n", opts)
	}

	err = uroot.CreateInitramfs(logger, opts)
	if err != nil {
		return err
	}

	err = m.VMResources.Apply(imgPath)
	if err != nil {
		log.Fatalf("failed to apply VM resources: %v", err)
	}

	return nil
}
