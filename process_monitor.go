package main

import (
	"io/ioutil"
	"sort"
	"strconv"
)

// ProcessMonitor keeps tracks of the processes running on the system.
type ProcessMonitor struct {
	List []*Process
	Map  map[int]*Process
}

func NewProcessMonitor() *ProcessMonitor {
	pm := &ProcessMonitor{}
	pm.Map = make(map[int]*Process)
	return pm
}

// Update updates the ProcessMonitor's state via the /proc filesystem.
func (pm *ProcessMonitor) Update() {
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

		pid, err := strconv.Atoi(file.Name())
		if err != nil {
			continue // non-PID directory
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

	sort.Sort(ByPid(pm.List))

	// sanity check
	if len(pm.List) != len(pm.Map) {
		panic("list and map are not in sync")
	}
}

func (pm *ProcessMonitor) addProcess(p *Process) {
	pm.List = append(pm.List, p)
	pm.Map[p.Pid] = p
}

func (pm *ProcessMonitor) removeDeadProcesses() {
	for i := len(pm.List) - 1; i >= 0; i-- {
		p := pm.List[i]

		if !p.Alive {
			pm.List = append(pm.List[:i], pm.List[i+1:]...)
			delete(pm.Map, p.Pid)
		}
	}
}
