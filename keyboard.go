package main

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"libvirt.org/go/libvirt"
)

type action struct {
	actionFunc    func(d *libvirt.Domain) error
	actionStart   string
	actionSuccess string
	actionFail    string
}

var actions = map[tcell.Key]action{
	tcell.KeyCtrlQ: {(*libvirt.Domain).Create, "Starting", "Started", "Failed to start"},
	tcell.KeyCtrlA: {(*libvirt.Domain).Shutdown, "Stopping", "Stopped", "Failed to stop"},
	tcell.KeyCtrlW: {(*libvirt.Domain).Resume, "Resuming", "Resumed", "Failed to resume"},
	tcell.KeyCtrlS: {(*libvirt.Domain).Suspend, "Suspending", "Suspended", "Failed to suspend"},
	tcell.KeyCtrlE: {func(d *libvirt.Domain) error { return d.Reboot(0) }, "Rebooting", "Rebooted", "Failed to reboot"},
	tcell.KeyCtrlD: {(*libvirt.Domain).Destroy, "Destroying", "Destroyed", "Failed to destroy"},
	tcell.KeyCtrlR: {attachSpecificDisk, "Attaching disk to", "Attached disk to", "Failed to attach disk to"},
	tcell.KeyCtrlF: {detachSpecificDisk, "Detaching disk from", "Detached disk from", "Failed to detach disk from"},
}

func libvirtError(err error) string {
	if libvirtErr, ok := err.(libvirt.Error); ok {
		return fmt.Sprintf("Libvirt err %d: %s", libvirtErr.Code, libvirtErr.Message)
	}
	return err.Error()
}

func handleKeypress(conn *libvirt.Connect, table *tview.Table, event *tcell.EventKey) *tcell.EventKey {
	row, _ := table.GetSelection()
	vmName := table.GetCell(row, 0).Text
	if len(vmName) < 2 {
		return event
	}
	vmName = vmName[1 : len(vmName)-1]
	dom, err := conn.LookupDomainByName(vmName)
	if err != nil {
		log.Println("Failed to get domain:", err)
		return event
	}
	defer dom.Free()

	if action, ok := actions[event.Key()]; ok {
		setStatus(action.actionStart + " " + vmName)
		if err := action.actionFunc(dom); err != nil {
			log.Println(action.actionFail+" domain:", err)
			setStatus(action.actionFail + " " + vmName + ". " + libvirtError(err))
		} else {
			log.Println("Domain " + action.actionSuccess + " successfully")
		}
	}
	return event

}

func attachSpecificDisk(dom *libvirt.Domain) error {
	diskXML := `
	<disk type='file' device='disk'>
		<driver name='qemu' type='qcow2'/>
		<source file='/media/radim/SSD/qcow2_drive2.img'/>
		<target dev='vdb' bus='virtio'/>
	</disk>
	`
	// Attach the disk to the domain
	if err := dom.AttachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to hotplug disk:", err)
		return err
	}
	log.Println("Disk hotplugged successfully")
	return nil
}

func detachSpecificDisk(dom *libvirt.Domain) error {
	diskXML := `
	<disk type='file' device='disk'>
		<driver name='qemu' type='qcow2'/>
		<source file='/media/radim/SSD/qcow2_drive2.img'/>
		<target dev='vdb' bus='virtio'/>
	</disk>
	`
	// Detach the disk from the domain
	if err := dom.DetachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to detach disk:", err)
		return err
	}
	log.Println("Disk detached successfully")
	return nil
}
