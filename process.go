package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	// The indices of the values in /proc/<pid>/stat
	statPidIdx = iota
	statCommIdx
	statStateIdx
	statPpidIdx
	statPgrpIdx
	statSessionIdx
	statTtyNrIdx
	statTpgidIdx
	statFlagsIdx
	statMinfltIdx
	statCminfltIdx
	statMajfltIdx
	statCmajfltIdx
	statUtimeIdx
	statStimeIdx
	statCutimeIdx
	statCstimeIdx
	statPriorityIdx
	statNiceIdx
	statNumThreadsIdx
	statItrealvalueIdx
	statStartTimeIdx
	statVsizeIdx
	statRssIdx
	statRsslimIdx
	statStartCodeIdx
	statEndCodeIdx
	statStartStackIdx
	statKstKespIdx
	statKstKeipIdx
	statSignalIdx
	statBlockedIdx
	statSigIgnoreIdx
	statSigCatchIdx
	statWchanIdx
	statNswapIdx
	statCnswapIdx
	statExitSignalIdx
	statProcessorIdx
	statRtPriorityIdx
	statPolicyIdx
	statDelayActBlkioTicksIdx
	statGuestTimeIdx
	statCguestTimeIdx
)

// Process represents an operating system process.
type Process struct {
	Pid     int
	User    *user.User
	Command string

	// Alive is a flag used by ProcessMonitor to determine if it should remove
	// this process.
	Alive bool

	// Data from /proc/<pid>/stat
	Pgrp  int
	Utime uint64
	Stime uint64

	UtimeDiff uint64
	StimeDiff uint64
}

// NewProcess returns a new Process if a process is currently running on
// the system with the passed in Pid.
func NewProcess(pid int) *Process {
	p := &Process{
		Pid: pid,
	}

	if err := p.parseCmdlineFile(); err != nil {
		return nil
	}

	if err := p.Update(); err != nil {
		return nil
	}

	return p
}

// Update updates the Process from various files in /proc/<pid>. It returns an
// error if the process was unable to be updated (probably because the process
// is no longer running).
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

// statProcDir updates p with any information it needs from statting /proc/<pid>.
func (p *Process) statProcDir() error {
	path := filepath.Join("/proc", strconv.Itoa(p.Pid))

	var stat syscall.Stat_t
	if err := syscall.Stat(path, &stat); err != nil {
		return err
	}

	user, err := UserByUID(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		return err
	}
	p.User = user

	return nil
}

// parseStatFile updates p with any information it needs from /proc/<pid>/stat.
func (p *Process) parseStatFile() error {
	path := filepath.Join("/proc", strconv.Itoa(p.Pid), "stat")

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	line := string(data)
	values := strings.Split(line, " ")

	p.Pgrp, err = strconv.Atoi(values[statPgrpIdx])
	if err != nil {
		panic(err)
	}

	lastUtime := p.Utime
	p.Utime, err = strconv.ParseUint(values[statUtimeIdx], 10, 64)
	if err != nil {
		panic(err)
	}
	p.UtimeDiff = p.Utime - lastUtime

	lastStime := p.Stime
	p.Stime, err = strconv.ParseUint(values[statStimeIdx], 10, 64)
	if err != nil {
		panic(err)
	}
	p.StimeDiff = p.Stime - lastStime

	return nil
}

// parseCmdlineFile sets p's Command via /proc/<pid>/cmdline.
func (p *Process) parseCmdlineFile() error {
	path := filepath.Join("/proc", strconv.Itoa(p.Pid), "cmdline")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	s := string(data)
	p.Command = strings.TrimSpace(strings.Replace(s, "\x00", " ", -1))
	return nil
}

// ByPid implements sort.Interface.
type ByPid []*Process

func (p ByPid) Len() int           { return len(p) }
func (p ByPid) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPid) Less(i, j int) bool { return p[i].Pid < p[j].Pid }

// ByCPUTimeDiff implements sort.Interface.
type ByCPUTimeDiff []*Process

func (p ByCPUTimeDiff) Len() int      { return len(p) }
func (p ByCPUTimeDiff) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p ByCPUTimeDiff) Less(i, j int) bool {
	p1, p2 := p[i], p[j]
	p1Total := p1.UtimeDiff + p1.StimeDiff
	p2Total := p2.UtimeDiff + p2.StimeDiff
	if p1Total == p2Total {
		// Fall back to sorting by PID
		return p1.Pid < p2.Pid
	}
	return p1Total > p2Total
}
