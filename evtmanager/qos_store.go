package evtmanager

import "container/heap"

// QoSStoreReq guarda o pedido, incluindo a Traffic Class (tc) para onde o item vai.
type QoSStoreReq[T any] struct {
	proc      *Process
	item      T
	target    *T
	tc        int // Traffic Class (Índice da fila)
	priority  int
	indiceOrd int64
	index     int
}

// QoSStoreReqQueue é a fila de prioridade para os bloqueios do QoSStore.
type QoSStoreReqQueue[T any] []*QoSStoreReq[T]

func (q QoSStoreReqQueue[T]) Len() int { return len(q) }
func (q QoSStoreReqQueue[T]) Less(i, j int) bool {
	if q[i].priority == q[j].priority {
		return q[i].indiceOrd < q[j].indiceOrd
	}
	return q[i].priority > q[j].priority
}
func (q QoSStoreReqQueue[T]) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}
func (q *QoSStoreReqQueue[T]) Push(x any) {
	req := x.(*QoSStoreReq[T])
	req.index = len(*q)
	*q = append(*q, req)
}
func (q *QoSStoreReqQueue[T]) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1
	old[n-1] = nil // evita memory leak
	*q = old[0 : n-1]
	return item
}

// QoSStore simula uma porta de rede com múltiplas filas (Traffic Classes)
// e algoritmo de agendamento Weighted Round Robin (WRR).
type QoSStore[T any] struct {
	name     string
	capacity int64 // Capacidade GLOBAL do buffer (soma de todas as filas)
	used     int64 // Quantidade de itens atualmente no buffer global

	queues  [][]T // As N filas internas
	weights []int // Os pesos de cada fila para o WRR

	// Estado do algoritmo Smooth Weighted Round Robin (SWRR)
	currentWeights []int // Peso acumulado atual de cada fila

	putQueue QoSStoreReqQueue[T]
	getQueue QoSStoreReqQueue[T]
	env      *Environment
	indAt    int64
}

// NewQoSStore cria um buffer com N filas baseadas no tamanho do slice de pesos.
// Exemplo: weights = []int{8, 4, 2, 1} cria 4 filas com essas proporções de leitura.
func NewQoSStore[T any](env *Environment, name string, capacity int64, weights []int) *QoSStore[T] {
	if len(weights) == 0 {
		panic("QoSStore requires at least one weight/queue")
	}

	queues := make([][]T, len(weights))
	for i := range queues {
		queues[i] = make([]T, 0)
	}

	s := &QoSStore[T]{
		name:     name,
		capacity: capacity,
		used:     0,
		queues:   queues,
		weights:  weights,

		currentWeights: make([]int, len(weights)),

		putQueue: make(QoSStoreReqQueue[T], 0),
		getQueue: make(QoSStoreReqQueue[T], 0),
		env:      env,
		indAt:    0,
	}
	heap.Init(&s.putQueue)
	heap.Init(&s.getQueue)
	return s
}

// extractNextWRR executa a lógica do Smooth Weighted Round Robin (SWRR) para escolher o próximo pacote.
// O algoritmo SWRR soma o peso configurado ao peso acumulado de todas as filas não vazias.
// Em seguida, seleciona a fila com o maior peso acumulado, desconta o peso total ativo dela e lê o pacote.
// Isso garante um balanceamento intercalado e suave entre as prioridades.
// IMPORTANTE: Esta função só deve ser chamada se tivermos certeza de que s.used > 0.
func (s *QoSStore[T]) extractNextWRR() T {
	totalActiveWeight := 0
	bestQueue := -1
	maxWeight := 0

	// Passo 1: Soma os pesos e identifica o total ativo
	for i := range s.queues {
		if len(s.queues[i]) > 0 {
			s.currentWeights[i] += s.weights[i]
			totalActiveWeight += s.weights[i]

			if bestQueue == -1 || s.currentWeights[i] > maxWeight {
				bestQueue = i
				maxWeight = s.currentWeights[i]
			}
		}
	}

	// Passo 2: Deduz o custo (peso total) da fila vencedora
	s.currentWeights[bestQueue] -= totalActiveWeight

	// Passo 3: Remove e retorna o item do topo desta fila
	item := s.queues[bestQueue][0]
	s.queues[bestQueue] = s.queues[bestQueue][1:]

	return item
}

// Put tenta colocar um item na fila específica (tc - Traffic Class).
func (s *QoSStore[T]) Put(item T, tc int, prio int) Command {
	return func(env *Environment, proc *Process) {
		// Proteção contra índice fora do limite
		if tc < 0 || tc >= len(s.weights) {
			tc = len(s.weights) - 1 // Joga para a fila de menor prioridade (última)
		}

		// 1. Se tem um leitor bloqueado (Get) esperando, entregamos direto!
		// Se havia alguém bloqueado, significa que o Store inteiro estava vazio (used == 0),
		// então não precisamos rodar o WRR, é só entregar a única coisa que chegou.
		if len(s.getQueue) > 0 {
			req := heap.Pop(&s.getQueue).(*QoSStoreReq[T])
			*(req.target) = item // Salva direto no ponteiro do leitor
			env.enqNow(req.proc) // Acorda o leitor
			env.enqNow(proc)     // O produtor continua
			return
		}

		// 2. Se há capacidade global, armazenamos na fila específica
		if s.used < s.capacity {
			s.queues[tc] = append(s.queues[tc], item)
			s.used++
			env.enqNow(proc)
			return
		}

		// 3. Buffer cheio: Backpressure. O produtor é suspenso.
		proc.status = StatusSuspended
		req := &QoSStoreReq[T]{
			proc:      proc,
			item:      item,
			tc:        tc,
			priority:  prio,
			indiceOrd: s.indAt,
		}
		s.indAt++
		heap.Push(&s.putQueue, req)
	}
}

// Get extrai um item do QoSStore aplicando as regras de proporção (WRR).
func (s *QoSStore[T]) Get(target *T, prio int) Command {
	return func(env *Environment, proc *Process) {
		// 1. Se tem itens no buffer global, aplicamos a matemática do WRR
		if s.used > 0 {
			*target = s.extractNextWRR()
			s.used--

			// Como tiramos 1 pacote, abriu 1 vaga global.
			// Vamos ver se tem algum produtor esperando para injetar pacote.
			if len(s.putQueue) > 0 {
				req := heap.Pop(&s.putQueue).(*QoSStoreReq[T])
				// Injetamos o pacote do produtor na fila que ele queria
				s.queues[req.tc] = append(s.queues[req.tc], req.item)
				s.used++
				env.enqNow(req.proc) // Acordamos o produtor
			}

			env.enqNow(proc) // O leitor continua
			return
		}

		// 2. Buffer vazio, mas pode ser um Store de capacidade 0 (Síncrono)
		if len(s.putQueue) > 0 {
			req := heap.Pop(&s.putQueue).(*QoSStoreReq[T])
			*target = req.item
			env.enqNow(req.proc)
			env.enqNow(proc)
			return
		}

		// 3. Tudo vazio. O leitor é suspenso até chegar pacote.
		proc.status = StatusSuspended
		req := &QoSStoreReq[T]{
			proc:      proc,
			target:    target,
			priority:  prio,
			indiceOrd: s.indAt,
		}
		s.indAt++
		heap.Push(&s.getQueue, req)
	}
}
