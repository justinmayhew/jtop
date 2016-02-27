package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

const usage = `Usage: jtop [options]

Options:
  -d, --delay    delay between updates
  -p, --pids     filter by PID (comma-separated list)
  -s, --sort     sort by the specified column (%s)
  -u, --users    filter by User (comma-separated list)
      --verbose  show full command line with arguments
`

const (
	defaultSortColumn  = "cpu"
	defaultUpdateDelay = time.Duration(1500 * time.Millisecond)
)

const (
	PidColumn        = "pid"
	UserColumn       = "user"
	RSSColumn        = "rss"
	MemPercentColumn = "mem"
	CPUPercentColumn = "cpu"
	CPUTimeColumn    = "time"
	CommandColumn    = "command"
)

var (
	delayFlag   time.Duration
	pidsFlag    string
	sortFlag    string
	usersFlag   string
	verboseFlag bool

	sortColumns = []string{
		PidColumn,
		UserColumn,
		RSSColumn,
		MemPercentColumn,
		CPUPercentColumn,
		CPUTimeColumn,
		CommandColumn,
	}
)

func exit(message string) {
	fmt.Fprintln(os.Stderr, message)
	flag.Usage()
	os.Exit(1)
}

func validateDelayFlag() {
	if delayFlag <= 0 {
		exit("flag error: delay must be positive")
	}
}

func validatePidsFlag() {
	if pidsFlag == "" {
		return
	}

	pids := strings.Split(pidsFlag, ",")
	for _, value := range pids {
		if pid, err := strconv.ParseUint(value, 10, 64); err != nil {
			message := fmt.Sprintf("flag error: %s is not a valid PID", value)
			exit(message)
		} else {
			PidWhitelist = append(PidWhitelist, pid)
		}
	}
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
	validateDelayFlag()
	validatePidsFlag()
	validateSortFlag()
	validateUsersFlag()
}

func init() {
	flag.DurationVar(&delayFlag, "d", defaultUpdateDelay, "")
	flag.DurationVar(&delayFlag, "delay", defaultUpdateDelay, "")

	flag.StringVar(&pidsFlag, "p", "", "")
	flag.StringVar(&pidsFlag, "pids", "", "")

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

	ticker := time.Tick(delayFlag)
	monitor := NewMonitor()
	monitor.Update()
	ui := NewUI(monitor)

	for {
		ui.Draw()

		select {
		case <-ticker:
			monitor.Update()

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
