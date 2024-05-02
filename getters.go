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

// StatProvider provides methods for retrieving various statistics from a libvirt domain.
type StatProvider struct {
	dom    *libvirt.Domain
	prevSt *prevStats
}

// NewStatProvider creates a new instance of StatsProvider with the given libvirt.Domain.
// It initializes the StatsProvider with an empty prevStats struct.
func NewStatProvider(dom *libvirt.Domain) *StatProvider {
	return &StatProvider{
		dom:    dom,
		prevSt: &prevStats{},
	}
}

func (sp *StatProvider) getCPUUsage(sleepTime uint64) (string, error) {
	info, err := sp.dom.GetInfo()
	if err != nil {
		return "", err
	}
	numCores := info.NrVirtCpu

	cpuStats, err := sp.dom.GetCPUStats(-1, 1, 0)
	if err != nil {
		return "", err
	}
	var cpuUsage float64 = 0
	if sp.prevSt.cpuTime == 0 {
		sp.prevSt.cpuTime = cpuStats[0].CpuTime
	} else {
		cpuUsage = float64(cpuStats[0].CpuTime-sp.prevSt.cpuTime) / float64(sleepTime*1000000000) * 100 / float64(numCores)
		if cpuUsage > 100 {
			cpuUsage = 100
		}
		sp.prevSt.cpuTime = cpuStats[0].CpuTime
	}
	return fmt.Sprintf("%.2f%%", cpuUsage), nil
}

func (sp *StatProvider) getDiskStats(sleepTime uint64) (string, error) {
	diskStats, err := sp.dom.BlockStats("")
	if err != nil {
		return "", err
	}

	if sp.prevSt.rdBytes == 0 && sp.prevSt.wrBytes == 0 {
		sp.prevSt.rdBytes = diskStats.RdBytes
		sp.prevSt.wrBytes = diskStats.WrBytes
		return "", nil
	}

	ioReadPerSecond := uint64(diskStats.RdBytes-sp.prevSt.rdBytes) / sleepTime
	ioWritePerSecond := uint64(diskStats.WrBytes-sp.prevSt.wrBytes) / sleepTime
	sp.prevSt.rdBytes = diskStats.RdBytes
	sp.prevSt.wrBytes = diskStats.WrBytes
	return humanize.IBytes(ioReadPerSecond) + " / " + humanize.IBytes(ioWritePerSecond), nil

}

func (sp *StatProvider) getMemoryStats() (string, error) {

	memStats, err := sp.dom.MemoryStats(math.MaxUint16, 0)
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

func (sp *StatProvider) getNetworkStats(sleepTime uint64) (string, error) {
	netRxPerSecond := "0"
	netTxPerSecond := "0"
	ifAddr, err := sp.dom.ListAllInterfaceAddresses(0)
	if err != nil {
		return "", err
	}
	var rxBytes int64 = 0
	var txBytes int64 = 0
	for _, iface := range ifAddr {
		netStats, err := sp.dom.InterfaceStats(iface.Name)
		if err != nil {
			return "", err
		}
		rxBytes += netStats.RxBytes
		txBytes += netStats.TxBytes
	}
	if sp.prevSt.rxBytes == 0 && sp.prevSt.txBytes == 0 {
		sp.prevSt.rxBytes = rxBytes
		sp.prevSt.txBytes = txBytes
	} else {
		netRxPerSecond = humanize.IBytes(uint64(rxBytes-sp.prevSt.rxBytes) / sleepTime)
		netTxPerSecond = humanize.IBytes(uint64(txBytes-sp.prevSt.txBytes) / sleepTime)
		sp.prevSt.rxBytes = rxBytes
		sp.prevSt.txBytes = txBytes
	}
	return netRxPerSecond + " / " + netTxPerSecond, nil
}
