package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"libvirt.org/go/libvirt"
)

type Message struct {
	actionStart   string
	actionSuccess string
	actionFail    string
}

func (m Message) StartMessage() string {
	return m.actionStart
}

func (m Message) SuccessMessage() string {
	return m.actionSuccess
}

func (m Message) FailMessage() string {
	return m.actionFail
}

type Action interface {
	Execute(dom *libvirt.Domain) error
	StartMessage() string
	SuccessMessage() string
	FailMessage() string
}

type SimpleAction struct {
	actionFunc func(dom *libvirt.Domain) error
	Message
}

func (a SimpleAction) Execute(dom *libvirt.Domain) error {
	return a.actionFunc(dom)
}

type UIAction struct {
	Message
	actionFunc func(dom *libvirt.Domain, app *tview.Application, pages *tview.Pages) error
	app        *tview.Application
	pages      *tview.Pages
}

func (a UIAction) Execute(dom *libvirt.Domain) error {
	return a.actionFunc(dom, a.app, a.pages)
}

func NewSimpleAction(start, success, fail string, actionFunc func(*libvirt.Domain) error) SimpleAction {
	return SimpleAction{Message: Message{start, success, fail}, actionFunc: actionFunc}
}

func NewUIAction(start, success, fail string, actionFunc func(*libvirt.Domain, *tview.Application, *tview.Pages) error, app *tview.Application, pages *tview.Pages) UIAction {
	return UIAction{Message: Message{start, success, fail}, actionFunc: actionFunc, app: app, pages: pages}
}

func initActions(app *tview.Application, pages *tview.Pages) map[tcell.Key]Action {
	return map[tcell.Key]Action{
		tcell.KeyCtrlQ: NewSimpleAction("Starting", "started", "Failed to start", (*libvirt.Domain).Create),
		tcell.KeyCtrlA: NewSimpleAction("Stopping", "stopped", "Failed to stop", (*libvirt.Domain).Shutdown),
		tcell.KeyCtrlW: NewSimpleAction("Resuming", "resumed", "Failed to resume", (*libvirt.Domain).Resume),
		tcell.KeyCtrlS: NewSimpleAction("Suspending", "suspended", "Failed to suspend", (*libvirt.Domain).Suspend),
		tcell.KeyCtrlE: NewSimpleAction("Rebooting", "rebooted", "Failed to reboot", func(d *libvirt.Domain) error { return d.Reboot(0) }),
		tcell.KeyCtrlD: NewSimpleAction("Destroying", "destroyed", "Failed to destroy", (*libvirt.Domain).Destroy),
		tcell.KeyCtrlR: NewUIAction("Attaching disk to", "attached disk to", "Failed to attach disk to", attachDisk, app, pages),
		tcell.KeyCtrlF: NewUIAction("Detaching disk from", "detached disk from", "Failed to detach disk from", detachDisk, app, pages),
	}
}
