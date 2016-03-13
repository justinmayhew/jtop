package main

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
)

const (
	// The values in /proc/<pid>/stat
	statState = iota
	statPpid
	statPgrp
	statSession
	statTtyNr
	statTpgid
	statFlags
	statMinflt
	statCminflt
	statMajflt
	statCmajflt
	statUtime
	statStime
	statCutime
	statCstime
	statPriority
	statNice
	statNumThreads
	statItrealvalue
	statStartTime
	statVsize
	statRSS
	statRSSLimit
	statStartCode
	statEndCode
	statStartStack
	statKstKesp
	statKstKeip
	statSignal
	statBlocked
	statSigIgnore
	statSigCatch
	statWchan
	statNswap
	statCnswap
	statExitSignal
	statProcessor
	statRtPriority
	statPolicy
	statDelayActBlkioTicks
	statGuestTime
	statCguestTime
)

// Process represents an operating system process.
type Process struct {
	Pid     uint64
	User    *user.User
	Name    string // foo
	Command string // /usr/bin/foo --args

	// Alive is a flag used by Monitor to determine if it should remove
	// this process.
	Alive bool

	// Tree view
	Parent      *Process
	Children    []*Process
	TreePrefix  string
	isLastChild bool

	// Data from /proc/<pid>/stat
	State byte
	Ppid  uint64
	Pgrp  uint64
	Utime uint64
	Stime uint64
	RSS   uint64

	UtimeDiff uint64
	StimeDiff uint64

	initializing bool
}

// NewProcess returns a new Process if a process is currently running on
// the system with the passed in Pid.
func NewProcess(pid uint64) *Process {
	p := &Process{
		Pid:          pid,
		initializing: true,
	}

	if err := p.Update(); err != nil {
		return nil
	}

	if !p.hasEmptyCmdlineFile() {
		if err := p.parseCmdlineFile(); err != nil {
			return nil
		}
	}

	p.initializing = false
	return p
}

func (p *Process) String() string {
	return fmt.Sprintf("%d (%s)", p.Pid, p.Name)
}

// Update updates Process from various files in /proc/<pid>. It returns an
// error if Process was unable to be updated (probably because the actual OS
// process is no longer running).
func (p *Process) Update() error {
	if err := p.statProcDir(); err != nil {
		return err
	}

	if err := p.parseStatFile(); err != nil {
		return err
	}

	return nil
}

// IsKernelThread returns whether or not Process is a kernel thread.
func (p *Process) IsKernelThread() bool {
	return p.Pgrp == 0
}

// TreeList returns a Process slice in "tree order" such that iterating
// over it and printing out the TreePrefix and Command will display a
// nice overview of the process hierarchy.
func (p *Process) TreeList(level uint) []*Process {
	const defaultEnd = "├─ "
	const lastChildEnd = "└─ "
	const defaultSegment = "│  "
	const lastChildSegment = "   "

	end := defaultEnd
	if p.isLastChild {
		end = lastChildEnd
	}

	switch level {
	case 0:
		p.TreePrefix = ""
	case 1:
		p.TreePrefix = end
	default:
		p.TreePrefix = ""
		for parent := p.Parent; parent != nil && parent.Pid != InitPid; parent = parent.Parent {
			if parent.isLastChild {
				p.TreePrefix = lastChildSegment + p.TreePrefix
			} else {
				p.TreePrefix = defaultSegment + p.TreePrefix
			}
		}
		p.TreePrefix = p.TreePrefix + end
	}

	var treeList []*Process
	treeList = append(treeList, p)
	for i, process := range p.Children {
		if i == len(p.Children)-1 {
			process.isLastChild = true
		} else {
			process.isLastChild = false
		}
		treeList = append(treeList, process.TreeList(level+1)...)
	}
	return treeList
}

func (p *Process) statProcDir() error {
	path := fmt.Sprintf("/proc/%d", p.Pid)

	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return err
	}

	user, err := UserByUid(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		return err
	}
	p.User = user

	return nil
}

func (p *Process) parseStatFile() error {
	path := fmt.Sprintf("/proc/%d/stat", p.Pid)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	line := string(data)

	commStart := strings.IndexByte(line, '(') + 1
	commEnd := strings.LastIndexByte(line, ')')

	if p.hasEmptyCmdlineFile() {
		p.Command = line[commStart:commEnd]
		p.Name = p.Command
	}

	values := strings.Split(line[commEnd+2:], " ")

	// One character from the string "RSDZTW" where R
	// is running, S is sleeping in an interruptible wait,
	// D is waiting in uninterruptible disk sleep, Z is
	// zombie, T is traced or stopped (on a signal), and W
	// is paging.
	p.State = values[statState][0]

	p.Ppid = MustParseUint64(values[statPpid])

	p.Pgrp = MustParseUint64(values[statPgrp])

	lastUtime := p.Utime
	p.Utime = MustParseUint64(values[statUtime])
	p.UtimeDiff = p.Utime - lastUtime

	lastStime := p.Stime
	p.Stime = MustParseUint64(values[statStime])
	p.StimeDiff = p.Stime - lastStime

	p.RSS = MustParseUint64(values[statRSS])

	// The state will only be running if it's running at the exact
	// moment this file was read. That's probably not what the
	// average user wants, even though it's what top and htop do.
	// Set it to be running if it's used any CPU since the last update.
	if !p.initializing && p.State == 'S' && (p.UtimeDiff > 0 || p.StimeDiff > 0) {
		p.State = 'R'
	}

	return nil
}

func (p *Process) hasEmptyCmdlineFile() bool {
	return p.IsKernelThread() || p.State == 'Z'
}

func (p *Process) parseCmdlineFile() error {
	path := fmt.Sprintf("/proc/%d/cmdline", p.Pid)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	cmdline := string(data)
	p.Command = strings.TrimSpace(strings.Replace(cmdline, "\x00", " ", -1))
	p.Name = commandToName(p.Command)
	return nil
}

// commandToName takes a string in a format like "/usr/bin/foo --arguments"
// and returns its base name without arguments, "foo".
func commandToName(cmdline string) string {
	command := strings.Split(cmdline, " ")[0]
	if strings.HasSuffix(command, ":") {
		// For processes that set their name in a format like
		// "postgres: writer process" the value is returned as is.
		return cmdline
	}
	return path.Base(command)
}

type ByPid []*Process

func (p ByPid) Len() int      { return len(p) }
func (p ByPid) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByPid) Less(i, j int) bool {
	return p[i].Pid < p[j].Pid
}

type ByUser []*Process

func (p ByUser) Len() int      { return len(p) }
func (p ByUser) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByUser) Less(i, j int) bool {
	return p[i].User.Username < p[j].User.Username
}

type ByRSS []*Process

func (p ByRSS) Len() int      { return len(p) }
func (p ByRSS) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByRSS) Less(i, j int) bool {
	return p[i].RSS > p[j].RSS
}

type ByCPU []*Process

func (p ByCPU) Len() int      { return len(p) }
func (p ByCPU) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByCPU) Less(i, j int) bool {
	p1, p2 := p[i], p[j]
	p1Diff := p1.UtimeDiff + p1.StimeDiff
	p2Diff := p2.UtimeDiff + p2.StimeDiff
	if p1Diff == p2Diff {
		return p1.Pid < p2.Pid
	}
	return p1Diff > p2Diff
}

type ByTime []*Process

func (p ByTime) Len() int      { return len(p) }
func (p ByTime) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByTime) Less(i, j int) bool {
	p1, p2 := p[i], p[j]
	p1Total := p1.Utime + p1.Stime
	p2Total := p2.Utime + p2.Stime
	if p1Total == p2Total {
		return p1.Pid < p2.Pid
	}
	return p1Total > p2Total
}

type ByState []*Process

func (p ByState) Len() int      { return len(p) }
func (p ByState) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByState) Less(i, j int) bool {
	p1, p2 := p[i], p[j]
	if p1.State == p2.State {
		return p1.Pid < p2.Pid
	}
	return p1.State < p2.State
}

type ByName []*Process

func (p ByName) Len() int      { return len(p) }
func (p ByName) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByName) Less(i, j int) bool {
	return p[i].Name < p[j].Name
}
