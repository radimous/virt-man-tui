package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/beevik/etree"
	"github.com/rivo/tview"
	"libvirt.org/go/libvirt"
)

type diskAction struct {
	title      string
	submitfunc func(*tview.Form) error
}
type Disk struct {
	Device string
	File   string
	Type   string
}

func createAttachDiskForm(pages *tview.Pages, dom *libvirt.Domain, diskAction diskAction) *tview.Form {
	vmName, err := dom.GetName()
	if err != nil {
		vmName = ""
	}
	form := tview.NewForm()
	diskPathInput := tview.NewInputField().
		SetLabel("Disk Path: ").
		SetFieldWidth(30).
		SetAutocompleteFunc(func(currentText string) (entries []string) {
			if len(currentText) == 0 {
				return
			}
			dir, prefix := filepath.Split(currentText)
			files, _ := filepath.Glob(dir + "*")
			for _, file := range files {
				if strings.HasPrefix(file, dir+prefix) {
					entries = append(entries, file)
				}
			}

			if len(entries) <= 1 {
				entries = nil
			}

			return
		})
	targetDevInput := tview.NewInputField().
		SetLabel("Target Dev: ").
		SetFieldWidth(30)
	form.
		AddFormItem(diskPathInput).
		AddFormItem(targetDevInput).
		AddButton("Submit", func() {
			diskAction.submitfunc(form)
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("MainTable")
			pages.RemovePage("DiskForm")
		})
	form.SetBorder(true).SetTitle(diskAction.title + " " + vmName).SetTitleAlign(tview.AlignLeft)
	return form
}

func createAttachDiskGrid(pages *tview.Pages, dom *libvirt.Domain, diskAction diskAction) *tview.Grid {
	form := createAttachDiskForm(pages, dom, diskAction)
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
		func(form *tview.Form) error {
			diskPath := form.GetFormItemByLabel("Disk Path: ").(*tview.InputField).GetText()
			targetDev := form.GetFormItemByLabel("Target Dev: ").(*tview.InputField).GetText()
			if err := attachSpecificDisk(dom, diskPath, targetDev); err != nil {
				log.Println("Failed to attach disk:", err)
				setStatus("Failed to attach disk: " + err.Error())
				return err
			} else {
				log.Println("Disk attached successfully")
				pages.SwitchToPage("MainTable")
				pages.RemovePage("DiskForm")
			}
			return nil
		},
	}

	pages.AddPage("DiskForm", createAttachDiskGrid(pages, dom, attachAction), true, false)
	pages.SwitchToPage("DiskForm")
	return nil
}

// TODO: disk type
func attachSpecificDisk(dom *libvirt.Domain, diskPath string, targetDev string) error {
	diskXML := fmt.Sprintf(`
    <disk type='file' device='disk'>
        <driver name='qemu' type='qcow2'/>
        <source file='%s'/>
        <target dev='%s' bus='virtio'/>
    </disk>
    `, diskPath, targetDev)

	if err := dom.AttachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to hotplug disk:", err)
		return err
	}
	log.Println("Disk hotplugged successfully")
	return nil
}

func createDiskList(dom *libvirt.Domain) ([]Disk, error) {
	var disks []Disk

	xmlDesc, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get XML description: %w", err)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromString(xmlDesc); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	diskElements := doc.FindElements("//disk")
	for _, disk := range diskElements {
		source := disk.SelectElement("source")
		target := disk.SelectElement("target")
		if source != nil && target != nil {
			file := source.SelectAttrValue("file", "")
			dev := target.SelectAttrValue("dev", "")
			disks = append(disks, Disk{Device: dev, File: file})
		}
	}

	return disks, nil
}

func createDetachDiskForm(pages *tview.Pages, dom *libvirt.Domain, diskAction diskAction) *tview.Form {
	vmName, err := dom.GetName()
	if err != nil {
		vmName = ""
	}
	diskList, err := createDiskList(dom)
	diskOptions := make([]string, 0, len(diskList))

	for _, disk := range diskList {
		diskOptions = append(diskOptions, fmt.Sprintf("Device: %s, File: %s", disk.Device, disk.File))
	}
	form := tview.NewForm()
	form.
		AddDropDown("Disks: ", diskOptions, 0, func(option string, optionIndex int) {
			diskInfo := strings.Split(option, ", ")
			diskDevice := strings.Split(diskInfo[0], ": ")[1]
			diskFile := strings.Split(diskInfo[1], ": ")[1]

			if err := detachSpecificDisk(dom, diskDevice, diskFile); err != nil {
				log.Println("Failed to detach disk:", err)
				setStatus("Failed to detach disk: " + err.Error())
			} else {
				log.Println("Disk detached successfully")
				pages.SwitchToPage("MainTable")
				pages.RemovePage("DiskForm")
			}
		}).
		AddButton("Submit", func() {
			diskAction.submitfunc(form)
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("MainTable")
			pages.RemovePage("DiskForm")
		})
	form.SetBorder(true).SetTitle(diskAction.title + " " + vmName).SetTitleAlign(tview.AlignLeft)
	return form
}

func createDetachDiskGrid(pages *tview.Pages, dom *libvirt.Domain, diskAction diskAction) *tview.Grid {
	form := createDetachDiskForm(pages, dom, diskAction)
	diskGrid := tview.NewGrid().
		SetRows(0, 1).
		SetColumns(0).
		SetBorders(false).
		AddItem(statusView, 1, 0, 1, 1, 0, 0, false).
		AddItem(form, 0, 0, 1, 1, 0, 0, true)
	return diskGrid
}
func detachDisk(dom *libvirt.Domain, app *tview.Application, pages *tview.Pages) error {
	detachAction := diskAction{
		"Detach disk from",
		func(form *tview.Form) error {
			_, dropText := form.GetFormItemByLabel("Disks: ").(*tview.DropDown).GetCurrentOption()
			diskInfo := strings.Split(dropText, ", ")
			targetDev := strings.Split(diskInfo[0], ": ")[1]
			diskPath := strings.Split(diskInfo[1], ": ")[1]
			if err := detachSpecificDisk(dom, diskPath, targetDev); err != nil {
				log.Println("Failed to detach disk:", err)
				setStatus("Failed to detach disk: " + err.Error())
				return err
			} else {
				log.Println("Disk detached successfully")
				pages.SwitchToPage("MainTable")
				pages.RemovePage("DiskForm")
			}
			return nil
		},
	}
	pages.AddPage("DiskForm", createDetachDiskGrid(pages, dom, detachAction), true, false)
	pages.SwitchToPage("DiskForm")
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

	if err := dom.DetachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to detach disk:", err)
		return err
	}
	log.Println("Disk detached successfully")
	return nil
}
