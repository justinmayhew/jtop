package main

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessType int

const (
	ProcessUser ProcessType = iota
	ProcessKernel
)

// Process represents a process discovered in /proc.
type Process struct {
	Pid     int
	Command string
	Type    ProcessType
}

// ByPid implements sort.Interface for []Process based on the Pid field.
type ByPid []Process

func (p ByPid) Len() int           { return len(p) }
func (p ByPid) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPid) Less(i, j int) bool { return p[i].Pid < p[j].Pid }

func getRunningProcesses() []Process {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		panic(err)
	}

	var processes []Process

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue // non-PID directory
		}

		command := cmdline(pid)
		t := ProcessUser
		if command == "" {
			t = ProcessKernel
		}

		// Skip kernel processes for now
		if t == ProcessUser {
			processes = append(processes, Process{
				Pid:     pid,
				Command: command,
				Type:    t,
			})
		}
	}

	return processes
}

// cmdline returns the command used to start `pid`.
func cmdline(pid int) string {
	path := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")

	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	s := string(data)

	// Sometimes the arguments are separated by NUL as well as ending in multiple
	// trailing NULs. Fix that so we return something that looks like you'd type
	// in the shell.
	return strings.TrimSpace(strings.Replace(s, "\x00", " ", -1))
}
