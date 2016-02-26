package main

import (
	"fmt"
	"strconv"

	"github.com/mattn/go-runewidth"
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

	commandColumnTitle = "COMMAND"

	titleFG     = termbox.ColorBlack
	titleBG     = termbox.ColorGreen
	titleSortBG = termbox.ColorCyan
)

type UI struct {
	pm *ProcessMonitor

	x int
	y int

	fg termbox.Attribute
	bg termbox.Attribute

	start    int
	selected int

	width  int
	height int
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
	ui.drawHeader()
	for i, process := range ui.visibleProcesses() {
		ui.drawProcess(i, process)
	}
	termbox.Flush()
}

func (ui *UI) drawHeader() {
	ui.y, ui.x = 0, 0
	ui.fg, ui.bg = titleFG, titleBG

	ui.bg = bgForTitle("pid")
	ui.writeColumn(pidColumnTitle, pidColumnWidth, true)

	ui.bg = bgForTitle("user")
	ui.writeColumn(userColumnTitle, userColumnWidth, false)

	ui.bg = bgForTitle("cpu")
	ui.writeColumn(cpuColumnTitle, cpuColumnWidth, true)

	ui.bg = bgForTitle("time")
	ui.writeColumn(timeColumnTitle, timeColumnWidth, true)

	ui.bg = bgForTitle("command")
	ui.writeColumn(commandColumnTitle, len(commandColumnTitle), false)

	ui.bg = titleBG
	ui.writeLastColumn("")

	ui.y++
}

func (ui *UI) drawProcess(i int, process *Process) {
	ui.x = 0
	ui.fg, ui.bg = termbox.ColorDefault, termbox.ColorDefault
	if i == ui.selected {
		ui.fg, ui.bg = termbox.ColorBlack, termbox.ColorCyan
	}

	// PID
	pidColumn := strconv.FormatUint(process.PID, 10)
	ui.writeColumn(pidColumn, pidColumnWidth, true)

	// User
	userColumn := runewidth.Truncate(process.User.Username, userColumnWidth, "+")
	ui.writeColumn(userColumn, userColumnWidth, false)

	// CPU Percentage
	totalUsage := float64(ui.pm.CPUTimeDiff)
	userUsage := 100 * float64(process.UtimeDiff) / totalUsage
	systemUsage := 100 * float64(process.StimeDiff) / totalUsage
	cpuColumn := fmt.Sprintf("%.1f", (userUsage+systemUsage)*float64(ui.pm.NumCPUs))
	ui.writeColumn(cpuColumn, cpuColumnWidth, true)

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
	ui.writeColumn(timeColumn, timeColumnWidth, true)

	// Command
	commandColumn := process.Name
	if verboseFlag {
		commandColumn = process.Command
	}
	ui.writeLastColumn(commandColumn)

	ui.y++
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

func (ui *UI) writeColumn(s string, columnWidth int, rightAlign bool) {
	sWidth := runewidth.StringWidth(s)
	if rightAlign {
		for i := 0; i < columnWidth-sWidth; i++ {
			ui.setCell(' ')
		}
	}

	for _, ch := range s {
		ui.setCell(ch)
	}

	if !rightAlign {
		for i := 0; i < columnWidth-sWidth; i++ {
			ui.setCell(' ')
		}
	}

	ui.setCell(' ')
}

func (ui *UI) writeLastColumn(s string) {
	for _, ch := range s {
		ui.setCell(ch)
	}

	for ui.x < ui.width {
		ui.setCell(' ')
	}
}

func (ui *UI) setCell(ch rune) {
	termbox.SetCell(ui.x, ui.y, ch, ui.fg, ui.bg)
	ui.x += runewidth.RuneWidth(ch)
}

func bgForTitle(column string) termbox.Attribute {
	if column == sortFlag {
		return titleSortBG
	}
	return titleBG
}
