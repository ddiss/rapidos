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
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	// embedded vendor repo
	"github.com/go-ini/ini"
)

type RapidosConf struct {
	// rapidos.conf key=val map
	f map[string]string

	// command line
	Debug bool
}

// basic support for conf variable expansion
func parseConfExpand(f map[string]string) error {
	for key, val := range f {
		// fast path if no variable to expand
		if !strings.Contains(val, "$") {
			continue
		}
		for rkey, rval := range f {
			val = strings.Replace(val, "${"+rkey+"}", rval, -1)
			val = strings.Replace(val, "$"+rkey, rval, -1)
		}
		// No support for recursive replacement or missing vars
		if strings.Contains(val, "$") {
			return fmt.Errorf("%s=%s is recursive or missing",
				key, val)
		}
		f[key] = val
	}
	return nil
}

// Parse confPath as an ini/shell env and return the resulting RapidosConf struct
func ParseConf(confPath string, overlay map[string]string,
	debug bool) (*RapidosConf, error) {
	// TODO set RapidosConfFile defaults
	conf := new(RapidosConf)

	cfg, err := ini.Load(confPath)
	if err != nil {
		return nil, err
	}

	// "in cases that you are very sure about only reading data through the
	// library, you can set cfg.BlockMode = false to speed up read
	// operations"
	cfg.BlockMode = false

	// the private config map is validated on demand, as component specific
	// parameters are requested via the accessor functions
	conf.f = cfg.Section(ini.DEFAULT_SECTION).KeysHash()

	// ideally NameMapper / ValueMapper could handle this in a single pass
	err = parseConfExpand(conf.f)
	if err != nil {
		return nil, err
	}

	for k, v := range overlay {
		conf.f[k] = v
	}

	conf.Debug = debug
	if conf.Debug {
		conf.DumpConf()
	}

	return conf, nil
}

func (conf *RapidosConf) DumpConf() {
	log.Printf("%+v\n", *conf)
}

func checkDirVal(f map[string]string, key string) (string, error) {
	val := f[key]
	if len(val) == 0 {
		return "", fmt.Errorf("%s not configured", key)
	}
	stat, err := os.Stat(val)
	if err != nil {
		return "", err
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("%s is not a directory", val)
	}

	return path.Clean(val), nil
}

type KmodsInfo struct {
	KernelInstModPath string
	KernelVersion     string
}

func (conf *RapidosConf) getRunningKmodsInfo() (*KmodsInfo, error) {
	if conf.f["KERNEL_INSTALL_MOD_PATH"] != "" {
		return nil, fmt.Errorf("error: KERNEL_INSTALL_MOD_PATH set" +
			" without corresponding KERNEL_SRC")
	}

	procVers, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return nil, err
	}
	// Linux version 4.4.159-73-default ...
	procVersSplit := strings.Fields(string(procVers))
	if len(procVersSplit) < 3 {
		return nil, fmt.Errorf("unexpected /proc/version format")
	}

	return &KmodsInfo{"/", procVersSplit[2]}, nil
}

func (conf *RapidosConf) getSourceKmodsInfo() (*KmodsInfo, error) {
	kernelSrc, err := checkDirVal(conf.f, "KERNEL_SRC")
	if err != nil {
		return nil, err
	}

	kverPath := path.Join(kernelSrc, "include/config/kernel.release")
	kver, err := ioutil.ReadFile(kverPath)
	if err != nil {
		return nil, err
	}
	kernelVersion := strings.TrimSpace(string(kver))

	kernelInstModPath, err := checkDirVal(conf.f, "KERNEL_INSTALL_MOD_PATH")
	if err != nil {
		return nil, err
	}

	return &KmodsInfo{kernelInstModPath, kernelVersion}, nil
}

func (conf *RapidosConf) GetKmodsInfo() (*KmodsInfo, error) {
	if conf.f["KERNEL_SRC"] == "" {
		log.Printf("KERNEL_SRC not configured, using running kernel\n")
		return conf.getRunningKmodsInfo()
	}

	return conf.getSourceKmodsInfo()
}

func (conf *RapidosConf) getRunningKernImgPath() (string, error) {
	kver, err := conf.getRunningKmodsInfo()
	if err != nil {
		return "", err
	}
	return "/boot/vmlinuz-" + kver.KernelVersion, nil
}

func (conf *RapidosConf) getSourceKernImgPath() (string, error) {
	kernelSrc, err := checkDirVal(conf.f, "KERNEL_SRC")
	if err != nil {
		return "", err
	}
	// TODO an x86-64 bzImage is currently assumed
	kernImg := path.Join(kernelSrc, "/arch/x86/boot/bzImage")

	return kernImg, nil
}

func (conf *RapidosConf) GetKernImgPath() (string, error) {
	var kernImg string
	var err error
	if conf.f["KERNEL_SRC"] == "" {
		log.Printf("KERNEL_SRC not configured, using running kernel\n")
		kernImg, err = conf.getRunningKernImgPath()
	} else {
		kernImg, err = conf.getSourceKernImgPath()
	}
	if err != nil {
		return "", err
	}

	stat, err := os.Stat(kernImg)
	if err != nil {
		return "", err
	}
	if !stat.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", kernImg)
	}

	return kernImg, nil
}

func (conf *RapidosConf) GetQEMUExtraArgs() ([]string, error) {
	return strings.Split(conf.f["QEMU_EXTRA_ARGS"], " "), nil
}

type RapidosConfVM struct {
	TapDev  string
	MACAddr string
	UseDHCP bool
	// following could be empty if UseDHCP is true
	IPAddr   string
	Hostname string
}

func (conf *RapidosConf) GetVMDef(vmIndex int) (*RapidosConfVM, error) {
	var vmDef RapidosConfVM

	if vmIndex < 1 {
		return nil, fmt.Errorf("invalid vmIndex %d", vmIndex)
	}

	// XXX config syntax is particularly horrid; TAP_DEV uses (vmIndex - 1)
	// while everything else uses vmIndex!
	// TODO: syntax should be fixed in future by moving to ini style
	// hierarchical configuration with one ini section per VM.
	confKey := "TAP_DEV" + strconv.Itoa(vmIndex-1)
	vmDef.TapDev = conf.f[confKey]

	// could use net.InterfaceByName() for validation, but leave it up to
	// qemu for now.
	if vmDef.TapDev == "" {
		return nil, fmt.Errorf("rapidos.conf missing %s\n", confKey)
	}

	confKey = "MAC_ADDR" + strconv.Itoa(vmIndex)
	vmDef.MACAddr = conf.f[confKey]
	if vmDef.MACAddr == "" {
		return nil, fmt.Errorf("rapidos.conf missing %s\n", confKey)
	}

	confKey = "IP_ADDR" + strconv.Itoa(vmIndex) + "_DHCP"
	if conf.f[confKey] == "" || conf.f[confKey] == "0" {
		vmDef.UseDHCP = false
	} else if conf.f[confKey] == "1" {
		vmDef.UseDHCP = true
	} else {
		return nil, fmt.Errorf("rapidos.conf %s invalid value: %s\n",
			confKey, conf.f[confKey])
	}

	confKey = "IP_ADDR" + strconv.Itoa(vmIndex)
	vmDef.IPAddr = conf.f[confKey]
	if vmDef.IPAddr == "" && !vmDef.UseDHCP {
		return nil, fmt.Errorf("rapidos.conf missing %s\n", confKey)
	}

	confKey = "HOSTNAME" + strconv.Itoa(vmIndex)
	vmDef.Hostname = conf.f[confKey]
	if vmDef.Hostname == "" && !vmDef.UseDHCP {
		return nil, fmt.Errorf("rapidos.conf missing %s\n", confKey)
	}

	return &vmDef, nil
}

func (conf *RapidosConf) GenGob() (*bytes.Buffer, error) {
	var b bytes.Buffer
	e := gob.NewEncoder(&b)

	err := e.Encode(conf.f)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
