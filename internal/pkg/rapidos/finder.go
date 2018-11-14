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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/u-root/u-root/pkg/kmodule"
)

func FindKmods(conf *RapidosConf, kmodNames []string) ([]string, error) {
	var absPaths []string

	kmodsInfo, err := conf.GetKmodsInfo()
	if err != nil {
		return nil, err
	}

	opts := kmodule.ProbeOpts{
		RootDir: kmodsInfo.KernelInstModPath,
		KVer:    kmodsInfo.KernelVersion,
		DryRunCB: func(modPath string) {
			var trimmer string
			// Ensure that any leading '/' from dst is trimmed...
			// (see path.Clean doc note about '/')
			if kmodsInfo.KernelInstModPath == "/" {
				trimmer = "/"
			} else {
				trimmer = kmodsInfo.KernelInstModPath + "/"
			}
			// strip local install base path for dstPath
			dst := strings.TrimPrefix(modPath, trimmer)
			// don't bother weeding out dups
			absPaths = append(absPaths, modPath+":"+dst)
		},
	}

	for _, name := range kmodNames {
		err = kmodule.ProbeOptions(name, "", opts)
		if err != nil {
			return nil, err
		}
	}

	// append modules.dep and modules.builtin, needed by modprobe
	for _, modMeta := range []string{"modules.dep", "modules.builtin"} {
		relModMeta := path.Join("lib/modules/", kmodsInfo.KernelVersion,
			modMeta)
		absPaths = append(absPaths,
			path.Join(kmodsInfo.KernelInstModPath,
				relModMeta)+":"+relModMeta)
	}

	return absPaths, nil
}

func FindBins(binNames []string, ignoreMissing bool) ([]string, error) {
	pathOld := os.Getenv("PATH")
	err := os.Setenv("PATH", pathOld+":/usr/sbin:/sbin")
	if err != nil {
		return nil, err
	}
	defer os.Setenv("PATH", pathOld)

	var absPaths []string
	for _, name := range binNames {
		f, err := exec.LookPath(name)
		if err != nil {
			if ignoreMissing {
				continue
			}
			return nil, err
		}
		f, err = filepath.Abs(f)
		if err != nil {
			return nil, err
		}
		absPaths = append(absPaths, f)
	}

	return absPaths, nil
}
