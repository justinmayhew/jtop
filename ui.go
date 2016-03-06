package main

import (
	"fmt"
	"strconv"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

const (
	headerRows = 1

	titleFG     = termbox.ColorBlack
	titleBG     = termbox.ColorGreen
	titleSortBG = termbox.ColorCyan

	selectedFG = termbox.ColorBlack
	selectedBG = termbox.ColorCyan

	offsetStep = 5
)

type Column struct {
	Title      string
	Width      int
	RightAlign bool
}

var (
	PidColumn        = Column{"PID", 5, true}
	UserColumn       = Column{"USER", 8, false}
	RSSColumn        = Column{"RSS", 5, true}
	MemPercentColumn = Column{"%MEM", 5, true}
	CPUPercentColumn = Column{"%CPU", 5, true}
	CPUTimeColumn    = Column{"TIME+", 9, true}
	StateColumn      = Column{"S", 1, false}
	CommandColumn    = Column{"COMMAND", -1, false}

	Columns = []Column{
		PidColumn,
		UserColumn,
		RSSColumn,
		MemPercentColumn,
		CPUPercentColumn,
		CPUTimeColumn,
		StateColumn,
		CommandColumn,
	}
)

type UI struct {
	monitor *Monitor

	x int
	y int

	offset int

	fg termbox.Attribute
	bg termbox.Attribute

	start    int
	selected int

	width  int
	height int
}

func NewUI(monitor *Monitor) *UI {
	ui := &UI{
		monitor: monitor,
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

	for _, column := range Columns {
		ui.bg = bgForTitle(column.Title)
		ui.writeColumn(column.Title, column.Width, column.RightAlign)
	}

	ui.bg = titleBG
	ui.writeLastColumn("")

	ui.y++
}

func (ui *UI) drawProcess(i int, process *Process) {
	ui.x = 0
	ui.fg, ui.bg = termbox.ColorDefault, termbox.ColorDefault
	if i == ui.selected {
		ui.fg, ui.bg = selectedFG, selectedBG
	}

	// Pid
	pid := strconv.FormatUint(process.Pid, 10)
	ui.writeColumn(pid, PidColumn.Width, PidColumn.RightAlign)

	// User
	user := runewidth.Truncate(process.User.Username, UserColumn.Width, "+")
	ui.writeColumn(user, UserColumn.Width, UserColumn.RightAlign)

	// RSS
	rssB := process.RSS * ui.monitor.PageSize
	rss := fmt.Sprintf("%dM", rssB/MB)
	if rssB < MB {
		if rssB == 0 {
			// As far as I've seen only kernel threads have 0 RSS.
			rss = "0"
		} else {
			rss = fmt.Sprintf("%dK", rssB/KB)
		}
	}
	ui.writeColumn(rss, RSSColumn.Width, RSSColumn.RightAlign)

	// Memory Percentage
	memUsage := 100 * float64(rssB) / float64(ui.monitor.MemTotal)
	mem := fmt.Sprintf("%.1f", memUsage)
	ui.writeColumn(mem, MemPercentColumn.Width, MemPercentColumn.RightAlign)

	// CPU Percentage
	totalUsage := float64(ui.monitor.CPUTimeDiff)
	userUsage := 100 * float64(process.UtimeDiff) / totalUsage
	systemUsage := 100 * float64(process.StimeDiff) / totalUsage
	cpu := fmt.Sprintf("%.1f", (userUsage+systemUsage)*float64(ui.monitor.NumCPUs))
	ui.writeColumn(cpu, CPUPercentColumn.Width, CPUPercentColumn.RightAlign)

	// CPU Time
	hertz := uint64(100)
	// TODO: this has only been tested on my Ubuntu 14.04 system that has
	// a CLK_TCK of 100. Test on other configurations. (getconf CLK_TCK)
	totalJiffies := process.Utime + process.Stime
	totalSeconds := totalJiffies / hertz

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	hundredths := totalJiffies % hertz

	// FIXME: this won't be pretty when minutes gets big, maybe format hours?
	time := fmt.Sprintf("%d:%02d:%02d", minutes, seconds, hundredths)
	ui.writeColumn(time, CPUTimeColumn.Width, CPUTimeColumn.RightAlign)

	// State
	tmpFG := ui.fg
	if i != ui.selected {
		switch process.State {
		case 'R':
			ui.fg = termbox.ColorGreen
		}
	}
	ui.writeColumn(string(process.State), StateColumn.Width, StateColumn.RightAlign)
	ui.fg = tmpFG

	// Command
	command := process.Name
	if verboseFlag {
		command = process.Command
	}
	if treeFlag {
		ui.writeCommandWithPrefix(command, process.TreePrefix)
	} else {
		ui.writeLastColumn(command)
	}

	ui.y++
}

func (ui *UI) HandleResize() {
	ui.updateTerminalSize()
}

func (ui *UI) HandleLeft() {
	if ui.offset > 0 {
		ui.offset--
	}
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

func (ui *UI) HandleRight() {
	ui.offset++
}

func (ui *UI) HandleResetOffset() {
	ui.offset = 0
}

func (ui *UI) HandleCtrlD() {
	halfPage := ui.numProcessesOnScreen() / 2
	for i := 0; i < halfPage; i++ {
		ui.HandleDown()
	}
}

func (ui *UI) HandleCtrlU() {
	halfPage := ui.numProcessesOnScreen() / 2
	for i := 0; i < halfPage; i++ {
		ui.HandleUp()
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
	bottom := len(ui.monitor.List) - 1
	if len(ui.monitor.List) > ui.numProcessesOnScreen() {
		// Not all processes fit on the same screen
		bottom = ui.numProcessesOnScreen() - 1
	}
	return ui.selected == bottom
}

func (ui *UI) topSelected() bool {
	return ui.selected == 0
}

func (ui *UI) moreProcessesDown() bool {
	return len(ui.monitor.List)-ui.start > ui.numProcessesOnScreen()
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
	end := len(ui.monitor.List)

	// Maybe they won't
	if end > ui.numProcessesOnScreen() {
		end = ui.start + ui.numProcessesOnScreen()

		// Maybe we need to scroll up because some process(es) died
		if end > len(ui.monitor.List) {
			diff := end - len(ui.monitor.List)
			ui.start -= diff
			end -= diff
		}
	}

	// When bottom process is selected and a process dies, update selected
	// to the new bottom process.
	if ui.selected >= end {
		ui.selected = end - 1
	}

	if treeFlag {
		init := ui.monitor.Map[InitPid]
		treeList := init.TreeList(0)
		if kernelFlag {
			kthreadd := ui.monitor.Map[KthreaddPid]
			treeList = append(treeList, kthreadd.TreeList(0)...)
		}
		return treeList[ui.start:end]
	}
	return ui.monitor.List[ui.start:end]
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

func (ui *UI) writeCommandWithPrefix(command, prefix string) {
	previous := ui.fg

	ui.fg = termbox.ColorBlack
	for _, ch := range prefix {
		ui.setCell(ch)
	}

	ui.fg = previous
	ui.writeLastColumn(command)
}

func (ui *UI) setCell(ch rune) {
	termbox.SetCell(ui.x-(ui.offset*offsetStep), ui.y, ch, ui.fg, ui.bg)
	ui.x += runewidth.RuneWidth(ch)
}

func bgForTitle(column string) termbox.Attribute {
	if column == sortFlag {
		return titleSortBG
	}
	return titleBG
}
