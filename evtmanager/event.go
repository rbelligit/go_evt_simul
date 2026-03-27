package evtmanager

// Event representa algo que vai acontecer no futuro.
// Processos podem aguardar por ele, e quando for acionado (Trigger/Succeed),
// todos os processos aguardando são retomados.
type Event struct {
	name      string
	env       *Environment
	triggered bool
	waiting   []*Process // Lista simples, todos acordam juntos
}

// NewEvent cria um novo evento não disparado.
func (env *Environment) NewEvent(name string) *Event {
	return &Event{
		name:      name,
		env:       env,
		triggered: false,
		waiting:   make([]*Process, 0),
	}
}

// Wait faz o processo aguardar até que o evento ocorra.
// Se o evento já ocorreu, o processo não suspende.
func (e *Event) Wait() Command {
	return func(env *Environment, proc *Process) {
		if e.triggered {
			// Já aconteceu no passado, segue a vida!
			env.enqNow(proc)
			return
		}

		// Ainda não aconteceu, vamos aguardar.
		proc.status = StatusSuspended
		e.waiting = append(e.waiting, proc)
	}
}

// Succeed dispara o evento. Todos que estavam esperando são acordados.
func (e *Event) Succeed() Command {
	return func(env *Environment, proc *Process) {
		if e.triggered {
			// Evita disparar duas vezes
			env.enqNow(proc)
			return
		}

		e.triggered = true

		// Acorda TODO MUNDO que estava esperando
		for _, waitingProc := range e.waiting {
			env.enqNow(waitingProc)
		}

		// Limpa a lista para liberar memória
		e.waiting = nil

		// O processo que disparou o evento também continua
		env.enqNow(proc)
	}
}

// IsTriggered permite checar o estado do evento sem ceder a execução
func (e *Event) IsTriggered() bool {
	return e.triggered
}
