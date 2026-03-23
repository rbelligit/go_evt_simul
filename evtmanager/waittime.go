package evtmanager

import (
	"container/heap"
	"time"
)

// Aqui teremos a implementação do command para adicionar o processo na lista com um timeout T

// WaitTime é um comando que faz o processo esperar por um tempo T

func WaitTime(t time.Duration) Command {
	return func(env *Environment, proc *Process) {
		proc.nextRun = env.now + t
		proc.status = StatusSuspended
		if proc.index >= 0 {
			heap.Fix(&env.queue, proc.index)
		} else {
			heap.Push(&env.queue, proc)
		}
	}
}
