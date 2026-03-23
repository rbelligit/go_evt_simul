package evtmanager

import (
	"testing"
	"time"
)

func TestResource_AcquireAndReleaseE2E(t *testing.T) {
	env := NewEnvironment()
	res := env.NewResource("CPU", 1) // Capacidade 1

	var p1, p2 *Process
	var finalTimeP2 time.Duration

	// Processo 1 vai adquirir, usar por 5s e liberar.
	task1 := func(yield func(Command) bool) {
		GetProcess(&p1, yield) // Magia para pegar o *Process da nossa própria task

		res.Acquire(p1, 1, 1, yield)

		// Segura o recurso por 5 segundos simulados
		yield(WaitTime(2 * time.Second))

		res.Release(p1, 1)
	}

	// Processo 2 vai tentar adquirir. Como o p1 pegou primeiro, ele deve ficar bloqueado
	// e só rodar DEPOIS que p1 liberar (no tempo simulado de 5s).
	task2 := func(yield func(Command) bool) {
		GetProcess(&p2, yield) // Magia para pegar o *Process da nossa própria task
		yield(WaitTime(time.Second))
		// Aqui o yield vai registrar que ele quer adquirir, e vai suspendê-lo internamente!
		res.Acquire(p2, 1, 1, yield)

		finalTimeP2 = env.GetTime()
		// Normalmente aqui é 2s

		// Segura por 3 segundos simulados e libera
		yield(WaitTime(3 * time.Second))

		res.Release(p2, 1)
	}

	env.AddTask(task1, 0)
	env.AddTask(task2, 0) // Ambos na fila com prioridade igual de escalonamento

	env.StartSimul(10 * time.Second)

	// Validações
	// O Processo p2 deveria ter rodado apenas depois que p1 soltou (que ocorreu no tempo 5)
	if finalTimeP2 != 2*time.Second {
		t.Errorf("expected Processo 2 to acquire resource at 2s, but got %v", finalTimeP2)
	}

	// A simulação completa deve rodar em 7s (5s do p1 + 2s do p2)
	if env.GetTime() != 5*time.Second {
		t.Errorf("expected total simulation time 7s, got %v", env.GetTime())
	}
}

func TestResource_PriorityOrderingE2E(t *testing.T) {
	env := NewEnvironment()
	res := env.NewResource("Printer", 1) // Capacidade 1

	var pBlocker, pHigh, pLow *Process
	logOrder := make([]string, 0)

	// Processo bloqueador trava a impressora logo no inicio do tempo 0
	taskBlocker := func(yield func(Command) bool) {
		GetProcess(&pBlocker, yield)
		res.Acquire(pBlocker, 1, 1, yield)

		// Segura por 10s para forçar os outros dois a entrarem na fila de pendência do recurso
		yield(WaitTime(10 * time.Second))
		res.Release(pBlocker, 1)
	}

	// Processo de BAIXA prioridade para o recurso (Priority 1)
	taskLow := func(yield func(Command) bool) {
		GetProcess(&pLow, yield)
		yield(WaitTime(1 * time.Second)) // Ele chega no tempo simulado 1s, tenta adquirir o recurso e enfileira

		res.Acquire(pLow, 1, 1, yield) // O Acquire vai suspender a thread aqui

		logOrder = append(logOrder, "Low")

		yield(WaitTime(1 * time.Second))
		res.Release(pLow, 1)
	}

	// Processo de ALTA prioridade para o recurso (Priority 10)
	taskHigh := func(yield func(Command) bool) {
		GetProcess(&pHigh, yield)
		yield(WaitTime(2 * time.Second)) // Chega DEPOIS, no tempo 2s. Tenta adquirir e enfileira.

		res.Acquire(pHigh, 1, 10, yield) // Prio = 10 e fura a fila!

		logOrder = append(logOrder, "High")

		yield(WaitTime(1 * time.Second))
		res.Release(pHigh, 1)
	}

	env.AddTask(taskBlocker, 0)
	env.AddTask(taskLow, 0)
	env.AddTask(taskHigh, 0)

	env.StartSimul(20 * time.Second)

	if len(logOrder) != 2 {
		t.Fatalf("esperava 2 logs finalizados, consegui %d", len(logOrder))
	}

	// Como o pHigh tinha prioridade 10 no Acquire do Resource, ele DEVE
	// ser escalonado para uso antes do pLow, não importa que o pLow tenha chegado primeiro.
	if logOrder[0] != "High" {
		t.Errorf("expected pHigh to win the resource priority dispute, got %s first", logOrder[0])
	}
	if logOrder[1] != "Low" {
		t.Errorf("expected pLow to execute after pHigh, got %s second", logOrder[1])
	}
}

func TestResource3(t *testing.T) {
	env := NewEnvironment()
	res := env.NewResource("Printer", 2) // Capacidade 1

	for i := 0; i < 100; i++ {
		task := func(yield func(Command) bool) {
			var p *Process
			GetProcess(&p, yield)
			res.Acquire(p, 1, 1, yield)
			yield(WaitTime(1 * time.Second))
			res.Release(p, 1)
		}
		env.AddTask(task, 0)
	}
	env.StartSimul(60 * time.Second)
	if env.GetTime() != 50*time.Second {
		t.Errorf("expected total simulation time 5s, got %v", env.GetTime())
	}

}
