package workerpool

import (
	"sync"
)

type Job interface{}

type Dispatcher struct {
	WorkerPool chan chan Job
	WorkerList []*Worker
	JobQueue   chan Job
	quit       chan bool
	quitWG     sync.WaitGroup
}

func NewDispatcher(maxWaitingJobs int, maxWorkers int) *Dispatcher {
	d := &Dispatcher{
		WorkerPool: make(chan chan Job, maxWorkers),
		JobQueue:   make(chan Job, maxWaitingJobs),
		quit:       make(chan bool, 1),
	}
	return d
}

type JobHandler func(w *Worker, j Job)

func (d *Dispatcher) AddWorker(handler JobHandler) {
	w := &Worker{
		Id:         len(d.WorkerList),
		WorkerPool: d.WorkerPool,
		JobChannel: make(chan Job),
		quit:       make(chan bool),
		handler:    handler,
	}
	d.WorkerList = append(d.WorkerList, w)
}

func (d *Dispatcher) Run() {
	for _, w := range d.WorkerList {
		w.Run()
	}
	go d.Dispatch()
}

func (d *Dispatcher) Dispatch() {
	for {
		select {
		case job := <-d.JobQueue:
			go func(job Job) {
				jobChannel := <-d.WorkerPool
				jobChannel <- job
			}(job)
		case <-d.quit:
			for _, w := range d.WorkerList {
				w.quitWG.Add(1)
				close(w.quit)
				w.quitWG.Wait()
				//w.Stop()
			}
			d.quitWG.Done()
			return
		}
	}
}

func (d *Dispatcher) Queue(job Job) {
	go func() {
		d.JobQueue <- job
	}()
}

func (d *Dispatcher) Stop() {
	d.quitWG.Add(1)
	close(d.quit)
	d.quitWG.Wait()

}

type Worker struct {
	Id         int
	WorkerPool chan chan Job
	JobChannel chan Job
	quit       chan bool
	quitWG     sync.WaitGroup
	handler    JobHandler
	Args       interface{}
}

func (w *Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

func (w *Worker) Run() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel
			select {
			case job := <-w.JobChannel:
				w.handler(w, job)
			case <-w.quit:
				w.quitWG.Done()
				return
			}
		}
	}()
}
