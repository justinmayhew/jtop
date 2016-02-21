package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/nsf/termbox-go"
)

func main() {
	if err := termbox.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer termbox.Close()

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	ticker := time.Tick(1500 * time.Millisecond)
	pm := NewProcessMonitor()
	pm.Update()

	for {
		drawUserInterface(pm)

		select {
		case <-ticker:
			pm.Update()

		case ev := <-events:
			if ev.Type == termbox.EventKey {
				switch {
				case ev.Ch == 'q':
					return

				case ev.Ch == 'j' || ev.Key == termbox.KeyArrowDown:
					_, height := termbox.Size()
					numProcessRows := height - headerRows
					if selectedIdx+1 != numProcessRows {
						// not at bottom of ui
						selectedIdx++
					} else if len(pm.List)-startIdx > numProcessRows {
						// at bottom of ui and there's more processes to show,
						// scroll down
						startIdx++
					}

				case ev.Ch == 'k' || ev.Key == termbox.KeyArrowUp:
					if selectedIdx != 0 {
						// not at top of ui
						selectedIdx--
					} else if startIdx > 0 {
						// at top of ui and there's more processes to show,
						// scroll up
						startIdx--
					}
				}
			}
		}
	}
}

const (
	headerRows = 1

	pidColumnTitle = "PID"
	pidColumnWidth = 5 // 32768 is the max pid on my system

	userColumnTitle = "USER"
	userColumnWidth = 8

	cpuColumnTitle = "%CPU"
	cpuColumnWidth = 5

	commandColumnTitle = "Command"
)

var (
	startIdx    = 0
	selectedIdx = 0
)

func drawUserInterface(pm *ProcessMonitor) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height := termbox.Size()

	y := 0
	x := 0

	writeColumn(pidColumnTitle, pidColumnWidth, true, &x, y, termbox.ColorBlack, termbox.ColorGreen)
	writeColumn(userColumnTitle, userColumnWidth, false, &x, y, termbox.ColorBlack, termbox.ColorGreen)
	writeColumn(cpuColumnTitle, cpuColumnWidth, true, &x, y, termbox.ColorBlack, termbox.ColorCyan)
	writeLastColumn(commandColumnTitle, width, x, y, termbox.ColorBlack, termbox.ColorGreen)

	y++

	displayProcesses := pm.List[startIdx:len(pm.List)]
	if startIdx+height < len(pm.List) {
		displayProcesses = pm.List[startIdx : startIdx+height]
	}

	for i, process := range displayProcesses {
		x = 0

		fg := termbox.ColorDefault
		bg := termbox.ColorDefault

		if i == selectedIdx {
			fg = termbox.ColorBlack
			bg = termbox.ColorCyan
		}

		// PID
		pidColumn := strconv.Itoa(process.Pid)
		writeColumn(pidColumn, pidColumnWidth, true, &x, y, fg, bg)

		// User
		maxUserLen := len(process.User.Username)
		if maxUserLen > userColumnWidth {
			maxUserLen = userColumnWidth
		}
		userColumn := process.User.Username[0:maxUserLen]
		writeColumn(userColumn, userColumnWidth, false, &x, y, fg, bg)

		// CPU Percentage
		totalUsage := float64(pm.CPUTimeDiff)
		userUsage := 100 * float64(process.UtimeDiff) / totalUsage
		systemUsage := 100 * float64(process.StimeDiff) / totalUsage
		cpuColumn := fmt.Sprintf("%.1f", (userUsage+systemUsage)*float64(pm.NumCPUs))
		writeColumn(cpuColumn, cpuColumnWidth, true, &x, y, fg, bg)

		// Command
		writeLastColumn(process.Command, width, x, y, fg, bg)

		y++
	}

	termbox.Flush()
}

func writeColumn(s string, columnWidth int, rightAlign bool, x *int, y int, fg, bg termbox.Attribute) {
	if rightAlign {
		for i := 0; i < columnWidth-len(s); i++ {
			termbox.SetCell(*x, y, ' ', fg, bg)
			*x++
		}
	}

	for _, ch := range s {
		termbox.SetCell(*x, y, ch, fg, bg)
		*x++
	}

	if !rightAlign {
		for i := 0; i < columnWidth-len(s); i++ {
			termbox.SetCell(*x, y, ' ', fg, bg)
			*x++
		}
	}

	// Space to separate columns
	termbox.SetCell(*x, y, ' ', fg, bg)
	*x++
}

func writeLastColumn(s string, terminalWidth, x, y int, fg, bg termbox.Attribute) {
	for _, ch := range s {
		termbox.SetCell(x, y, ch, fg, bg)
		x++
	}

	for x < terminalWidth {
		termbox.SetCell(x, y, ' ', fg, bg)
		x++
	}
}
