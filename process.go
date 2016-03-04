package main

import (
	"io/ioutil"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
)

const (
	// The values in /proc/<pid>/stat
	statPid = iota
	statCommand
	statState
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

	// Data from /proc/<pid>/stat
	Pgrp  uint64
	Utime uint64
	Stime uint64
	RSS   uint64

	UtimeDiff uint64
	StimeDiff uint64
}

// NewProcess returns a new Process if a process is currently running on
// the system with the passed in Pid.
func NewProcess(pid uint64) *Process {
	p := &Process{
		Pid: pid,
	}

	if err := p.Update(); err != nil {
		return nil
	}

	if !p.IsKernelThread() {
		if err := p.parseCmdlineFile(); err != nil {
			return nil
		}
	}

	return p
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

func (p *Process) statProcDir() error {
	path := path.Join("/proc", strconv.FormatUint(p.Pid, 10))

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
	path := path.Join("/proc", strconv.FormatUint(p.Pid, 10), "stat")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	line := string(data)
	values := strings.Split(line, " ")

	p.Pgrp, err = strconv.ParseUint(values[statPgrp], 10, 64)
	if err != nil {
		panic(err)
	}

	if p.IsKernelThread() {
		// Kernel threads have an empty cmdline file.
		command := values[statCommand]
		command = command[1 : len(command)-1] // strip '(' and ')'
		p.Command = command
		p.Name = command
	}

	lastUtime := p.Utime
	p.Utime, err = strconv.ParseUint(values[statUtime], 10, 64)
	if err != nil {
		panic(err)
	}
	p.UtimeDiff = p.Utime - lastUtime

	lastStime := p.Stime
	p.Stime, err = strconv.ParseUint(values[statStime], 10, 64)
	if err != nil {
		panic(err)
	}
	p.StimeDiff = p.Stime - lastStime

	p.RSS, err = strconv.ParseUint(values[statRSS], 10, 64)
	if err != nil {
		panic(err)
	}

	return nil
}

func (p *Process) parseCmdlineFile() error {
	path := path.Join("/proc", strconv.FormatUint(p.Pid, 10), "cmdline")

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

// ByPid sorts by Pid.
type ByPid []*Process

func (p ByPid) Len() int      { return len(p) }
func (p ByPid) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByPid) Less(i, j int) bool {
	return p[i].Pid < p[j].Pid
}

// ByUser sorts by the username of the processes user.
type ByUser []*Process

func (p ByUser) Len() int      { return len(p) }
func (p ByUser) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByUser) Less(i, j int) bool {
	return p[i].User.Username < p[j].User.Username
}

// ByRSS sorts by resident set size.
type ByRSS []*Process

func (p ByRSS) Len() int      { return len(p) }
func (p ByRSS) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByRSS) Less(i, j int) bool {
	return p[i].RSS > p[j].RSS
}

// ByCPU sorts by the amount of CPU time used since the last update.
type ByCPU []*Process

func (p ByCPU) Len() int      { return len(p) }
func (p ByCPU) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByCPU) Less(i, j int) bool {
	p1, p2 := p[i], p[j]
	p1Total := p1.UtimeDiff + p1.StimeDiff
	p2Total := p2.UtimeDiff + p2.StimeDiff
	if p1Total == p2Total {
		return p1.Pid < p2.Pid
	}
	return p1Total > p2Total
}

// ByTime sorts by the amount of CPU time used total.
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

// ByName sorts by Name.
type ByName []*Process

func (p ByName) Len() int      { return len(p) }
func (p ByName) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByName) Less(i, j int) bool {
	return p[i].Name < p[j].Name
}
