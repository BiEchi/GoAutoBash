package queue

import (
	"gopkg.in/go-playground/webhooks.v5/github"
	"sync"
	"time"
)

// Task struct prototype for the tasks pushed onto GitHub */
type Task struct {
	Name 		string
	IsManual 	bool
	ManualList	[]string
	Payload    	*github.PushPayload
}

// Status of the queueing system
type Status struct {
	Running 	bool
	LastRun 	time.Time
	Error   	error
}

// Queue is the queue of Task submitted
var Queue chan *Task

// statusMap is what?
var statusMap map[string]*Status

// mutex is a local variable used to perform atomic operation on R/W logic
var mutex sync.RWMutex

// GetStatus makes a copy of Status in the memory and return the address
func GetStatus() *map[string]Status {
	statusCopy := make(map[string]Status)
	mutex.RLock()
	for key, value := range statusMap {
		statusCopy[key] = *value
	}
	mutex.RUnlock()
	return &statusCopy
}
