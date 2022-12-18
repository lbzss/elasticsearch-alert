package alert

import "sync"

const defaultNumAttempts int = 3

type inventory struct {
	alerts map[string]int
	lock   *sync.RWMutex
}

func newInventory() *inventory {
	return &inventory{
		alerts: make(map[string]int),
		lock:   new(sync.RWMutex),
	}
}

func (i *inventory) register(id string) {
	i.lock.Lock()
	if _, ok := i.alerts[id]; ok {
		return
	}

	i.alerts[id] = defaultNumAttempts
	i.lock.Unlock()
}

func (i *inventory) deregister(id string) {
	i.lock.Lock()
	delete(i.alerts, id)
	i.lock.Unlock()
}

func (i *inventory) decrement(id string) {
	i.lock.Lock()
	defer i.lock.Unlock()
	if v, ok := i.alerts[id]; ok {
		i.alerts[id] = v - 1
	}
}

func (i *inventory) remaining(id string) int {
	i.lock.Lock()
	defer i.lock.Unlock()
	remaining := 0
	if v, ok := i.alerts[id]; ok {
		remaining = v
	}
	return remaining
}
