package evtmanager

import (
	"container/heap"
	"iter"
	"time"
)

// Interafa do sistema de gerenciados eventos, onde tasks são adicionadas e mais....

type Environment struct {
	now             time.Duration
	queue           PriorityQueue // Implementação de container/heap
	idCounter       uint64
	shouldStop      bool
	activeProcesses int

	// Opcional: Para estatísticas
	eventsProcessed uint64

	isRunning bool
}

func NewEnvironment() *Environment {
	env := &Environment{
		now:             0,
		idCounter:       0,
		shouldStop:      false,
		activeProcesses: 0,
		eventsProcessed: 0,
		queue:           make(PriorityQueue, 0),
		isRunning:       false,
	}
	heap.Init(&env.queue)
	return env
}

func (ev *Environment) Enq(proc *Process) {
	proc.status = StatusScheduled
	heap.Push(&ev.queue, proc)
}

func (ev *Environment) enqNow(proc *Process) {
	proc.nextRun = ev.now

	ev.Enq(proc)
}

func (ev *Environment) AddTask(task TaskSimul, priority int) error {
	ev.idCounter++
	next, stop := iter.Pull(iter.Seq[Command](task))
	proc := &Process{
		id:           ev.idCounter,
		env:          ev,
		next:         next,
		stop:         stop,
		nextRun:      ev.now,
		priority:     priority,
		status:       StatusCreated,
		currentAgent: nil,
	}
	if ev.isRunning {
		proc.status = StatusScheduled
	}
	ev.Enq(proc)

	return nil
}

func (ev *Environment) StartSimul(runFor time.Duration) error {
	ev.isRunning = true
	ev.shouldStop = false

	targetTime := ev.now + runFor

	for len(ev.queue) > 0 && !ev.shouldStop {
		// Peek the next process
		proc := heap.Pop(&ev.queue).(*Process)

		if proc.nextRun > targetTime {
			break
		}

		if proc.nextRun > ev.now {
			ev.now = proc.nextRun
		}

		proc.index = -1
		proc.status = StatusRunning

		command, sts := proc.next()
		if sts {
			if command != nil {
				command(ev, proc)
			}
		}
	}

	//if !ev.shouldStop && ev.now < targetTime {
	//	ev.now = targetTime
	//}

	ev.isRunning = false
	return nil
}

func (ev *Environment) StopSimul() error {
	ev.shouldStop = true
	return nil
}

func (ev *Environment) GetTime() time.Duration {
	return ev.now
}

func GetProcess(pr **Process, yield func(Command) bool) {
	yield(func(env *Environment, p *Process) {
		*pr = p
		// Reenfileira para continuar a execução no mesmo instante
		p.status = StatusScheduled
		p.nextRun = env.now
		env.Enq(p)
	})
}
