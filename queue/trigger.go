package queue

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	numMP := "2"
	commitId := task.Payload.HeadCommit.ID[:6]
	/* the cache dir is used to store the commit content */
	dt := time.Now()
	dir := "report" + "/" + task.Payload.Pusher.Name + "/MP" + numMP + "_" + commitId + "_" + dt.Format("01-02_15-04-05")

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

		/* delete the github hook for the subdir */
		execCommand(dir, "rm", "-rf", ".git")
		/* extract the MP source file to the report subdir */
		execCommand(dir, "mkdir", "report")
		execCommand(dir, "cp", "mp/mp"+numMP+"/mp"+numMP+".asm", "report/student.asm")
		/* copy all the files to the report generation dir, must copy one by one */
		/* MP2 */
		execCommand(dir, "cp", "../../../mp"+numMP+"/gold.asm", "report")
		execCommand(dir, "cp", "../../../mp"+numMP+"/mem_alloc.asm", "report")
		execCommand(dir, "cp", "../../../mp"+numMP+"/test_data.asm", "report")
		execCommand(dir, "cp", "../../../mp"+numMP+"/replay.sh", "report")

		/* MP3 */
		// execCommand(dir, "cp", "../../../mp"+numMP+"/extra.asm", "report")
		// execCommand(dir, "cp", "../../../mp"+numMP+"/gold.asm", "report")
		// execCommand(dir, "cp", "../../../mp"+numMP+"/replay.sh", "report")
		// execCommand(dir, "cp", "../../../mp"+numMP+"/sched_alloc_.asm", "report")
		// execCommand(dir, "cp", "../../../mp"+numMP+"/sched.asm", "report")
		// execCommand(dir, "cp", "../../../mp"+numMP+"/stack_alloc_.asm", "report")
	}

	/* allow the container to write to the host machine */
	execCommand(dir, "chmod", "0777", "report")
	/* run the klc-3 regression test */
	if MPExists("report"+"/"+task.Payload.Pusher.Name, numMP, dir) {
		/* mkdir the regression dir */
		execCommand(dir, "mkdir", "report/regression")
		/* make a copy of the regression testcases to the report/regression dir */
		execCommand(dir, "cp", "report/student.asm", "report/regression/student.asm")
		execCommand(dir, "cp", "report/gold.asm", "report/regression/gold.asm")
		execCommand(dir, "cp", "report/replay.sh", "report/regression/replay.sh")
		execCommand(dir, "cp", "report/mem_alloc.sh", "report/regression/mem_alloc.sh")
		/* we have previous run history, add the commits to list regTestList */
		var regTestString string
		fDirs, _ := ioutil.ReadDir("report" + "/" + task.Payload.Pusher.Name)
		i := 0
		for _, fDir := range fDirs {
			/* add to regression test when has prefix but not this dir */
			if strings.HasPrefix(fDir.Name(), "MP"+numMP) && fDir.Name() != "MP"+numMP+"_"+commitId+"_"+dt.Format("01-02_15-04-05") {
				i += 1
				/* add the dir to list regTestList */
				regTestString += " report/regression/" + strconv.Itoa(i) + ".asm"
				/* copy the testcase files to dir/report */
				execCommand(".", "cp", "report/"+task.Payload.Pusher.Name+"/"+fDir.Name()+"/report/klc3-out-0/test0/test0-test_data.asm",
					dir+"/report/regression/"+strconv.Itoa(i)+".asm")
			}
		}
		/* allow the container to write to the host machine */
		execCommand(dir, "chmod", "0777", "report/regression")
		/* run the regression test on all previous testcases */
		/* split regTestString to regTestList with splitter " " */
		regTestList := strings.Split(regTestString[1:], " ")

		for _, regTest := range regTestList {
			println(regTest)
			/* BUG: run the regression test */
			execCommand(".", "docker", "run", "-P", "-v=/root/GoAutoBash/"+dir+"/report/regression:/home/klee/report/regression:Z", "liuzikai/klc3",
				"klc3", "--test=report/regression/student.asm", "--gold=report/regression/gold.asm", "--use-forked-solver=false",
				"--copy-additional-file=report/regression/replay.sh", "--max-lc3-step-count=200000", "--max-lc3-out-length=1100",
				regTest, "report/regression/mem_alloc.sh")
		}
		execCommand(dir, "rm", "report/regression/gold.asm")
		execCommand(dir, "rm", "report/regression/student.asm")
	}

	/* run the klc-3 main test */
	execCommand(".", "docker", "run", "-P", "-v=/root/GoAutoBash/"+dir+"/report:/home/klee/report:Z", "liuzikai/klc3",
		"klc3", "--test=report/student.asm", "--gold=report/gold.asm", "--use-forked-solver=false",
		"--copy-additional-file=report/replay.sh", "--max-lc3-step-count=200000", "--max-lc3-out-length=1100",
		/* MP2 */
		"report/mem_alloc.asm", "report/test_data.asm")
	/* MP3 */
	/* "report/sched_alloc_.asm", "report/stack_alloc_.asm", "report/sched.asm", "report/extra.asm"*/

	/* append disclaimer to the report markdown */
	append(dir+"/report/klc3-out-0/report.md", "### KLC3 DISCLAIMER\n\t\n\tKLC3 feedback tool first runs the tests distributed with mp2 to you, reported in section [Easy Test](#easy-test).\n\tIf you pass all these tests, KLC3 starts symbolic execution ([what is this?](https://en.wikipedia.org/wiki/Symbolic_execution)\n\ton your code trying to find any input (test case) to trigger your bugs. When a bug is detected, a test case will be provided to you.\n\t\n\tWe want you to resolve bugs detected before KLC3\n\truns time-consuming symbolic execution again, so on your next commit, KLC3 will runs all test cases previously provided\n\tto you, in the section [Regression Test](#regression-test). If they are all passed, KLC3 will try to find new test\n\tcases that can trigger bugs in your code, in the section [Report](#report).\n\t\n\tKLC3 is still under test. This report can be **incorrect** or even **misleading**. If you think there is\n\tsomething wrong or unclear, please contact the TAs on [Piazza](http://piazza.com/illinois/fall2020/ece220zjui)\n\t(but do not share your code, test cases or reports). Suggestions are also welcomed. Remember that the tool is only\n\tto **assist** your work. Even if it can't find any issue, it's **not** guaranteed that you will get the full score,\n\tand vice versa.\n\t\n\t**If lc3sim on your own machine generates different result than the feedback, first check whether you have used uninitialized memory or registers.**\n\t\n\t## How to Use Test Cases (Advanced)\n\n\tIf an issue is detected, a corresponding test case will be generated in the folder `test******`. The test data is in\n\tthe asm file. You may copy its content and test your subroutine yourself.\n\n\tThe lcs file is the lc3sim script for you to debug. We have provided a script file for you. Download or checkout this\n\tbranch. In current folder, run the command:\n\n\t```\n\t./replay.sh <test name or index>\n\t```\n\n\twhere `index` is a decimal index of the test case, and the script will launch lc3sim for you, where you can debug.\n\tIf you can't execute the script, you may need:\n\n\t```\n\tchmod +x replay.sh\n\t```")
	/* delete the source files */
	execCommand(dir, "rm", "report/gold.asm")
	execCommand(dir, "rm", "report/student.asm")

	/* check whether .git exists and push to the report branch */
	if PathExists("report" + "/" + task.Payload.Pusher.Name + "/.git") {
		/* the git is already linked: simply push to the remote */
		/* push the generated dir to another branch on GitHub */
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "checkout", "report")
		execCommand(dir, "git", "add", "report")
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "commit", "-m", "Report Generated.")
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "push", "origin", "report", "--force")
	} else {
		/* the git is not linked: init the repo and push the first report to the report branch of the server */
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "init")
		execCommand(dir, "git", "add", "report")
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "commit", "-m", "Report Generated.")
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "remote", "add", "origin", "https://haob2:"+string(PAT)+"@"+task.Payload.Repository.CloneURL[8:])
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "branch", "-m", "report")
		execCommand("report"+"/"+task.Payload.Pusher.Name, "git", "push", "origin", "-u", "report")
	}

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
	logrus.Info(string(output) + ": first argument is " + arg[0])
	if err != nil {
		logrus.Error(err)
		return output, err
	}
	return output, nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

/* append some content to a file */
func append(filename string, content string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(content); err != nil {
		panic(err)
	}
	return nil
}

/* check if there exists a file "MP..." in the directory dir */
func MPExists(dir string, numMP string, exclude string) bool {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.Error(err)
		return false
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "MP"+numMP) && dir+"/"+f.Name() != exclude {
			return true
		}
	}
	return false
}
