package evtmanager

import "container/heap"

// StoreReq representa um processo aguardando para fazer um Put ou Get
type StoreReq[T any] struct {
	proc      *Process
	item      T  // Preenchido quando é um request de Put
	target    *T // Ponteiro para onde salvar o dado quando é Get
	priority  int
	indiceOrd int64
	index     int
}

// StoreReqQueue é a fila de prioridade genérica para o Store
type StoreReqQueue[T any] []*StoreReq[T]

func (q StoreReqQueue[T]) Len() int { return len(q) }

func (q StoreReqQueue[T]) Less(i, j int) bool {
	if q[i].priority == q[j].priority {
		return q[i].indiceOrd < q[j].indiceOrd
	}
	return q[i].priority > q[j].priority
}

func (q StoreReqQueue[T]) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *StoreReqQueue[T]) Push(x any) {
	req := x.(*StoreReq[T])
	req.index = len(*q)
	*q = append(*q, req)
}

func (q *StoreReqQueue[T]) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1
	old[n-1] = nil // evita memory leak
	*q = old[0 : n-1]
	return item
}

// Store é um buffer (Fila de Dados) para comunicação entre processos.
// Pode simular filas de pacotes, buffers de switch ou canais limitados.
type Store[T any] struct {
	name     string
	capacity int64
	items    []T
	putQueue StoreReqQueue[T]
	getQueue StoreReqQueue[T]
	env      *Environment
	indAt    int64
}

// NewStore cria uma nova fila de dados com capacidade limitada.
func NewStore[T any](env *Environment, name string, capacity int64) *Store[T] {
	s := &Store[T]{
		name:     name,
		capacity: capacity,
		items:    make([]T, 0),
		putQueue: make(StoreReqQueue[T], 0),
		getQueue: make(StoreReqQueue[T], 0),
		env:      env,
		indAt:    0,
	}
	heap.Init(&s.putQueue)
	heap.Init(&s.getQueue)
	return s
}

// Put tenta colocar um item no Store. Se estiver cheio, o processo é suspenso.
func (s *Store[T]) Put(item T, prio int) Command {
	return func(env *Environment, proc *Process) {
		// 1. Se tem alguém esperando para ler (Get), entregamos direto "em mãos"
		if len(s.getQueue) > 0 {
			req := heap.Pop(&s.getQueue).(*StoreReq[T])
			*(req.target) = item // Salva no ponteiro do processo leitor
			env.enqNow(req.proc) // Acorda o leitor
			env.enqNow(proc)     // Este escritor continua imediatamente
			return
		}

		// 2. Se há espaço no buffer, armazenamos
		if int64(len(s.items)) < s.capacity {
			s.items = append(s.items, item)
			env.enqNow(proc)
			return
		}

		// 3. Buffer cheio: suspendemos o processo atual (Backpressure)
		proc.status = StatusSuspended
		req := &StoreReq[T]{
			proc:      proc,
			item:      item,
			priority:  prio,
			indiceOrd: s.indAt,
		}
		s.indAt++
		heap.Push(&s.putQueue, req)
	}
}

func (s *Store[T]) Len() int {
	return len(s.items) + len(s.putQueue)
}

// Get tenta retirar um item do Store. Se estiver vazio, o processo é suspenso.
// O item será salvo na variável apontada por `target`.
func (s *Store[T]) Get(target *T, prio int) Command {
	return func(env *Environment, proc *Process) {
		// 1. Se tem item no buffer, pegamos o primeiro (FIFO)
		if len(s.items) > 0 {
			*target = s.items[0]
			s.items = s.items[1:]

			// Como abrimos espaço, checamos se havia algum escritor bloqueado
			if len(s.putQueue) > 0 {
				req := heap.Pop(&s.putQueue).(*StoreReq[T])
				s.items = append(s.items, req.item) // Colocamos o item dele no buffer
				env.enqNow(req.proc)                // Acordamos o escritor
			}

			env.enqNow(proc) // Este leitor continua
			return
		}

		// 2. Buffer vazio, mas pode haver um escritor bloqueado (capacidade 0)
		if len(s.putQueue) > 0 {
			req := heap.Pop(&s.putQueue).(*StoreReq[T])
			*target = req.item   // Pegamos direto do escritor
			env.enqNow(req.proc) // Acordamos o escritor
			env.enqNow(proc)     // Este leitor continua
			return
		}

		// 3. Tudo vazio: suspendemos o processo atual
		proc.status = StatusSuspended
		req := &StoreReq[T]{
			proc:      proc,
			target:    target,
			priority:  prio,
			indiceOrd: s.indAt,
		}
		s.indAt++
		heap.Push(&s.getQueue, req)
	}
}
