package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcessType int

const (
	ProcessUser ProcessType = iota
	ProcessKernel
)

var (
	users map[int]string
)

// Process represents a process discovered in /proc.
type Process struct {
	Pid     int
	User    string
	Command string
	Type    ProcessType
}

// ByPid implements sort.Interface for []Process based on the Pid field.
type ByPid []Process

func (p ByPid) Len() int           { return len(p) }
func (p ByPid) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ByPid) Less(i, j int) bool { return p[i].Pid < p[j].Pid }

func init() {
	users = make(map[int]string)
	path := filepath.Join("/etc", "passwd")

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// justin:x:1000:1000:Justin,,,:/home/justin:/bin/zsh
		line := scanner.Text()
		pieces := strings.Split(line, ":")
		uid, err := strconv.Atoi(pieces[2])
		if err != nil {
			panic(err)
		}

		username := pieces[0]
		users[uid] = username
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

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
			username := user(pid)
			processes = append(processes, Process{
				Pid:     pid,
				User:    username,
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

// user returns the effective user running process `pid`.
func user(pid int) string {
	var uid int
	path := filepath.Join("/proc", strconv.Itoa(pid), "status")

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}

		//       R     E     SS    FS
		// Uid:\t1000\t1000\t1000\t1000

		pieces := strings.Split(line, "\t")
		uid, err = strconv.Atoi(pieces[2])
		if err != nil {
			panic(err)
		}
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return users[uid]
}
