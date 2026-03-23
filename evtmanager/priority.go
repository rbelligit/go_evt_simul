package evtmanager

type PriorityQueue []*Process

func (pq PriorityQueue) Len() int { return len(pq) }

// Less define a ordem da fila (Menor tempo = Maior prioridade)
func (pq PriorityQueue) Less(i, j int) bool {
	// Se os tempos forem iguais, usamos a prioridade do processo como desempate
	if pq[i].nextRun == pq[j].nextRun {
		return pq[i].priority > pq[j].priority
	}
	return pq[i].nextRun < pq[j].nextRun
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i // Atualiza o índice para permitir heap.Fix
	pq[j].index = j
}

// Push e Pop são usados internamente pelo pacote heap
func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Process)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // evita memory leak
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

// Peek retorna o próximo processo sem removê-lo
func (pq PriorityQueue) Peek() *Process {
	if len(pq) == 0 {
		return nil
	}
	return pq[0]
}
