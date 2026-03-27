package evtmanager

import "container/heap"

// FilterStoreReq guarda o pedido, incluindo a função de filtro
type FilterStoreReq[T any] struct {
	proc      *Process
	item      T
	target    *T
	filter    func(T) bool // <-- O Segredo está aqui!
	priority  int
	indiceOrd int64
	index     int
}

// Fila de prioridade para o FilterStore (mesma lógica das outras)
type FilterStoreReqQueue[T any] []*FilterStoreReq[T]

func (q FilterStoreReqQueue[T]) Len() int { return len(q) }
func (q FilterStoreReqQueue[T]) Less(i, j int) bool {
	if q[i].priority == q[j].priority {
		return q[i].indiceOrd < q[j].indiceOrd
	}
	return q[i].priority > q[j].priority
}
func (q FilterStoreReqQueue[T]) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}
func (q *FilterStoreReqQueue[T]) Push(x any) {
	req := x.(*FilterStoreReq[T])
	req.index = len(*q)
	*q = append(*q, req)
}
func (q *FilterStoreReqQueue[T]) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1
	old[n-1] = nil
	*q = old[0 : n-1]
	return item
}

// FilterStore permite obter itens baseados em critérios customizados
type FilterStore[T any] struct {
	name     string
	capacity int64
	items    []T
	putQueue FilterStoreReqQueue[T]
	getQueue FilterStoreReqQueue[T]
	env      *Environment
	indAt    int64
}

func NewFilterStore[T any](env *Environment, name string, capacity int64) *FilterStore[T] {
	s := &FilterStore[T]{
		name:     name,
		capacity: capacity,
		items:    make([]T, 0),
		putQueue: make(FilterStoreReqQueue[T], 0),
		getQueue: make(FilterStoreReqQueue[T], 0),
		env:      env,
		indAt:    0,
	}
	heap.Init(&s.putQueue)
	heap.Init(&s.getQueue)
	return s
}

// Put tenta inserir um item. Se houver um Get bloqueado que aceite este item, entrega direto.
func (s *FilterStore[T]) Put(item T, prio int) Command {
	return func(env *Environment, proc *Process) {
		// 1. Procura na fila de Get se alguém quer esse item específico
		for i := 0; i < len(s.getQueue); i++ {
			req := s.getQueue[i]
			// Testa o filtro do consumidor bloqueado contra o novo item
			if req.filter == nil || req.filter(item) {
				// Achou um interessado! Remove da fila de espera (Heap Remove)
				heap.Remove(&s.getQueue, req.index)

				*(req.target) = item // Entrega o item
				env.enqNow(req.proc) // Acorda o consumidor
				env.enqNow(proc)     // O produtor segue a vida
				return
			}
		}

		// 2. Ninguém quis imediatamente. Tem espaço no buffer?
		if int64(len(s.items)) < s.capacity {
			s.items = append(s.items, item)
			env.enqNow(proc)
			return
		}

		// 3. Buffer cheio. Bloqueia o produtor.
		proc.status = StatusSuspended
		req := &FilterStoreReq[T]{
			proc:      proc,
			item:      item,
			priority:  prio,
			indiceOrd: s.indAt,
		}
		s.indAt++
		heap.Push(&s.putQueue, req)
	}
}

// Get solicita um item que passe na função 'filter'.
func (s *FilterStore[T]) Get(target *T, filter func(T) bool, prio int) Command {
	return func(env *Environment, proc *Process) {
		// 1. Procura no buffer um item que passe no filtro
		for i, item := range s.items {
			if filter == nil || filter(item) {
				*target = item

				// Remove o item do buffer (mantendo a ordem dos outros)
				s.items = append(s.items[:i], s.items[i+1:]...)

				// Como abriu espaço, acorda UM produtor que estava bloqueado (se houver)
				if len(s.putQueue) > 0 {
					req := heap.Pop(&s.putQueue).(*FilterStoreReq[T])
					s.items = append(s.items, req.item) // Coloca o item dele no buffer
					env.enqNow(req.proc)                // Acorda o produtor
				}

				env.enqNow(proc) // O consumidor segue a vida
				return
			}
		}

		// 2. Não achou nada que sirva. Suspende e salva a função de filtro!
		proc.status = StatusSuspended
		req := &FilterStoreReq[T]{
			proc:      proc,
			target:    target,
			filter:    filter, // Guarda o critério de busca
			priority:  prio,
			indiceOrd: s.indAt,
		}
		s.indAt++
		heap.Push(&s.getQueue, req)
	}
}
