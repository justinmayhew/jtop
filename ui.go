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
	timeColumnWidth = 9

	commandColumnTitle = "Command"

	titleFG     = termbox.ColorBlack
	titleBG     = termbox.ColorGreen
	titleSortBG = termbox.ColorCyan
)

type UI struct {
	pm *ProcessMonitor

	width  int
	height int

	start    int
	selected int
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

	fg := titleFG
	bg := titleBG

	bg = bgForTitle("pid")
	writeColumn(pidColumnTitle, pidColumnWidth, true, &x, y, fg, bg)

	bg = bgForTitle("user")
	writeColumn(userColumnTitle, userColumnWidth, false, &x, y, fg, bg)

	bg = bgForTitle("cpu")
	writeColumn(cpuColumnTitle, cpuColumnWidth, true, &x, y, fg, bg)

	bg = bgForTitle("time")
	writeColumn(timeColumnTitle, timeColumnWidth, true, &x, y, fg, bg)

	bg = bgForTitle("command")
	writeColumn(commandColumnTitle, len(commandColumnTitle), false, &x, y, fg, bg)

	bg = titleBG
	writeLastColumn("", ui.width, x, y, fg, bg)

	y++

	for i, process := range ui.visibleProcesses() {
		x = 0

		fg = termbox.ColorDefault
		bg = termbox.ColorDefault

		if i == ui.selected {
			fg = termbox.ColorBlack
			bg = termbox.ColorCyan
		}

		// PID
		pidColumn := strconv.FormatUint(process.PID, 10)
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
		// a CLK_TCK of 100. Test on other configurations. (getconf CLK_TCK)
		totalJiffies := process.Utime + process.Stime
		totalSeconds := totalJiffies / hertz

		minutes := totalSeconds / 60
		seconds := totalSeconds % 60
		hundredths := totalJiffies % hertz

		// FIXME: this won't be pretty when minutes gets big, maybe format hours?
		timeColumn := fmt.Sprintf("%d:%02d:%02d", minutes, seconds, hundredths)
		writeColumn(timeColumn, timeColumnWidth, true, &x, y, fg, bg)

		// Command
		commandColumn := process.Name
		if verboseFlag {
			commandColumn = process.Command
		}
		writeLastColumn(commandColumn, ui.width, x, y, fg, bg)

		y++
	}

	termbox.Flush()
}

func (ui *UI) HandleResize() {
	ui.updateTerminalSize()
}

func (ui *UI) HandleDown() {
	if ui.shouldScrollDown() {
		ui.scrollDown()
		return
	}

	if !ui.bottomSelected() {
		ui.down()
	}
}

func (ui *UI) HandleUp() {
	if ui.shouldScrollUp() {
		ui.scrollUp()
		return
	}

	if !ui.topSelected() {
		ui.up()
	}
}

func (ui *UI) down() {
	ui.selected++
}

func (ui *UI) up() {
	ui.selected--
}

func (ui *UI) scrollDown() {
	ui.start++
}

func (ui *UI) scrollUp() {
	ui.start--
}

func (ui *UI) shouldScrollDown() bool {
	return ui.bottomSelected() && ui.moreProcessesDown()
}

func (ui *UI) shouldScrollUp() bool {
	return ui.topSelected() && ui.moreProcessesUp()
}

func (ui *UI) bottomSelected() bool {
	bottom := len(ui.pm.List) - 1
	if len(ui.pm.List) > ui.numProcessesOnScreen() {
		// Not all processes fit on the same screen
		bottom = ui.numProcessesOnScreen() - 1
	}
	return ui.selected == bottom
}

func (ui *UI) topSelected() bool {
	return ui.selected == 0
}

func (ui *UI) moreProcessesDown() bool {
	return len(ui.pm.List)-ui.start > ui.numProcessesOnScreen()
}

func (ui *UI) moreProcessesUp() bool {
	return ui.start > 0
}

func (ui *UI) numProcessesOnScreen() int {
	return ui.height - headerRows
}

func (ui *UI) updateTerminalSize() {
	ui.width, ui.height = termbox.Size()
}

func (ui *UI) visibleProcesses() []*Process {
	// Maybe all processes will fit on the same screen
	end := len(ui.pm.List)

	// Maybe they won't
	if end > ui.numProcessesOnScreen() {
		end = ui.start + ui.numProcessesOnScreen()

		// Maybe we need to scroll up because some process(es) died
		if end > len(ui.pm.List) {
			diff := end - len(ui.pm.List)
			ui.start -= diff
			end -= diff
		}
	}

	// When bottom process is selected and a process dies, update selected
	// to the new bottom process.
	if ui.selected >= end {
		ui.selected = end - 1
	}

	return ui.pm.List[ui.start:end]
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

func bgForTitle(title string) termbox.Attribute {
	if title == sortFlag {
		return titleSortBG
	}
	return titleBG
}
