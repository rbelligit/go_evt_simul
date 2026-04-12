package evtmanager

import "time"

// TimeoutEvent é apenas um wrapper(atalho) em cima de env.Timeout
// para manter um padrão de nomenclatura condutivo a retornos de Eventos.
func TimeoutEvent(env *Environment, delay time.Duration) *Event {
	return env.Timeout(delay)
}

// ContainerGetEvent cria um Evento englobando o ato de esperar por um nível em um Container.
// Isso permite usar chamadas de Container de forma compatível com "AnyOf" e "AllOf".
func ContainerGetEvent(env *Environment, c *Container, amount float64, prio int) *Event {
	evt := env.NewEvent("ContainerGetEvent")
	canceled := false

	evt.AddCancelCallback(func() {
		canceled = true
	})

	env.AddTask(func(yield func(Command) bool) {
		// Tenta alocar a quantidade usando o yield tradicional do simulador
		yield(c.Get(amount, prio))

		// Ao ser acordado porque conseguiu o recurso...
		if !canceled {
			yield(evt.Succeed())
		} else {
			// Solução de design (Workaround): se fomos cancelados por um AnyOf, 
			// o container já nos cedeu o recurso neste ponto pois só fomos acordados 
			// quando chegou nossa vez. Precisamos "devolver" o recurso para não dar vazamento.
			yield(c.Put(amount, prio))
		}
	}, 0)

	return evt
}

// ContainerPutEvent cria um Evento englobando o ato de esperar para inserir carga em um Container.
func ContainerPutEvent(env *Environment, c *Container, amount float64, prio int) *Event {
	evt := env.NewEvent("ContainerPutEvent")
	canceled := false

	evt.AddCancelCallback(func() {
		canceled = true
	})

	env.AddTask(func(yield func(Command) bool) {
		yield(c.Put(amount, prio))
		if !canceled {
			yield(evt.Succeed())
		} else {
			// Remove o que foi inserido já que a operação como um todo foi cancelada pelo AnyOf
			yield(c.Get(amount, prio))
		}
	}, 0)

	return evt
}

// ResourceAcquireEvent cria um Evento englobando o ato de alocar capacidade em um Resource.
func ResourceAcquireEvent(env *Environment, r *Resource, capacity int64, prio int) *Event {
	evt := env.NewEvent("ResourceAcquireEvent")
	canceled := false

	evt.AddCancelCallback(func() {
		canceled = true
	})

	env.AddTask(func(yield func(Command) bool) {
		var proc *Process
		GetProcess(&proc, yield)

		// O Acquire usa uma função de yield e um env.enqNow nativo para retornar
		r.Acquire(proc, capacity, prio, yield)

		if !canceled {
			yield(evt.Succeed())
		} else {
			// Workaround de cancelamento
			r.Release(proc, capacity)
		}
	}, 0)

	return evt
}

// StoreGetEvent cria um Evento englobando o ato de esperar por um item no Store.
func StoreGetEvent[T any](env *Environment, s *Store[T], target *T, prio int) *Event {
	evt := env.NewEvent("StoreGetEvent")
	canceled := false
	evt.AddCancelCallback(func() { canceled = true })

	env.AddTask(func(yield func(Command) bool) {
		yield(s.Get(target, prio))
		if !canceled {
			yield(evt.Succeed())
		} else {
			// Workaround: Devolve ao store (cuidado: irá para o final da fila FIFO)
			yield(s.Put(*target, prio))
		}
	}, 0)

	return evt
}

// StorePutEvent cria um Evento englobando o ato de inserir um item no Store.
func StorePutEvent[T any](env *Environment, s *Store[T], item T, prio int) *Event {
	evt := env.NewEvent("StorePutEvent")
	canceled := false
	evt.AddCancelCallback(func() { canceled = true })

	env.AddTask(func(yield func(Command) bool) {
		yield(s.Put(item, prio))
		if !canceled {
			yield(evt.Succeed())
		} else {
			// Workaround genérico: Remove um item do store para abater o put cancelado
			var dummy T
			yield(s.Get(&dummy, prio))
		}
	}, 0)

	return evt
}

// FilterStoreGetEvent cria um Evento englobando o Get do FilterStore.
func FilterStoreGetEvent[T any](env *Environment, s *FilterStore[T], target *T, filter func(T) bool, prio int) *Event {
	evt := env.NewEvent("FilterStoreGetEvent")
	canceled := false
	evt.AddCancelCallback(func() { canceled = true })

	env.AddTask(func(yield func(Command) bool) {
		yield(s.Get(target, filter, prio))
		if !canceled {
			yield(evt.Succeed())
		} else {
			// Workaround: devolve o item obtido
			yield(s.Put(*target, prio))
		}
	}, 0)

	return evt
}

// FilterStorePutEvent cria um Evento englobando o Put do FilterStore.
func FilterStorePutEvent[T any](env *Environment, s *FilterStore[T], item T, prio int) *Event {
	evt := env.NewEvent("FilterStorePutEvent")
	canceled := false
	evt.AddCancelCallback(func() { canceled = true })

	env.AddTask(func(yield func(Command) bool) {
		yield(s.Put(item, prio))
		if !canceled {
			yield(evt.Succeed())
		} else {
			// Workaround: Remove o item específico que tentamos por caso consigamos
			var dummy T
			yield(s.Get(&dummy, nil, prio))
		}
	}, 0)

	return evt
}
