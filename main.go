package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	headerRows = 1
)

var (
	startIndex    = 0
	selectedIndex = 0
)

func main() {
	err := termbox.Init()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to initialize termbox.")
		os.Exit(1)
	}
	defer termbox.Close()

	events := make(chan termbox.Event)
	go func() {
		for {
			events <- termbox.PollEvent()
		}
	}()

	for {
		processes := getRunningProcesses()
		sort.Sort(ByPid(processes))
		drawProcessList(processes)

		select {
		case <-time.After(time.Second):
			// nothing, redraw on next iteration

		case ev := <-events:
			if ev.Type == termbox.EventKey {
				switch {
				case ev.Ch == 'q':
					return

				case ev.Ch == 'j' || ev.Key == termbox.KeyArrowDown:
					_, height := termbox.Size()
					numProcessRows := height - headerRows
					if selectedIndex+1 != numProcessRows {
						// not at bottom of ui
						selectedIndex++
					} else if len(processes)-startIndex > numProcessRows {
						// at bottom of ui and there's more processes to show,
						// scroll down
						startIndex++
					}

				case ev.Ch == 'k' || ev.Key == termbox.KeyArrowUp:
					if selectedIndex != 0 {
						// not at top of ui
						selectedIndex--
					} else if startIndex > 0 {
						// at top of ui and there's more processes to show,
						// scroll up
						startIndex--
					}
				}
			}
		}
	}
}

func drawProcessList(processes []Process) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	width, height := termbox.Size()

	y := 0
	x := 0

	// 32768 is the max pid on my system, so ensure the column is at least 5 wide
	pidColumnTitle := "PID"
	pidColumnWidth := len(pidColumnTitle) + 2

	userColumnTitle := "USER"
	userColumnWidth := 8

	commandColumnTitle := "Command"

	// spaces to right align pid title
	for i := 0; i < pidColumnWidth-len(pidColumnTitle); i++ {
		setTitleCell(&x, y, ' ', termbox.ColorCyan)
	}

	// pid title
	for _, ch := range pidColumnTitle {
		setTitleCell(&x, y, ch, termbox.ColorCyan)
	}

	// space to separate column
	setTitleCell(&x, y, ' ', termbox.ColorCyan)

	// user title
	for _, ch := range userColumnTitle {
		setTitleCell(&x, y, ch, termbox.ColorGreen)
	}

	// spaces to end user column
	for i := 0; i < userColumnWidth-len(userColumnTitle); i++ {
		setTitleCell(&x, y, ' ', termbox.ColorGreen)
	}

	// space to separate column
	setTitleCell(&x, y, ' ', termbox.ColorGreen)

	// command title
	for _, ch := range commandColumnTitle {
		setTitleCell(&x, y, ch, termbox.ColorGreen)
	}

	// finish header background
	for x < width {
		setTitleCell(&x, y, ' ', termbox.ColorGreen)
	}

	y++

	displayProcesses := processes[startIndex : startIndex+height]
	if startIndex+height > len(processes) {
		displayProcesses = processes[startIndex:len(processes)]
	}

	for i, process := range displayProcesses {
		x = 0
		strPid := strconv.Itoa(process.Pid)

		fg := termbox.ColorDefault
		bg := termbox.ColorDefault

		if i == selectedIndex {
			fg = termbox.ColorBlack
			bg = termbox.ColorCyan
		}

		// spaces to right align pid
		for i := 0; i < pidColumnWidth-len(strPid); i++ {
			setCell(&x, y, ' ', fg, bg)
		}

		// pid
		for _, ch := range strPid {
			setCell(&x, y, ch, fg, bg)
		}

		// space to separate column
		setCell(&x, y, ' ', fg, bg)

		// user
		maxUserLen := len(process.User.Name)
		if maxUserLen > userColumnWidth {
			maxUserLen = userColumnWidth
		}
		for _, ch := range process.User.Name[0:maxUserLen] {
			setCell(&x, y, ch, fg, bg)
		}

		// spaces to end user column (column is left-aligned)
		for i := 0; i < userColumnWidth-len(process.User.Name); i++ {
			setCell(&x, y, ' ', fg, bg)
		}

		// space to separate column
		setCell(&x, y, ' ', fg, bg)

		// command
		for _, ch := range process.Command {
			setCell(&x, y, ch, fg, bg)
		}

		if i == selectedIndex {
			// finish row background
			for x < width {
				setCell(&x, y, ' ', fg, bg)
			}
		}

		y++
	}

	termbox.Flush()
}

func setTitleCell(x *int, y int, ch rune, bg termbox.Attribute) {
	termbox.SetCell(*x, y, ch, termbox.ColorBlack, bg)
	*x++
}

func setCell(x *int, y int, ch rune, fg, bg termbox.Attribute) {
	termbox.SetCell(*x, y, ch, fg, bg)
	*x++
}
