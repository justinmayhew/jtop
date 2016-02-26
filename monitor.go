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
	// PIDWhitelist contains the PIDs whitelisted via the --pids option.
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

// Monitor monitors the processes and resource utilization of the system.
type Monitor struct {
	List []*Process
	Map  map[uint64]*Process

	NumCPUs      int
	CPUTimeTotal uint64
	CPUTimeDiff  uint64
}

// NewMonitor returns an initialized Monitor.
func NewMonitor() *Monitor {
	return &Monitor{
		Map:     make(map[uint64]*Process),
		NumCPUs: runtime.NumCPU(),
	}
}

// Update updates the Monitor state via the proc filesystem.
func (m *Monitor) Update() {
	lastCPUTimeTotal := m.CPUTimeTotal
	m.parseStatFile()
	m.CPUTimeDiff = m.CPUTimeTotal - lastCPUTimeTotal

	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		panic(err)
	}

	// Mark all processes as Dead
	for _, p := range m.List {
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

		if p, ok := m.Map[pid]; ok {
			err := p.Update()
			if err == nil {
				p.Alive = true
			}
		} else {
			p := NewProcess(pid)
			if p != nil {
				p.Alive = true

				if !p.IsKernelThread() {
					m.addProcess(p)
				}
			}
		}
	}

	m.removeDeadProcesses()

	switch sortFlag {
	case "pid":
		sort.Sort(ByPID(m.List))
	case "user":
		sort.Sort(ByUser(m.List))
	case "cpu":
		sort.Sort(ByCPU(m.List))
	case "time":
		sort.Sort(ByTime(m.List))
	case "command":
		sort.Sort(ByName(m.List))
	}
}

func (m *Monitor) addProcess(p *Process) {
	m.List = append(m.List, p)
	m.Map[p.PID] = p
}

func (m *Monitor) removeDeadProcesses() {
	for i := len(m.List) - 1; i >= 0; i-- {
		p := m.List[i]

		if !p.Alive {
			m.List = append(m.List[:i], m.List[i+1:]...)
			delete(m.Map, p.PID)
		}
	}
}

func (m *Monitor) parseStatFile() {
	file, err := os.Open("/proc/stat")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			m.CPUTimeTotal = 0
			cpuTimeValues := strings.Split(line, " ")[2:] // skip "cpu" and ""
			for _, cpuTimeValue := range cpuTimeValues {
				value, err := strconv.ParseUint(cpuTimeValue, 10, 64)
				if err != nil {
					panic(err)
				}
				m.CPUTimeTotal += value
			}

			// Only parsing the first line for now, ignore rest of file.
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
