package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

const usage = `Usage: gtop [options]

Options:
  -s, --sort     sort by the specified column (%s)
  -u, --users    filter by user (comma-separated list)
      --verbose  show full command line with arguments
`

const (
	defaultSortColumn = "cpu"
)

var (
	sortColumns = []string{"pid", "user", "cpu", "time"}
	sortFlag    string
	usersFlag   string
	verboseFlag bool
)

func exit(message string) {
	fmt.Fprintln(os.Stderr, message)
	flag.Usage()
	os.Exit(1)
}

func validateSortFlag() {
	for _, column := range sortColumns {
		if sortFlag == column {
			return
		}
	}
	message := fmt.Sprintf("flag error: %s is not a valid sort column", sortFlag)
	exit(message)
}

func validateUsersFlag() {
	if usersFlag == "" {
		return
	}

	users := strings.Split(usersFlag, ",")
	for _, username := range users {
		if user, err := user.Lookup(username); err != nil {
			message := fmt.Sprintf("flag error: user %s does not exist", username)
			exit(message)
		} else {
			UserWhitelist = append(UserWhitelist, user)
		}
	}
}

func validateFlags() {
	validateSortFlag()
	validateUsersFlag()
}

func init() {
	flag.StringVar(&sortFlag, "s", defaultSortColumn, "")
	flag.StringVar(&sortFlag, "sort", defaultSortColumn, "")

	flag.StringVar(&usersFlag, "u", "", "")
	flag.StringVar(&usersFlag, "users", "", "")

	flag.BoolVar(&verboseFlag, "verbose", false, "")

	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, usage, strings.Join(sortColumns, ", "))
	}
}

func main() {
	flag.Parse()
	validateFlags()

	if err := termbox.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
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
				case ev.Ch == 'v':
					verboseFlag = !verboseFlag
				}
			} else if ev.Type == termbox.EventResize {
				ui.HandleResize()
			}
		}
	}
}
