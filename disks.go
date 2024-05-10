package main

import (
	"fmt"
	"log"

	"github.com/rivo/tview"
	"libvirt.org/go/libvirt"
)

type diskAction struct {
	title      string
	submitfunc func(*tview.Form)
}

func createDiskForm(pages *tview.Pages, dom *libvirt.Domain, diskAction diskAction) *tview.Form {
	log.Println("cdf enter")
	vmName, err := dom.GetName()
	if err != nil {
		vmName = ""
	}
	form := tview.NewForm()
	form.
		AddInputField("Disk Path", "", 20, nil, nil).
		AddInputField("Target Dev", "", 20, nil, nil).
		AddButton("Submit", func() {
			diskAction.submitfunc(form)
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("MainTable")
			pages.RemovePage("DiskForm")
		})
	form.SetBorder(true).SetTitle(diskAction.title + " " + vmName).SetTitleAlign(tview.AlignLeft)
	log.Println("cdf leave")

	return form
}
func createDiskGrid(pages *tview.Pages, dom *libvirt.Domain, diskAction diskAction) *tview.Grid {
	form := createDiskForm(pages, dom, diskAction)
	diskGrid := tview.NewGrid().
		SetRows(0, 1).
		SetColumns(0).
		SetBorders(false).
		AddItem(statusView, 1, 0, 1, 1, 0, 0, false).
		AddItem(form, 0, 0, 1, 1, 0, 0, true)
	return diskGrid
}
func attachDisk(dom *libvirt.Domain, app *tview.Application, pages *tview.Pages) error {
	attachAction := diskAction{
		"Attach disk to",
		func(form *tview.Form) {
			diskPath := form.GetFormItemByLabel("Disk Path").(*tview.InputField).GetText()
			targetDev := form.GetFormItemByLabel("Target Dev").(*tview.InputField).GetText()
			if err := attachSpecificDisk(dom, diskPath, targetDev); err != nil {
				log.Println("Failed to attach disk:", err)
				setStatus("Failed to attach disk: " + err.Error())
			} else {
				log.Println("Disk attached successfully")
				pages.SwitchToPage("MainTable")
				pages.RemovePage("DiskForm")
			}
		},
	}

	pages.AddPage("DiskForm", createDiskGrid(pages, dom, attachAction), true, false)
	pages.SwitchToPage("DiskForm")
	return nil
}

func detachDisk(dom *libvirt.Domain, app *tview.Application, pages *tview.Pages) error {
	detachAction := diskAction{
		"Detach disk from",
		func(form *tview.Form) {
			diskPath := form.GetFormItemByLabel("Disk Path").(*tview.InputField).GetText()
			targetDev := form.GetFormItemByLabel("Target Dev").(*tview.InputField).GetText()
			if err := attachSpecificDisk(dom, diskPath, targetDev); err != nil {
				log.Println("Failed to detach disk:", err)
				setStatus("Failed to detach disk: " + err.Error())
			} else {
				log.Println("Disk detached successfully")
				pages.SwitchToPage("MainTable")
				pages.RemovePage("DiskForm")
			}
		},
	}
	pages.AddPage("DiskForm", createDiskGrid(pages, dom, detachAction), true, false)
	pages.SwitchToPage("DiskForm")
	return nil
}

func attachSpecificDisk(dom *libvirt.Domain, diskPath string, targetDev string) error {
	setStatus("attaching specific")
	diskXML := fmt.Sprintf(`
    <disk type='file' device='disk'>
        <driver name='qemu' type='qcow2'/>
        <source file='%s'/>
        <target dev='%s' bus='virtio'/>
    </disk>
    `, diskPath, targetDev)

	// Attach the disk to the domain
	if err := dom.AttachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to hotplug disk:", err)
		return err
	}
	log.Println("Disk hotplugged successfully")
	return nil
}

func detachSpecificDisk(dom *libvirt.Domain, diskPath string, targetDev string) error {
	diskXML := fmt.Sprintf(`
    <disk type='file' device='disk'>
        <driver name='qemu' type='qcow2'/>
        <source file='%s'/>
        <target dev='%s' bus='virtio'/>
    </disk>
    `, diskPath, targetDev)

	// Detach the disk from the domain
	if err := dom.DetachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to detach disk:", err)
		return err
	}
	log.Println("Disk detached successfully")
	return nil
}
