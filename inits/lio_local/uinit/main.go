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
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strings"

	"github.com/u-root/u-root/pkg/kmodule"
	"github.com/u-root/u-root/pkg/mount"

	"gitlab.com/rapidos/rapidos/inits/uinit_common"
)

const (
	zramDisksize         = "1G"
	cfsZramBackstorePath = "/sys/kernel/config/target/core/iblock_0/zramo"
	cfsiSCSIPath         = "/sys/kernel/config/target/iscsi"
)

func main() {
	for _, mod := range []string{"target_core_mod", "target_core_iblock",
		"iscsi_target_mod"} {
		err := kmodule.Probe(mod, "")
		if err != nil {
			log.Fatalf("failed to load %s kmod: %v", mod, err)
		}
	}

	zramDev := uinit_common.ProvisionZram(zramDisksize)

	c, err := uinit_common.ReadConfGob()
	if err != nil {
		log.Fatalf("failed to parse conf %v\n", err)
	}
	uinit_common.EnableDynDebug(c)
	targetIQN, _ := uinit_common.GetiSCSIConf(c)

	_, err = mount.Mount("configfs", "/sys/kernel/config/", "configfs", "", 0)
	if err != nil {
		log.Fatalf("mount failed: %v", err)
	}

	err = os.MkdirAll(cfsiSCSIPath, 0755)
	if err != nil {
		log.Fatal(err)
	}
	err = os.MkdirAll("/var/target/pr", 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsiSCSIPath,
		"discovery_auth/enforce_discovery_auth"),
		[]byte("0"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(cfsZramBackstorePath, 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsZramBackstorePath, "control"),
		[]byte("udev_path="+zramDev), 0644)
	if err != nil {
		log.Fatal(err)
	}

	serial := strings.Replace(zramDev, "/", "_", -1)

	err = os.MkdirAll("/var/target/alua/tpgs_"+serial, 0755)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsZramBackstorePath,
		"wwn/vpd_unit_serial"),
		[]byte(serial), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// ignore failure if UNMAP support can't be enabled
	ioutil.WriteFile(path.Join(cfsZramBackstorePath, "attrib/emulate_tpu"),
		[]byte("1"), 0644)

	err = ioutil.WriteFile(path.Join(cfsZramBackstorePath, "enable"),
		[]byte("1"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(path.Join(cfsiSCSIPath, targetIQN,
		"tpgt_0/lun/lun_0"),
		0755)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Symlink(cfsZramBackstorePath,
		path.Join(cfsiSCSIPath, targetIQN,
			"tpgt_0/lun/lun_0/zramo"))
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsiSCSIPath, targetIQN,
		"tpgt_0/attrib/authentication"),
		[]byte("0"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsiSCSIPath, targetIQN,
		"tpgt_0/attrib/demo_mode_write_protect"),
		[]byte("0"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsiSCSIPath, targetIQN,
		"tpgt_0/attrib/generate_node_acls"),
		[]byte("1"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(cfsiSCSIPath, targetIQN,
		"tpgt_0/param/AuthMethod"),
		[]byte("CHAP,None"), 0644)
	if err != nil {
		log.Fatal(err)
	}

	// LIO "demo-mode" dynamically creates acls for connecting initiators
	//	for _, iIQN := range(initiatorIQNs) {
	//		err = os.MkdirAll(path.Join(cfsiSCSIPath, targetIQN,
	//			"tpgt_0/acls/", iIQN), 0755)
	//		if err != nil {
	//			log.Fatal(err)
	//		}
	//	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalf("failed to get network addresses: %v", err)
	}
	ready := false
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue // skip loopback and non-V4 addresses
		}
		ipPort := ipnet.IP.String() + ":3260"
		err = os.MkdirAll(path.Join(cfsiSCSIPath, targetIQN,
			"tpgt_0/np/", ipPort), 0755)
		if err != nil {
			log.Fatal(err)
		}

		err = ioutil.WriteFile(path.Join(cfsiSCSIPath, targetIQN,
			"tpgt_0", "enable"),
			[]byte("1"), 0644)
		log.Printf("target ready at: iscsi://%s/%s/0\n", ipPort,
			targetIQN)
		ready = true
	}
	if !ready {
		log.Fatalf("failed to find any IP addresses to listen on\n")
	}
}
