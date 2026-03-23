package evtmanager

import "container/heap"

type ResourceWaiting struct {
	// processo que está esperando
	proc *Process
	// qual capacidade que o processo quer
	needCapacity int64
	// prioridade
	priority int
	// indice inserido para manter ordenação
	indiceOrd int64
	// index salvo no heap
	index int
	// função chamada ao conseguir o recurso
	funcAfter func()
	//
	res *Resource
}

func (rw *ResourceWaiting) Trigger(env *Environment) bool {
	rw.funcAfter()
	return true
}

func (rw *ResourceWaiting) Cancel(env *Environment) {
	// Remove o processo da fila de espera
	if rw.index >= 0 {
		heap.Remove(&rw.res.waiting, rw.index)
	}
}

type ResourceWaitingQueue []*ResourceWaiting

func (q ResourceWaitingQueue) Len() int { return len(q) }

func (q ResourceWaitingQueue) Less(i, j int) bool {
	if q[i].priority == q[j].priority {
		// CERTO: Usar a propriedade imutável que indica quem chegou primeiro
		return q[i].indiceOrd < q[j].indiceOrd
	}
	return q[i].priority > q[j].priority
}

func (q ResourceWaitingQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *ResourceWaitingQueue) Push(x any) {
	*q = append(*q, x.(*ResourceWaiting))
}

func (q *ResourceWaitingQueue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1
	old[n-1] = nil // evita memory leak
	*q = old[0 : n-1]
	return item
}

type Resource struct {
	name         string
	capacity     int64
	usedCapacity int64
	waiting      ResourceWaitingQueue
	env          *Environment
	indAt        int64
}

func (env *Environment) NewResource(name string, capacity int64) *Resource {
	res := &Resource{
		name:         name,
		capacity:     capacity,
		usedCapacity: 0,
		waiting:      make(ResourceWaitingQueue, 0),
		env:          env,
		indAt:        0,
	}
	heap.Init(&res.waiting)
	return res
}

func (r *Resource) Acquire(proc *Process, capacity int64, prio int, yield func(Command) bool) bool {
	if r.usedCapacity+capacity <= r.capacity {
		r.usedCapacity += capacity
		return true
	}
	// A ideia da função é colocar novamente o processo na fila de eventos para executar instantaneamente
	funcAfter := func() {
		r.env.enqNow(proc)
	}

	yield(func(env *Environment, proc *Process) {
		val := &ResourceWaiting{
			proc:         proc,
			needCapacity: capacity,
			priority:     prio,
			index:        -1,
			indiceOrd:    r.indAt,
			funcAfter:    funcAfter,
			res:          r,
		}
		r.indAt++
		heap.Push(&r.waiting, val)
	})
	return true
}

func (r *Resource) Release(proc *Process, capacity int64) {
	r.usedCapacity -= capacity
	// Checar se tem alguém agora esperando que possa rodar.
	ll := len(r.waiting)
	if ll > 0 {
		first := r.waiting[0]
		available := r.capacity - r.usedCapacity
		if first.needCapacity <= available {
			first := heap.Pop(&r.waiting).(*ResourceWaiting)
			r.usedCapacity += first.needCapacity
			first.Trigger(r.env)
		}
	}
}
