package evtmanager

import (
	"container/heap"
	"iter"
	"time"
)

type ProcessStatus int

const (
	StatusCreated ProcessStatus = iota
	StatusScheduled
	StatusRunning
	StatusSuspended
	StatusTerminated
)

// PendingAgent define a interface para entidades na simulação (como processos)
// que podem ser bloqueadas aguardando por eventos, recursos ou timeouts.
type PendingAgent interface {
	// Trigger é chamado pelo Recurso ou Evento para acordar o agente.
	// Retorna true se o agente realmente acordou, false se ele ignorou (ex: All não satisfeito).
	Trigger(env *Environment) bool

	// Cancel é usado para limpezas (ex: no WaitAny, o perdedor é cancelado).
	Cancel(env *Environment)
}

func (s ProcessStatus) String() string {
	return [...]string{"Created", "Scheduled", "Running", "Suspended", "Terminated"}[s]
}

// Process representa uma entidade de execução dentro da simulação (como uma
// thread ou corrotina). Ele gerencia seu próprio ciclo de agendamento, estado e
// iteração dos comandos definidos em sua rotina.
type Process struct {
	id  uint64
	env *Environment

	next func() (Command, bool)
	stop func()

	index int

	// Scheduling
	nextRun  time.Duration
	priority int
	status   ProcessStatus

	// Sync
	currentAgent PendingAgent // O que estou esperando agora
}

// Coloca de volta na fila um process que estava esperando
// e foi desbloqueado.
func (p *Process) Trigger(env *Environment) bool {
	if p.status != StatusSuspended {
		return false
	}

	// Removemos a referência ao agente que o bloqueava
	p.currentAgent = nil

	// Colocamos de volta na Priority Queue de eventos
	p.status = StatusScheduled
	p.nextRun = env.now
	heap.Push(&env.queue, p)

	return true
}

func (p *Process) Cancel(env *Environment) {
	p.currentAgent = nil
	// Aqui poderíamos logar ou atualizar estatísticas de desistência
}

type Command func(env *Environment, proc *Process)

type TaskSimul iter.Seq[Command]
