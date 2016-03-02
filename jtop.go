package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
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
  -s, --sort     sort by the specified column
  -u, --users    filter by User (comma-separated list)
      --verbose  show full command line with arguments
`

var (
	delayFlag   time.Duration
	pidsFlag    string
	sortFlag    string
	usersFlag   string
	verboseFlag bool
)

func exit(message string) {
	fmt.Fprintln(os.Stderr, message)
	flag.Usage()
	os.Exit(1)
}

func signalSelf(sig syscall.Signal) {
	if err := syscall.Kill(os.Getpid(), sig); err != nil {
		panic(err)
	}
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
	for _, column := range Columns {
		if sortFlag == column.Title {
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
	defaultDelay := time.Duration(1500 * time.Millisecond)
	flag.DurationVar(&delayFlag, "d", defaultDelay, "")
	flag.DurationVar(&delayFlag, "delay", defaultDelay, "")

	flag.StringVar(&pidsFlag, "p", "", "")
	flag.StringVar(&pidsFlag, "pids", "", "")

	defaultSort := CPUPercentColumn.Title
	flag.StringVar(&sortFlag, "s", defaultSort, "")
	flag.StringVar(&sortFlag, "sort", defaultSort, "")

	flag.StringVar(&usersFlag, "u", "", "")
	flag.StringVar(&usersFlag, "users", "", "")

	flag.BoolVar(&verboseFlag, "verbose", false, "")

	flag.Usage = func() {
		fmt.Fprint(os.Stdout, usage)
	}
}

func termboxInit() {
	if err := termbox.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func main() {
	flag.Parse()
	validateFlags()

	termboxInit()
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
				case ev.Ch == 'q' || ev.Key == termbox.KeyCtrlC:
					return
				case ev.Ch == 'j' || ev.Key == termbox.KeyArrowDown:
					ui.HandleDown()
				case ev.Ch == 'k' || ev.Key == termbox.KeyArrowUp:
					ui.HandleUp()
				case ev.Ch == 'v':
					verboseFlag = !verboseFlag
				case ev.Key == termbox.KeyCtrlD:
					ui.HandleCtrlD()
				case ev.Key == termbox.KeyCtrlU:
					ui.HandleCtrlU()
				case ev.Key == termbox.KeyCtrlZ:
					termbox.Close()
					signalSelf(syscall.SIGTSTP)
					termboxInit()
				}
			} else if ev.Type == termbox.EventResize {
				ui.HandleResize()
			}
		}
	}
}
