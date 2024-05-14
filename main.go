package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"libvirt.org/go/libvirt"
)

func setCellSpaces(table *tview.Table, row, col int, text string) {
	table.SetCellSimple(row, col, " "+text+" ")
}

func humanState(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_NOSTATE:
		return "No state"
	case libvirt.DOMAIN_RUNNING:
		return "Running"
	case libvirt.DOMAIN_BLOCKED:
		return "Blocked"
	case libvirt.DOMAIN_PAUSED:
		return "Paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "Shutting down"
	case libvirt.DOMAIN_SHUTOFF:
		return "Shut off"
	case libvirt.DOMAIN_CRASHED:
		return "Crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "Suspended"
	default:
		return "Unknown"
	}
}
func attachDiskXML(dom *libvirt.Domain, diskXML string) error {
	// Attach the disk to the domain
	if err := dom.AttachDeviceFlags(diskXML, libvirt.DOMAIN_DEVICE_MODIFY_LIVE); err != nil {
		log.Println("Failed to hotplug disk:", err)
		return err
	}
	log.Println("Disk hotplugged successfully")
	return nil
}

func createTable() *tview.Table {
	table := tview.NewTable().
		SetBorders(false).
		SetSeparator(tview.Borders.Vertical).
		SetSelectable(true, false).
		SetFixed(1, 1)
	setCellSpaces(table, 0, 0, "Name")
	setCellSpaces(table, 0, 1, "State")
	setCellSpaces(table, 0, 2, "CPU Usage")
	setCellSpaces(table, 0, 3, "Memory Usage")
	setCellSpaces(table, 0, 4, "I/O")
	setCellSpaces(table, 0, 5, "Network Usage")
	table.Select(1, 0)
	table.SetBackgroundColor(tcell.ColorDefault)

	// header diffrent background color
	for i := 0; i < 6; i++ {
		table.GetCell(0, i).SetBackgroundColor(tcell.ColorDarkMagenta)
		table.GetCell(0, i).SetSelectable(false)
	}

	return table
}

func runTableRefresher(app *tview.Application, table *tview.Table, conn *libvirt.Connect) {

	statProviders := make(map[string]StatProvider)
	ticker := time.NewTicker(1 * time.Second)

	for {
		domainList, err := conn.ListAllDomains(0)
		if err != nil {
			log.Println("Failed to get domain list:", err)
			continue
		}

		select {
		case <-ticker.C:
			for i, domain := range domainList {
				name, err := domain.GetName()
				if err != nil {
					log.Println("Failed to get domain name:", err)
				}
				setCellSpaces(table, i+1, 0, name)
				st, _, err := domain.GetState()
				if err != nil {
					log.Println("Failed to get domain state:", err)
				}
				setCellSpaces(table, i+1, 1, humanState(st))

				if st == libvirt.DOMAIN_SHUTOFF {
					for j := 2; j < 6; j++ {
						table.SetCellSimple(i+1, j, "")
					}
					continue
				}

				domStatProvider, ok := statProviders[name]
				if !ok {
					domStatProvider = *NewStatProvider(&domain)
					statProviders[name] = domStatProvider
				}

				CPU, err := domStatProvider.getCPUUsage(1)
				if err != nil {
					log.Println("Failed to get CPU usage:", err)

				}
				setCellSpaces(table, i+1, 2, CPU)
				//println(CPU)
				netStats, err := domStatProvider.getNetworkStats(1)
				if err != nil {
					log.Println("Failed to get network stats:", err)
				}
				setCellSpaces(table, i+1, 5, netStats)

				diskStats, err := domStatProvider.getDiskStats(1)
				if err != nil {
					log.Println("Failed to get disk stats:", err)
				}
				setCellSpaces(table, i+1, 4, diskStats)

				memStats, err := domStatProvider.getMemoryStats()
				if err != nil {
					log.Println("Failed to get memory stats:", err)
				}
				setCellSpaces(table, i+1, 3, memStats)

			}
			updateStatusHeight()
			app.Draw()

		}
	}

}

func transparentTextView(text string) *tview.TextView {
	view := tview.NewTextView()
	view.SetText(text).SetBackgroundColor(tcell.ColorDefault)
	return view
}

// TODO: improve keybinds, this would be nightmare on non qwerty layouts
func keybindsGrid() *tview.Grid {
	grid := tview.NewGrid().
		SetRows(1, 1).
		SetColumns(0, 0, 0, 0).
		SetBorders(false).
		AddItem(transparentTextView("^Q: Start"), 0, 0, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^A: Stop"), 1, 0, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^W: Resume"), 0, 1, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^S: Suspend"), 1, 1, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^E: Reboot"), 0, 2, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^D: Destroy"), 1, 2, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^R: Attach disk"), 0, 3, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^F: Detach disk"), 1, 3, 1, 1, 0, 0, false)

	return grid
}

var grid *tview.Grid

func main() {
	f, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer f.Close()
	log.SetOutput(f)
	var connectionURI string
	flag.StringVar(&connectionURI, "c", "qemu:///system", "libvirt connection URI")
	flag.Parse()
	statusView.SetBackgroundColor(tcell.ColorLightGray)
	statusView.SetTextColor(tcell.ColorBlack)
	var app *tview.Application = tview.NewApplication()
	pages := tview.NewPages()
	var actions = initActions(app, pages)
	table := createTable()

	grid = tview.NewGrid().
		SetRows(0, 1, 2).
		SetColumns(0).
		SetBorders(false).
		AddItem(statusView, 1, 0, 1, 1, 0, 0, false).
		AddItem(keybindsGrid(), 2, 0, 1, 1, 0, 0, false).
		AddItem(table, 0, 0, 1, 1, 0, 0, true)

	conn, err := libvirt.NewConnect(connectionURI)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		return handleKeypress(conn, table, event, actions)
	})

	go runTableRefresher(app, table, conn)

	pages.AddAndSwitchToPage("MainTable", grid, true)
	if err := app.SetRoot(pages, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}

}
