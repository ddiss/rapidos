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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
)

func getPidPath(pidsDir string, vmIndex int) string {
	// XXX use github.com/rapido-linux compatible pidfile for now, so that
	// images can be booted by vm.sh alongside rapidos -boot.
	file := fmt.Sprintf("rapido_vm%d.pid", vmIndex)
	return path.Join(pidsDir, file)
}

// qemu may put garbage in its pidfile, so read the first line only
func checkQEMUProc(vmPidPath string) (bool, error) {
	file, err := os.Open(vmPidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // not running - no pid file
		}
		return false, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	pidB, isPrefix, err := reader.ReadLine()
	if err != nil || isPrefix {
		err = fmt.Errorf("pidfile read error or overflow")
		return false, err
	}

	pidStr := strings.TrimSpace(string(pidB))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return false, err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	}

	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		// assume ESRCH or "os: process already finished"
		return false, nil // not running
	}

	return true, nil // running - kill(0) succeeds
}

func getQEMURscArgs(conf *RapidosConf, vmResources Resources,
	vmIndex int) ([]string, string, error) {
	var rsc []string

	if vmResources.CPUs == 0 {
		rsc = append(rsc, "-smp", "cpus=2")
	} else {
		nCPUs := strconv.FormatUint(uint64(vmResources.CPUs), 10)
		rsc = append(rsc, "-smp", "cpus="+nCPUs)
	}

	if vmResources.Memory == "" {
		rsc = append(rsc, "-m", "512")
	} else {
		rsc = append(rsc, "-m", vmResources.Memory)
	}

	if !vmResources.Network {
		// all done, no network devices required
		rsc = append(rsc, "-net", "none")
		return rsc, "ip=none", nil
	}

	vmDef, err := conf.GetVMDef(vmIndex)
	if err != nil {
		return nil, "", err
	}

	rsc = append(rsc, "-device",
		"e1000,netdev=nw1,mac="+vmDef.MACAddr, "-netdev",
		"tap,id=nw1,script=no,downscript=no,ifname="+vmDef.TapDev)

	kernIP := "ip="
	if vmDef.UseDHCP {
		kernIP += "dhcp"
	} else {
		kernIP += vmDef.IPAddr + ":::255.255.255.0:" + vmDef.Hostname
	}

	return rsc, kernIP, nil
}

func runQEMU(conf *RapidosConf, imgPath string, resc Resources,
	vmPidPath string, vmIndex int) error {
	qemuBins, err := FindBins([]string{"qemu-kvm", "kvm"},
		true) // ignoreMissing=true
	if len(qemuBins) == 0 || err != nil {
		return fmt.Errorf("failed to find qemu binary")
	}

	kern, err := conf.GetKernImgPath()
	if err != nil {
		return err
	}
	qemuCmd := []string{"-kernel", kern}

	if imgPath != "" {
		qemuCmd = append(qemuCmd, "-initrd", imgPath)
	}

	qemuCmd = append(qemuCmd, "-pidfile", vmPidPath)

	qemuRscArgs, kernIP, err := getQEMURscArgs(conf, resc, vmIndex)
	if err != nil {
		return err
	}
	qemuCmd = append(qemuCmd, qemuRscArgs...)

	qemuCmd = append(qemuCmd, "-append", kernIP+" console=ttyS0")

	qemuExtraArgs, err := conf.GetQEMUExtraArgs()
	if err != nil {
		return err
	}
	qemuCmd = append(qemuCmd, qemuExtraArgs...)

	// TODO support manifest provided supplimental QEMU args

	if conf.Debug {
		fmt.Printf("running: %v\n", qemuCmd)
	}

	cmd := exec.Command(qemuBins[0], qemuCmd...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func Boot(conf *RapidosConf, imgPath string, pidsDir string) error {
	var resc Resources
	var maxVMs int

	// interface here is pretty suboptimal
	err := resc.Retrieve(imgPath)
	if err != nil {
		return err
	}

	if resc.Network {
		// for network enabled VMs we need per-VM MAC/IP configuration
		// and a corresponding tap device. As such we limit to the
		// number of VM network configs supported in rapidos.conf.
		maxVMs = 3
	} else {
		maxVMs = 1000 // no effective limit
	}

	for vmIndex := 1; vmIndex <= maxVMs; vmIndex++ {
		vmPidPath := getPidPath(pidsDir, vmIndex)
		isRunning, err := checkQEMUProc(vmPidPath)
		if err != nil {
			return err
		}
		if isRunning {
			continue
		}
		return runQEMU(conf, imgPath, resc, vmPidPath, vmIndex)
	}

	// we only get here if no VMs were started
	return fmt.Errorf("rapidos.go only supports a maximum of %d VMs", maxVMs)
}
