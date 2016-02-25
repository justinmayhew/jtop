package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var (
	// PIDWhitelist contains the PIDs whitelisted via the --pid option.
	PIDWhitelist []uint64
)

func pidWhitelisted(pid uint64) bool {
	if len(PIDWhitelist) == 0 {
		return true
	}
	for _, p := range PIDWhitelist {
		if p == pid {
			return true
		}
	}
	return false
}

// ProcessMonitor keeps tracks of the processes running on the system.
type ProcessMonitor struct {
	List []*Process
	Map  map[uint64]*Process

	NumCPUs      int
	CPUTimeTotal uint64
	CPUTimeDiff  uint64
}

// NewProcessMonitor returns an initialized ProcessMonitor ready for use.
func NewProcessMonitor() *ProcessMonitor {
	pm := &ProcessMonitor{}
	pm.Map = make(map[uint64]*Process)
	pm.NumCPUs = runtime.NumCPU()
	return pm
}

// Update updates the ProcessMonitor's state via the /proc filesystem.
func (pm *ProcessMonitor) Update() {
	lastCPUTimeTotal := pm.CPUTimeTotal
	pm.parseStatFile()
	pm.CPUTimeDiff = pm.CPUTimeTotal - lastCPUTimeTotal

	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		panic(err)
	}

	// Mark all processes as Dead
	for _, p := range pm.List {
		p.Alive = false
	}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		pid, err := strconv.ParseUint(file.Name(), 10, 64)
		if err != nil {
			continue // non-PID directory
		}

		if !pidWhitelisted(pid) {
			continue
		}

		if p, ok := pm.Map[pid]; ok {
			err := p.Update()
			if err == nil {
				p.Alive = true
			}
		} else {
			p := NewProcess(pid)
			if p != nil {
				p.Alive = true

				if !p.IsKernelThread() {
					pm.addProcess(p)
				}
			}
		}
	}

	pm.removeDeadProcesses()

	switch sortFlag {
	case "pid":
		sort.Sort(ByPID(pm.List))
	case "user":
		sort.Sort(ByUser(pm.List))
	case "cpu":
		sort.Sort(ByCPU(pm.List))
	case "time":
		sort.Sort(ByTime(pm.List))
	case "command":
		sort.Sort(ByName(pm.List))
	}

	// sanity check
	if len(pm.List) != len(pm.Map) {
		panic("list and map are not in sync")
	}
}

func (pm *ProcessMonitor) addProcess(p *Process) {
	pm.List = append(pm.List, p)
	pm.Map[p.PID] = p
}

func (pm *ProcessMonitor) removeDeadProcesses() {
	for i := len(pm.List) - 1; i >= 0; i-- {
		p := pm.List[i]

		if !p.Alive {
			pm.List = append(pm.List[:i], pm.List[i+1:]...)
			delete(pm.Map, p.PID)
		}
	}
}

func (pm *ProcessMonitor) parseStatFile() {
	file, err := os.Open("/proc/stat")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			pm.CPUTimeTotal = 0
			cpuTimeValues := strings.Split(line, " ")[2:] // skip "cpu" and ""
			for _, cpuTimeValue := range cpuTimeValues {
				value, err := strconv.ParseUint(cpuTimeValue, 10, 64)
				if err != nil {
					panic(err)
				}
				pm.CPUTimeTotal += value
			}

			// Only parsing the first line for now, ignore rest of file.
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
