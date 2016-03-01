package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

var (
	// PidWhitelist contains the Pids whitelisted via the --pids option.
	PidWhitelist []uint64
)

func pidWhitelisted(pid uint64) bool {
	if len(PidWhitelist) == 0 {
		return true
	}
	for _, p := range PidWhitelist {
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

	NumCPUs  int
	MemTotal uint64
	PageSize uint64

	CPUTimeTotal uint64
	CPUTimeDiff  uint64
}

// NewMonitor returns an initialized Monitor.
func NewMonitor() *Monitor {
	m := &Monitor{
		Map:     make(map[uint64]*Process),
		NumCPUs: runtime.NumCPU(),
	}
	m.queryPageSize()
	m.parseMeminfoFile()
	return m
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
			continue // non-Pid directory
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
	case PidColumn.Title:
		sort.Sort(ByPid(m.List))
	case UserColumn.Title:
		sort.Sort(ByUser(m.List))
	case RSSColumn.Title, MemPercentColumn.Title:
		sort.Sort(ByRSS(m.List))
	case CPUPercentColumn.Title:
		sort.Sort(ByCPU(m.List))
	case CPUTimeColumn.Title:
		sort.Sort(ByTime(m.List))
	case CommandColumn.Title:
		sort.Sort(ByName(m.List))
	}
}

func (m *Monitor) addProcess(p *Process) {
	m.List = append(m.List, p)
	m.Map[p.Pid] = p
}

func (m *Monitor) removeDeadProcesses() {
	for i := len(m.List) - 1; i >= 0; i-- {
		p := m.List[i]

		if !p.Alive {
			m.List = append(m.List[:i], m.List[i+1:]...)
			delete(m.Map, p.Pid)
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

			// Only parsing the CPU jiffies for now, ignore rest of file.
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func (m *Monitor) parseMeminfoFile() {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			// As far as I know this value is always expressed in KB.
			// line = "MemTotal:       16371752 kB"
			memKBStr := strings.TrimPrefix(line, "MemTotal:")
			var memKB uint64
			_, err := fmt.Sscanf(memKBStr, "%d", &memKB)
			if err != nil {
				panic(err)
			}
			m.MemTotal = memKB * KB

			// Only parsing the MemTotal for now, ignore rest of file.
			break
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func (m *Monitor) queryPageSize() {
	out, err := exec.Command("getconf", "PAGESIZE").Output()
	if err != nil {
		panic(err)
	}

	pageSizeStr := strings.TrimSpace(string(out))
	m.PageSize, err = strconv.ParseUint(pageSizeStr, 10, 64)
	if err != nil {
		panic(err)
	}
}
