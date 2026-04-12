package evtmanager

// Event representa algo que vai acontecer no futuro.
// Processos podem aguardar por ele, e quando for acionado (Trigger/Succeed),
// todos os processos aguardando são retomados.
type Event struct {
	name      string
	env       *Environment
	triggered bool
	canceled  bool       // Flag indicando se o evento foi cancelado
	waiting   []*Process // Lista simples, todos acordam juntos
	callbacks []func()   // Funções internas a serem executadas no Succeed
	cancel    []func()   // Função chamada se o event for cancelado, chamado se o cancel do evento for chamado
}

// NewEvent cria um novo evento não disparado.
func (env *Environment) NewEvent(name string) *Event {
	return &Event{
		name:      name,
		env:       env,
		triggered: false,
		waiting:   make([]*Process, 0),
		callbacks: make([]func(), 0),
	}
}

func (e *Event) Cancel() {
	if e.canceled || e.triggered {
		return
	}
	e.canceled = true

	for _, c := range e.cancel {
		c()
	}
	e.cancel = nil
	e.callbacks = nil // Limpa callbacks pois não vai mais disparar

	// Acorda quem estava esperando o evento
	for _, waitingProc := range e.waiting {
		e.env.enqNow(waitingProc)
	}
	e.waiting = nil
}

// AddCancelCallback adiciona uma função para ser executada se o evento for cancelado.
func (e *Event) AddCancelCallback(cb func()) {
	if e.canceled { return }
	e.cancel = append(e.cancel, cb)
}

// AddCallback adiciona uma função para ser executada internamente quando o evento ocorrer (Succeed).
func (e *Event) AddCallback(cb func()) {
	e.callbacks = append(e.callbacks, cb)
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
		if e.triggered || e.canceled {
			// Evita disparar duas vezes ou se foi cancelado
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

		//Dispara os callbacks do WaitAny/WaitAll
		for _, cb := range e.callbacks {
			cb()
		}
		e.callbacks = nil // Limpa para liberar memória

		// O processo que disparou o evento também continua
		env.enqNow(proc)
	}
}

// IsTriggered permite checar o estado do evento sem ceder a execução
func (e *Event) IsTriggered() bool {
	return e.triggered
}
