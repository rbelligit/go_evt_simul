package evtmanager

import "time"

// AnyOf cria um Meta-Evento que é disparado quando PELO MENOS UM dos eventos fornecidos ocorrer.
func (env *Environment) AnyOf(events ...*Event) *Event {
	condEvent := env.NewEvent("AnyOf")

	// 1. Verificação instantânea: Algum já disparou no passado?
	for _, e := range events {
		if e.triggered {
			condEvent.triggered = true
			return condEvent // Retorna já disparado, o Wait() não vai suspender o processo.
		}
	}

	// Registra para que, quando o AnyOf for disparado ou cancelado, ele cancele quem ainda está "vivo"
	condEvent.AddCallback(func() {
		for _, e := range events {
			if !e.triggered {
				e.Cancel()
			}
		}
	})

	// 2. Nenhum disparou ainda. Vamos "grampear" todos eles.
	for _, e := range events {
		// Adiciona um ouvinte em cada evento filho
		e.AddCallback(func() {
			// Se o condEvent AINDA não disparou (garante o One-Shot)
			if !condEvent.triggered {
				condEvent.triggered = true

				// Acorda manualmente quem estava esperando pelo AnyOf
				for _, p := range condEvent.waiting {
					env.enqNow(p)
				}
				condEvent.waiting = nil

				// Dispara os callbacks registrados no próprio AnyOf (como o de cancelamento)
				for _, cb := range condEvent.callbacks {
					cb()
				}
				condEvent.callbacks = nil
			}
		})
	}

	return condEvent
}

// AllOf cria um Meta-Evento que só é disparado quando TODOS os eventos fornecidos ocorrerem.
func (env *Environment) AllOf(events ...*Event) *Event {
	condEvent := env.NewEvent("AllOf")

	checkAll := func() {
		if condEvent.triggered {
			return
		}
		for _, e := range events {
			if !e.triggered {
				return // Faltou um! Aborta.
			}
		}

		// Todos dispararam!
		condEvent.triggered = true
		for _, p := range condEvent.waiting {
			env.enqNow(p)
		}
		condEvent.waiting = nil

		// Dispara os callbacks
		for _, cb := range condEvent.callbacks {
			cb()
		}
		condEvent.callbacks = nil
	}

	// 1. Verificação instantânea
	checkAll()
	if condEvent.triggered {
		return condEvent
	}

	// 2. Grampeia os eventos que ainda não dispararam
	for _, e := range events {
		if !e.triggered {
			e.AddCallback(checkAll)
		}
	}

	return condEvent
}

// Timeout cria um evento que dispara automaticamente após um determinado tempo virtual.
// Fundamental para usar junto com AnyOf para simular falhas de rede.
func (env *Environment) Timeout(delay time.Duration) *Event {
	e := env.NewEvent("Timeout")

	canceled := false
	e.AddCancelCallback(func() {
		canceled = true
	})

	// Cria um micro-processo interno invisível só para contar o tempo e disparar
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(delay))
		if !canceled {
			yield(e.Succeed())
		}
	}, 0)

	return e
}
