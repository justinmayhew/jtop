package main

import (
	"fmt"
	"strconv"

	"github.com/nsf/termbox-go"
)

const (
	headerRows = 1

	pidColumnTitle = "PID"
	pidColumnWidth = 5 // 32768 is the max pid on my system

	userColumnTitle = "USER"
	userColumnWidth = 8

	cpuColumnTitle = "%CPU"
	cpuColumnWidth = 5

	timeColumnTitle = "TIME+"
	timeColumnWidth = 8

	commandColumnTitle = "Command"
)

type UI struct {
	pm *ProcessMonitor

	width  int
	height int

	startIdx    int
	selectedIdx int
}

func NewUI(pm *ProcessMonitor) *UI {
	ui := &UI{
		pm: pm,
	}
	ui.updateTerminalSize()
	return ui
}

func (ui *UI) Draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	y := 0
	x := 0

	writeColumn(pidColumnTitle, pidColumnWidth, true, &x, y, termbox.ColorBlack, termbox.ColorGreen)
	writeColumn(userColumnTitle, userColumnWidth, false, &x, y, termbox.ColorBlack, termbox.ColorGreen)
	writeColumn(cpuColumnTitle, cpuColumnWidth, true, &x, y, termbox.ColorBlack, termbox.ColorCyan)
	writeColumn(timeColumnTitle, timeColumnWidth, true, &x, y, termbox.ColorBlack, termbox.ColorGreen)
	writeLastColumn(commandColumnTitle, ui.width, x, y, termbox.ColorBlack, termbox.ColorGreen)

	y++

	displayProcesses := ui.pm.List[ui.startIdx:len(ui.pm.List)]
	if ui.startIdx+ui.height < len(ui.pm.List) {
		displayProcesses = ui.pm.List[ui.startIdx : ui.startIdx+ui.height]
	}

	for i, process := range displayProcesses {
		x = 0

		fg := termbox.ColorDefault
		bg := termbox.ColorDefault

		if i == ui.selectedIdx {
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
		totalUsage := float64(ui.pm.CPUTimeDiff)
		userUsage := 100 * float64(process.UtimeDiff) / totalUsage
		systemUsage := 100 * float64(process.StimeDiff) / totalUsage
		cpuColumn := fmt.Sprintf("%.1f", (userUsage+systemUsage)*float64(ui.pm.NumCPUs))
		writeColumn(cpuColumn, cpuColumnWidth, true, &x, y, fg, bg)

		// Time
		hertz := uint64(100)
		// TODO: this has only been tested on my Ubuntu 14.04 system that has
		// a CLK_TICK of 100. Test on other configurations. (getconf CLK_TICK)
		totalJiffies := process.Utime + process.Stime
		totalSeconds := totalJiffies / hertz

		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		hundredths := totalJiffies % hertz

		// FIXME: this won't be pretty when minutes gets big, maybe format hours?
		timeColumn := fmt.Sprintf("%d:%02d:%02d", minutes, seconds, hundredths)
		writeColumn(timeColumn, timeColumnWidth, true, &x, y, fg, bg)

		// Command
		writeLastColumn(process.Command, ui.width, x, y, fg, bg)

		y++
	}

	termbox.Flush()
}

func (ui *UI) HandleResize() {
	ui.updateTerminalSize()
}

func (ui *UI) HandleDown() {
	if ui.shouldScrollDown() {
		ui.startIdx++
		return
	}

	if !ui.bottomSelected() {
		ui.selectedIdx++
	}
}

func (ui *UI) HandleUp() {
	if ui.shouldScrollUp() {
		ui.startIdx--
		return
	}

	if !ui.topSelected() {
		ui.selectedIdx--
	}
}

func (ui *UI) shouldScrollDown() bool {
	return ui.bottomSelected() && ui.moreProcessesToShow()
}

func (ui *UI) shouldScrollUp() bool {
	return ui.topSelected() && ui.startIdx > 0
}

func (ui *UI) bottomSelected() bool {
	bottomIdx := len(ui.pm.List) - 1
	if len(ui.pm.List) > ui.numProcessesOnScreen() {
		// Not all processes fit on the same screen
		bottomIdx = ui.numProcessesOnScreen() - 1
	}
	return ui.selectedIdx == bottomIdx
}

func (ui *UI) topSelected() bool {
	return ui.selectedIdx == 0
}

func (ui *UI) moreProcessesToShow() bool {
	return len(ui.pm.List)-ui.startIdx > ui.numProcessesOnScreen()
}

func (ui *UI) numProcessesOnScreen() int {
	return ui.height - headerRows
}

func (ui *UI) updateTerminalSize() {
	ui.width, ui.height = termbox.Size()
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
