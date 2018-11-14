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
	"log"
	"strconv"
	"strings"
	"syscall"
)

type Resources struct {
	// Whether the VM should be connected to the Rapidos bridge network
	Network bool
	// Number of SMP virtual CPUs to assign to the VM (QEMU uses a single
	// vCPU by default).
	CPUs uint8
	// Amount of memory to assign to the VM (use QEMU default when not set).
	// This value is MiB by default, but can be specified with an explicit
	// M or G suffix
	Memory string
}

// XXX the awkward QEMU-param on-disk format is used to retain compatibility
// with Rapido vm.sh. Transition to gob like (file?) encoding in future.
func packMemCPU(CPUs uint8, Memory string) (string, error) {
	if CPUs < 1 {
		return "", fmt.Errorf("invalid CPUs count %d", CPUs)
	}
	if len(Memory) == 0 {
		return "", fmt.Errorf("invalid memory string %s", Memory)
	}

	xattrVal := fmt.Sprintf("-smp cpus=%d -m %s", CPUs, Memory)
	return xattrVal, nil
}

// parse Number with M/m or G/g suffix
func ValidateMemStr(mem string) error {
	memSuffix := ""
	trailerStripped := strings.TrimRight(mem, "MmGg")
	if trailerStripped != mem {
		memSuffix = strings.TrimPrefix(mem, trailerStripped)
		if len(memSuffix) > 1 {
			return fmt.Errorf("invalid mem resource suffix: %s\n",
				memSuffix)
		}
	}

	_, err := strconv.ParseUint(trailerStripped, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid mem resource: %v\n", err)
	}
	return nil
}

// based on @xattrVal, return CPU and Memory resource values
func unpackMemCPU(xattrVal string) (uint8, string, error) {
	var nCPUs int
	var mem string

	items, err := fmt.Sscanf(xattrVal, "-smp cpus=%d -m %s", &nCPUs, &mem)
	if err != nil {
		log.Printf("resource sscanf failed: %v\n", err)
		return 0, "", err
	}
	if items != 2 {
		err = fmt.Errorf("unexpected resource sscanf item count %d",
			items)
		return 0, "", err
	}

	if nCPUs < 1 || nCPUs > 255 {
		return 0, "", fmt.Errorf("invalid CPU resource %u", nCPUs)
	}
	err = ValidateMemStr(mem)
	if err != nil {
		return 0, "", err
	}

	return uint8(nCPUs), mem, nil
}

// store the Resources state as xattrs on @imgPath
// XXX intentionally use the "user.rapido." namespace, to remain compatible
// with github.com/rapido-linux
func (resc *Resources) Apply(imgPath string) error {
	var err error

	if !resc.Network {
		// Rapido's vm.sh currently assumes network unless the
		// vm_networkless xattr is explicitly provided.
		err = syscall.Setxattr(imgPath, "user.rapido.vm_networkless",
			[]byte("1"), 0)
		if err != nil {
			return err
		}
	}

	xattrVal, err := packMemCPU(resc.CPUs, resc.Memory)
	if err != nil {
		return err
	}
	if len(xattrVal) > 0 {
		err = syscall.Setxattr(imgPath, "user.rapido.vm_resources",
			[]byte(xattrVal), 0)
		if err != nil {
			return err
		}
	}

	return nil
}

// restore Resources state based on @imgPath xattrs
// XXX intentionally use the "user.rapido." namespace, to remain compatible
// with github.com/rapido-linux
func (resc *Resources) Retrieve(imgPath string) error {
	b := make([]byte, 256)

	// defaults to be kept if getxattr returns ENODATA (no xattr)
	resc.Network = true
	resc.CPUs = 2
	resc.Memory = "512M"

	sz, err := syscall.Getxattr(imgPath, "user.rapido.vm_networkless", b)
	if err != nil && err != syscall.ENODATA {
		log.Printf("getxattr failed: %v\n", err)
		return err
	} else if err == nil {
		if sz > len(b) {
			return fmt.Errorf("unexpected vm_networkless xattr val")
		}
		if string(b[:sz]) == "1" {
			resc.Network = false
		}
	}

	sz, err = syscall.Getxattr(imgPath, "user.rapido.vm_resources", b)
	if err != nil && err != syscall.ENODATA {
		log.Printf("getxattr failed: %v\n", err)
		return err
	} else if err == nil {
		if sz > len(b) {
			return fmt.Errorf("unexpected vm_resources xattr val")
		}
		nCPUs, mem, err := unpackMemCPU(string(b[:sz]))
		if err != nil {
			log.Printf("unpackMemCPU failed: %v\n", err)
			return err
		}
		resc.CPUs = nCPUs
		resc.Memory = mem
	}

	return nil
}
