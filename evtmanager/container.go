package evtmanager

import "container/heap"

// ContainerReq representa um processo aguardando para inserir (Put) ou retirar (Get)
type ContainerReq struct {
	proc      *Process
	amount    float64
	priority  int
	indiceOrd int64
	index     int
}

// ContainerReqQueue é a fila de prioridade para os bloqueios do Container
type ContainerReqQueue []*ContainerReq

func (q ContainerReqQueue) Len() int { return len(q) }

func (q ContainerReqQueue) Less(i, j int) bool {
	if q[i].priority == q[j].priority {
		return q[i].indiceOrd < q[j].indiceOrd
	}
	return q[i].priority > q[j].priority
}

func (q ContainerReqQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *ContainerReqQueue) Push(x any) {
	req := x.(*ContainerReq)
	req.index = len(*q)
	*q = append(*q, req)
}

func (q *ContainerReqQueue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1
	old[n-1] = nil // evita memory leak
	*q = old[0 : n-1]
	return item
}

// Container modela recursos contínuos e homogêneos (ex: água, energia, bytes).
type Container struct {
	name     string
	capacity float64
	level    float64
	putQueue ContainerReqQueue
	getQueue ContainerReqQueue
	env      *Environment
	indAt    int64
}

// NewContainer cria um novo reservatório.
// capacity: limite máximo. initLevel: quantidade com a qual o tanque já começa.
func (env *Environment) NewContainer(name string, capacity float64, initLevel float64) *Container {
	c := &Container{
		name:     name,
		capacity: capacity,
		level:    initLevel,
		putQueue: make(ContainerReqQueue, 0),
		getQueue: make(ContainerReqQueue, 0),
		env:      env,
		indAt:    0,
	}
	heap.Init(&c.putQueue)
	heap.Init(&c.getQueue)
	return c
}

// triggerGets é uma função interna que tenta acordar processos esperando por material em cascata.
func (c *Container) triggerGets() {
	for len(c.getQueue) > 0 {
		first := c.getQueue[0] // Apenas Peek
		if c.level >= first.amount {
			// Tem material suficiente! Remove da fila e acorda.
			req := heap.Pop(&c.getQueue).(*ContainerReq)
			c.level -= req.amount
			c.env.enqNow(req.proc)
		} else {
			// Respeita a fila: se o primeiro não pode ser atendido, os demais devem esperar.
			break
		}
	}
}

// triggerPuts é uma função interna que tenta acordar processos esperando por espaço em cascata.
func (c *Container) triggerPuts() {
	for len(c.putQueue) > 0 {
		first := c.putQueue[0]
		spaceLeft := c.capacity - c.level
		if spaceLeft >= first.amount {
			// Tem espaço livre suficiente! Remove da fila e acorda.
			req := heap.Pop(&c.putQueue).(*ContainerReq)
			c.level += req.amount
			c.env.enqNow(req.proc)
		} else {
			break
		}
	}
}

// Put tenta adicionar uma quantidade ao container.
// Se não houver espaço suficiente para acomodar a quantidade, o processo suspende.
func (c *Container) Put(amount float64, prio int) Command {
	return func(env *Environment, proc *Process) {
		if amount <= 0 {
			env.enqNow(proc)
			return
		}

		spaceLeft := c.capacity - c.level

		// Sucesso imediato: a fila está vazia e temos espaço.
		if len(c.putQueue) == 0 && spaceLeft >= amount {
			c.level += amount
			c.triggerGets() // Avisa quem estava com "sede"
			env.enqNow(proc)
			return
		}

		// Bloqueio: sem espaço suficiente ou há processos prioritários na frente.
		proc.status = StatusSuspended
		req := &ContainerReq{
			proc:      proc,
			amount:    amount,
			priority:  prio,
			indiceOrd: c.indAt,
		}
		c.indAt++
		heap.Push(&c.putQueue, req)
	}
}

// Get tenta retirar uma quantidade do container.
// Se não houver material suficiente, o processo suspende.
func (c *Container) Get(amount float64, prio int) Command {
	return func(env *Environment, proc *Process) {
		if amount <= 0 {
			env.enqNow(proc)
			return
		}

		// Sucesso imediato: a fila está vazia e temos material suficiente.
		if len(c.getQueue) == 0 && c.level >= amount {
			c.level -= amount
			c.triggerPuts() // Avisa quem queria esvaziar sua carga no tanque
			env.enqNow(proc)
			return
		}

		// Bloqueio: sem material suficiente ou há processos prioritários na frente.
		proc.status = StatusSuspended
		req := &ContainerReq{
			proc:      proc,
			amount:    amount,
			priority:  prio,
			indiceOrd: c.indAt,
		}
		c.indAt++
		heap.Push(&c.getQueue, req)
	}
}

// Level retorna a quantidade atual disponível no container.
func (c *Container) Level() float64 {
	return c.level
}
