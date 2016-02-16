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

// Process represents an operating system process.
type Process struct {
	Pid     int
	User    *user.User
	Command string

	// Alive is a flag used by ProcessMonitor to determine if it should remove
	// this process.
	Alive bool

	// Data from proc/<pid>/stat
	Pgrp  int
	Utime uint64
	Stime uint64
}

// NewProcess returns a new Process if a process is currently running on
// the system with the passed in Pid.
func NewProcess(pid int) *Process {
	p := &Process{
		Pid: pid,
	}

	if err := p.ParseCmdlineFile(); err != nil {
		return nil
	}

	if err := p.Update(); err != nil {
		return nil
	}

	return p
}

func (p *Process) Update() error {
	if err := p.StatProcDir(); err != nil {
		return err
	}

	if err := p.ParseStatFile(); err != nil {
		return err
	}

	return nil
}

// StatProcDir updates p with any information it needs from statting /proc/<pid>.
func (p *Process) StatProcDir() error {
	path := filepath.Join("/proc", strconv.Itoa(p.Pid))

	var stat syscall.Stat_t
	err := syscall.Stat(path, &stat)
	if err != nil {
		return err
	}

	user, err := userByUid(strconv.FormatUint(uint64(stat.Uid), 10))
	if err != nil {
		panic(err)
	}
	p.User = user

	return nil
}

// ParseStatFile updates p with any information it needs from /proc/<pid>/stat.
func (p *Process) ParseStatFile() error {
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

	pgrp, err := strconv.Atoi(values[4])
	if err != nil {
		panic(err)
	}
	p.Pgrp = pgrp

	utime, err := strconv.Atoi(values[13])
	if err != nil {
		panic(err)
	}
	p.Utime = uint64(utime)

	stime, err := strconv.Atoi(values[14])
	if err != nil {
		panic(err)
	}
	p.Stime = uint64(stime)

	return nil
}

// ParseCmdlineFile sets p's Command via /proc/<pid>/cmdline.
func (p *Process) ParseCmdlineFile() error {
	path := filepath.Join("/proc", strconv.Itoa(p.Pid), "cmdline")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	s := string(data)
	p.Command = strings.TrimSpace(strings.Replace(s, "\x00", " ", -1))
	return nil
}

func (p *Process) IsKernelThread() bool {
	return p.Pgrp == 0
}

type ByPid []*Process

func (p ByPid) Len() int           { return len(p) }
func (p ByPid) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPid) Less(i, j int) bool { return p[i].Pid < p[j].Pid }
