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

package uinit_common

import (
	"bufio"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/u-root/u-root/pkg/kmodule"
	"github.com/u-root/u-root/pkg/mount"
)

const (
	confGobPath = "/rapidos.conf.bin"
)

type RapidosConfMap struct {
	// rapidos.conf key=val map
	f map[string]string
}

func ReadConfGob() (*RapidosConfMap, error) {
	var conf = new(RapidosConfMap)

	file, err := os.Open(confGobPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	d := gob.NewDecoder(reader)
	err = d.Decode(&conf.f)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// enable kernel dynamic debug if configured in rapidos.conf
// should be called after loading all kernel modules
func EnableDynDebug(conf *RapidosConfMap) {
	const dynDebugCtrlPath = "/sys/kernel/debug/dynamic_debug/control"

	_, err := os.Stat(dynDebugCtrlPath)
	if os.IsNotExist(err) {
		err = mount.Mount("debugfs", "/sys/kernel/debug/", "debugfs", "", 0)
		if err != nil {
			log.Fatalf("mount failed: %v", err)
		}
	}

	for _, mod := range strings.Fields(conf.f["DYN_DEBUG_MODULES"]) {
		err = ioutil.WriteFile(dynDebugCtrlPath,
			[]byte("module "+mod+" +pf"), 0644)
		if err != nil {
			log.Fatalf("failed to enable dynamic debug: %v", err)
		}
	}
	for _, f := range strings.Fields(conf.f["DYN_DEBUG_FILES"]) {
		err = ioutil.WriteFile(dynDebugCtrlPath,
			[]byte("file "+f+" +pf"), 0644)
		if err != nil {
			log.Fatalf("failed to enable dynamic debug: %v", err)
		}
	}
}

func ProvisionZram(disksize string) string {
	err := kmodule.Probe("lzo", "")
	if err != nil {
		log.Fatalf("failed to load lzo kmod: %v", err)
	}
	err = kmodule.Probe("lzo-rle", "")
	if err != nil {
		log.Fatalf("failed to load lzo-rle kmod: %v", err)
	}
	err = kmodule.Probe("zram", "num_devices=0")
	if err != nil {
		log.Fatalf("failed to load zram kmod: %v", err)
	}
	zIdx, err := ioutil.ReadFile("/sys/class/zram-control/hot_add")
	if err != nil {
		log.Fatalf("failed to hot-add zram device: %v", err)
	}
	zramName := "zram" + strings.TrimSpace(string(zIdx))
	err = ioutil.WriteFile("/sys/block/"+zramName+"/disksize",
		[]byte(disksize), 0644)
	if err != nil {
		log.Fatalf("failed to write %s disksize: %v", zramName, err)
	}

	return "/dev/" + zramName
}

func GetiSCSIConf(conf *RapidosConfMap) (string, []string) {
	if conf.f["TARGET_IQN"] == "" {
		log.Fatalf("error: TARGET_IQN missing in rapidos.conf\n")
	}

	// missing INITIATOR_IQNS config is not an error

	return conf.f["TARGET_IQN"], strings.Fields(conf.f["INITIATOR_IQNS"])
}
