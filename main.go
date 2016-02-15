package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/nsf/termbox-go"
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

Loop:
	for {
		processes := getRunningProcesses()
		sort.Sort(ByPid(processes))
		drawProcessList(processes)

		select {
		case <-time.After(time.Second):
			// nothing, redraw on next iteration

		case ev := <-events:
			if ev.Type == termbox.EventKey {
				if ev.Ch == 'q' {
					break Loop
				}

				// TODO: handle other user input
			}
		}
	}
}

func drawProcessList(processes []Process) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	y := 0
	x := 0

	// 32768 is the max pid on my system, so ensure the column is at least 5 wide
	pidColumnTitle := "PID"
	pidColumnWidth := len(pidColumnTitle) + 2

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

	// command title
	for _, ch := range commandColumnTitle {
		setTitleCell(&x, y, ch, termbox.ColorGreen)
	}

	// finish header background
	w, _ := termbox.Size()
	for x < w {
		setTitleCell(&x, y, ' ', termbox.ColorGreen)
	}

	y++

	for _, process := range processes {
		x = 0
		strPid := strconv.Itoa(process.Pid)
		pidLength := len(strPid)

		// spaces to right align pid
		for i := 0; i < pidColumnWidth-pidLength; i++ {
			setCell(&x, y, ' ')
		}

		// pid
		for _, ch := range strPid {
			setCell(&x, y, ch)
		}

		// space to separate column
		setCell(&x, y, ' ')

		// command
		for _, ch := range process.Command {
			setCell(&x, y, ch)
		}

		y++
	}

	termbox.Flush()
}

func setTitleCell(x *int, y int, ch rune, bg termbox.Attribute) {
	termbox.SetCell(*x, y, ch, termbox.ColorBlack, bg)
	*x++
}

func setCell(x *int, y int, ch rune) {
	termbox.SetCell(*x, y, ch, termbox.ColorDefault, termbox.ColorDefault)
	*x++
}
