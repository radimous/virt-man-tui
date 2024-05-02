package main

import (
	"flag"
	"log"
	"os"
	"strings"
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
func attachDisk(dom *libvirt.Domain, diskXML string) error {
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
		SetSeparator('\u2502').
		SetSelectable(true, false).
		SetFixed(1, 1)
	setCellSpaces(table, 0, 0, "Name")
	setCellSpaces(table, 0, 1, "State")
	setCellSpaces(table, 0, 2, "CPU Usage")
	setCellSpaces(table, 0, 3, "Memory")
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

	prevSt := make(map[string]prevStats)
	ticker := time.NewTicker(1 * time.Second)

	for {
		domainList, err := conn.ListAllDomains(0)
		if err != nil {
			log.Println("Failed to get domain list:", err)
			continue
		}
		//TODO: remove this
		for i := 1; i < 10; i++ {
			domainList = append(domainList, domainList[0])
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

				info, err := domain.GetInfo()
				if err != nil {
					log.Println("Failed to get domain info:", err)
				}
				numCores := info.NrVirtCpu

				domStats := prevSt[name]

				CPU, err := getCPUUsage(&domain, &domStats, numCores, 1)
				if err != nil {
					log.Println("Failed to get CPU usage:", err)

				}
				setCellSpaces(table, i+1, 2, CPU)
				//println(CPU)
				netStats, err := getNetworkStats(&domain, &domStats, 1)
				if err != nil {
					log.Println("Failed to get network stats:", err)
				}
				setCellSpaces(table, i+1, 5, netStats)

				diskStats, err := getDiskStats(&domain, &domStats, 1)
				if err != nil {
					log.Println("Failed to get disk stats:", err)
				}
				setCellSpaces(table, i+1, 4, diskStats)

				prevSt[name] = domStats
				memStats, err := getMemoryStats(&domain)
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
		AddItem(transparentTextView("^R: Hotplug disk"), 0, 3, 1, 1, 0, 0, false).
		AddItem(transparentTextView("^F: Unplug disk"), 1, 3, 1, 1, 0, 0, false)

	return grid
}

var statusView *tview.TextView = transparentTextView("")
var grid *tview.Grid

func updateStatusHeight() {
	_, _, width, _ := statusView.GetInnerRect()
	text := statusView.GetText(false)
	text = strings.ReplaceAll(text, "\n", " ")
	var builder strings.Builder
	for len(text) > width {
		spaceIndex := strings.LastIndex(text[:width], " ")
		if spaceIndex == -1 {
			spaceIndex = width
		}

		builder.WriteString(text[:spaceIndex])
		builder.WriteRune('\n')
		text = text[spaceIndex+1:]
	}

	builder.WriteString(text)
	statusView.SetText(builder.String())
	statusHeight := strings.Count(statusView.GetText(false), "\n") + 1
	if statusHeight > 6 {
		statusHeight = 6
	}
	grid.SetRows(0, statusHeight, 2)
}

func setStatus(status string) {
	statusView.SetText(status)
	updateStatusHeight()
}
func main() {
	f, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	defer f.Close()
	log.SetOutput(f)
	var connectionURI string
	flag.StringVar(&connectionURI, "c", "qemu:///system", "libvirt connection URI")
	flag.Parse()

	app := tview.NewApplication()

	table := createTable()
	statusView.SetBackgroundColor(tcell.ColorLightGray)
	statusView.SetTextColor(tcell.ColorBlack)
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
		return handleKeypress(conn, table, event)
	})

	go runTableRefresher(app, table, conn)
	if err := app.SetRoot(grid, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}

}
