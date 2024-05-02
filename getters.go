package main

import (
	"errors"
	"fmt"
	"math"

	"github.com/dustin/go-humanize"
	"libvirt.org/go/libvirt"
)

type prevStats struct {
	cpuTime uint64
	rdBytes int64
	wrBytes int64
	rxBytes int64
	txBytes int64
}

func getCPUUsage(dom *libvirt.Domain, prevSt *prevStats, numCores uint, sleepTime uint64) (string, error) {
	cpuStats, err := dom.GetCPUStats(-1, 1, 0)
	if err != nil {
		return "", err
	}
	var cpuUsage float64 = 0
	if prevSt.cpuTime == 0 {
		prevSt.cpuTime = cpuStats[0].CpuTime
	} else {
		cpuUsage = float64(cpuStats[0].CpuTime-prevSt.cpuTime) / float64(sleepTime*1000000000) * 100 / float64(numCores)
		if cpuUsage > 100 {
			cpuUsage = 100
		}
		prevSt.cpuTime = cpuStats[0].CpuTime
	}
	return fmt.Sprintf("%.2f%%", cpuUsage), nil
}

func getDiskStats(dom *libvirt.Domain, prevSt *prevStats, sleepTime uint64) (string, error) {
	diskStats, err := dom.BlockStats("")
	if err != nil {
		return "", err
	}

	if prevSt.rdBytes == 0 && prevSt.wrBytes == 0 {
		prevSt.rdBytes = diskStats.RdBytes
		prevSt.wrBytes = diskStats.WrBytes
		return "", nil
	}

	ioReadPerSecond := uint64(diskStats.RdBytes-prevSt.rdBytes) / sleepTime
	ioWritePerSecond := uint64(diskStats.WrBytes-prevSt.wrBytes) / sleepTime
	prevSt.rdBytes = diskStats.RdBytes
	prevSt.wrBytes = diskStats.WrBytes
	return humanize.IBytes(ioReadPerSecond) + " / " + humanize.IBytes(ioWritePerSecond), nil

}

func getMemoryStats(dom *libvirt.Domain) (string, error) {

	memStats, err := dom.MemoryStats(math.MaxUint16, 0)
	if err != nil {
		return "", err
	}

	var totalMemory, unusedMemory uint64
	totalMemorySet, unusedMemorySet := false, false

	for _, stat := range memStats {
		switch stat.Tag {
		case int32(libvirt.DOMAIN_MEMORY_STAT_ACTUAL_BALLOON):
			totalMemory = stat.Val
			totalMemorySet = true
		case int32(libvirt.DOMAIN_MEMORY_STAT_UNUSED):
			unusedMemory = stat.Val
			unusedMemorySet = true
		}
	}

	if !totalMemorySet || !unusedMemorySet {
		return "", errors.New("required memory stats not available")
	}

	usedMemory := totalMemory - unusedMemory
	return humanize.IBytes(usedMemory*1024) + " / " + humanize.IBytes(totalMemory*1024), nil
}

func getNetworkStats(dom *libvirt.Domain, prevSt *prevStats, sleepTime uint64) (string, error) {
	netRxPerSecond := "0"
	netTxPerSecond := "0"
	ifAddr, err := dom.ListAllInterfaceAddresses(0)
	if err != nil {
		return "", err
	}
	var rxBytes int64 = 0
	var txBytes int64 = 0
	for _, iface := range ifAddr {
		netStats, err := dom.InterfaceStats(iface.Name)
		if err != nil {
			return "", err
		}
		rxBytes += netStats.RxBytes
		txBytes += netStats.TxBytes
	}
	if prevSt.rxBytes == 0 && prevSt.txBytes == 0 {
		prevSt.rxBytes = rxBytes
		prevSt.txBytes = txBytes
	} else {
		netRxPerSecond = humanize.IBytes(uint64(rxBytes-prevSt.rxBytes) / sleepTime)
		netTxPerSecond = humanize.IBytes(uint64(txBytes-prevSt.txBytes) / sleepTime)
		prevSt.rxBytes = rxBytes
		prevSt.txBytes = txBytes
	}
	return netRxPerSecond + " / " + netTxPerSecond, nil
}
