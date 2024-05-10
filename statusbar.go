package main

import (
	"strings"

	"github.com/rivo/tview"
)

var statusView *tview.TextView = transparentTextView("")

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
