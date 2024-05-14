package main

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"libvirt.org/go/libvirt"
)

func libvirtError(err error) string {
	if libvirtErr, ok := err.(libvirt.Error); ok {
		return fmt.Sprintf("Libvirt err %d: %s", libvirtErr.Code, libvirtErr.Message)
	}
	return err.Error()
}

func handleKeypress(conn *libvirt.Connect, table *tview.Table, event *tcell.EventKey, actions map[tcell.Key]Action) *tcell.EventKey {
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

	if action, ok := actions[event.Key()]; ok {
		setStatus(action.StartMessage() + " " + vmName)
		if err := action.Execute(dom); err != nil {
			log.Println(action.FailMessage()+" domain:", err)
			setStatus(action.FailMessage() + " " + vmName + ". " + libvirtError(err))
		} else {
			log.Println("Successfully " + action.SuccessMessage() + " " + vmName)
		}
	}
	return event

}
