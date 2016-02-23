package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

var (
	sortFlag = flag.String("sort", "cpu", "Column to sort processes by")
)

func validateSortFlag() {
	sortColumns := []string{"pid", "user", "cpu", "time"}
	for _, column := range sortColumns {
		if *sortFlag == column {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Can't sort by %s, available sort columns: %s\n",
		*sortFlag, strings.Join(sortColumns, ", "))
	os.Exit(1)
}

func validateFlags() {
	validateSortFlag()
}

func main() {
	flag.Parse()
	validateFlags()

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
	ui := NewUI(pm)

	for {
		ui.Draw()

		select {
		case <-ticker:
			pm.Update()

		case ev := <-events:
			if ev.Type == termbox.EventKey {
				switch {
				case ev.Ch == 'q':
					return

				case ev.Ch == 'j' || ev.Key == termbox.KeyArrowDown:
					ui.HandleDown()

				case ev.Ch == 'k' || ev.Key == termbox.KeyArrowUp:
					ui.HandleUp()
				}
			} else if ev.Type == termbox.EventResize {
				ui.HandleResize()
			}
		}
	}
}
