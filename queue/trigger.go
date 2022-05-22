package queue

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"
	"os"
	"os/exec"
	"time"
)

// StartQueue utilizes the producer-consumer model for the queueing system
func StartQueue(consumerCount int, chanSize int, waitTime time.Duration) error {
	Queue = make(chan *Task, chanSize)
	// statusMap is used for storing a series of status using a key(name) and value
	statusMap = make(map[string]*Status)
	for i := 1; i <= consumerCount; i++ {
		// use go func to start new threads
		go func(id int) {
			logrus.Infof("Consumer %d started", id)
			for task := range Queue {
				shouldRun := true
				var lastRun time.Time
				mutex.Lock()
				status, ok := statusMap[task.Name]
				if !ok {
					// no status with this name is available; initialize new status
					now := time.Now()
					statusMap[task.Name] = &Status{
						Running: true,
						LastRun: now,
					}
					shouldRun = true
				} else {
					if status.Running {
						shouldRun = true // this forces running even though same user has another task running
					} else {
						if task.IsManual || time.Now().Sub(status.LastRun) >= waitTime {
							shouldRun = true
						} else {
							shouldRun = true
						}
					}
				}
				lastRun = statusMap[task.Name].LastRun
				statusMap[task.Name].Error = nil
				if shouldRun {
					statusMap[task.Name].LastRun = time.Now()
					statusMap[task.Name].Running = true
				}
				mutex.Unlock()

				if !shouldRun {
					// grader is running for current tasks, we ignore that grading request
					logrus.Warnf("grader is running/waiting %s, lastRun = %+v, we ignore that grading request", task.Name, lastRun)
					continue
				}

				logrus.Info("Consumer ", id, " is executing next task...")
				var err error
				if err = ExecuteTask(task); err != nil {
					logrus.Error(err)
				}

				mutex.Lock()
				statusMap[task.Name].Running = false
				statusMap[task.Name].Error = err
				mutex.Unlock()
			}
		}(i) /**/
	}

	return nil
}

// ExecuteTask is the function to execute whatever you want to trigger after an event occurs!
func ExecuteTask(task *Task) error {
	/* global configs */
	numMP := "1"
	commitId := task.Payload.HeadCommit.ID[:4]
	/* the cache dir is used to store the commit content */
	dt := time.Now()
	dir := "report" + "/" + task.Payload.Pusher.Name + "/MP" + numMP + "-" + commitId + "_" + dt.Format("2016-01-02 15:04:05")

	/* clone the commit to local for later use */
	PAT, errRead := os.ReadFile("./queue/PAT.txt")
	if errRead != nil {
		return errRead
	}
	cmdClone := exec.Command("git", "clone", "https://haob2:"+string(PAT)+"@"+task.Payload.Repository.CloneURL[8:], dir)
	outputClone, errClone := cmdClone.Output()
	if errClone != nil {
		logrus.Error(errClone, string(outputClone))
		return errClone
	} else {
		logrus.Info("Cloned ", task.Payload.Pusher.Name+"/"+task.Payload.HeadCommit.ID)
	}

	/* dispatch other tasks to external bash program */
	cmdBash := exec.Command("bash", "mp"+numMP+".sh")
	outputBash, errBash := cmdBash.Output()
	if errBash != nil {
		logrus.Error(errBash, string(outputBash))
		return errBash
	}

	/* generate a README.md */
	execCommand(dir, "touch", "README.md")
	/* push the generated dir to another branch on GitHub */
	execCommand(dir, "git", "branch", "report")
	execCommand(dir, "git", "checkout", "report")
	execCommand(dir, "git", "add", ".")
	execCommand(dir, "git", "commit", "-m", "Report Generated.")
	execCommand(dir, "git", "push", "origin", "report")

	return nil
}

// TaskEnqueue enqueues a queue.Task object to the queueing system
func TaskEnqueue(payload *github.PushPayload) error {
	name := payload.Pusher.Name
	Queue <- &Task{
		Name:     name,
		IsManual: false,
		Payload:  payload,
	}
	return nil
}

/* wrapper function for exec.Command calls with dir and error check (logrus) */
func execCommand(dir string, name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	output, err := cmd.Output()
	logrus.Info(string(output))
	if err != nil {
		logrus.Error(err, string(output)+": first argument is "+arg[0])
		return output, err
	}
	return output, nil
}
