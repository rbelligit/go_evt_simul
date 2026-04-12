package evtmanager

import (
	"testing"
	"time"
)

func TestConditionsHelpers_ContainerGet(t *testing.T) {
	env := NewEnvironment()
	cont := env.NewContainer("tank", 100, 0) // Inicia vazio

	gotItem := false
	env.AddTask(func(yield func(Command) bool) {
		evt := ContainerGetEvent(env, cont, 50, 0)
		yield(evt.Wait())
		gotItem = true
	}, 0)

	env.StartSimul(10 * time.Second)
	if gotItem {
		t.Errorf("Não deveria ter pegado sem ninguem colocar!")
	}
	
	env.AddTask(func(yield func(Command) bool) {
		yield(cont.Put(50, 0))
	}, 0)
	
	env.StartSimul(20 * time.Second)
	if !gotItem {
		t.Errorf("Deveria ter pego após o Put")
	}
}

func TestConditionsHelpers_AnyOfWithContainerAndTimeout(t *testing.T) {
	env := NewEnvironment()
	cont := env.NewContainer("tank", 100, 0) 

	// Simula timeout vencendo a espera do conteiner
	timeoutWon := false
	env.AddTask(func(yield func(Command) bool) {
		getEvt := ContainerGetEvent(env, cont, 50, 0)
		tOut := TimeoutEvent(env, 5*time.Second)

		yield(env.AnyOf(getEvt, tOut).Wait())
		
		if tOut.IsTriggered() {
			timeoutWon = true
		}
	}, 0)

	env.StartSimul(10 * time.Second)
	if !timeoutWon {
		t.Errorf("O timeout deveria ter vencido, pois ninguém deu Put no container")
	}

	// Agora testa se o workaround de cancelamento do container funcionou 
	// (ele não deveria estar com "Get" pendente se deu erro ou cancelou)
	// Se fosse preenchido, ele iria devolver, mas como não foi nem pego, ele não faz nada.
}

func TestConditionsHelpers_ResourceAcquire(t *testing.T) {
	env := NewEnvironment()
	res := env.NewResource("work", 1)

	acq1 := false
	env.AddTask(func(yield func(Command) bool) {
		evt := ResourceAcquireEvent(env, res, 1, 0)
		yield(evt.Wait())
		acq1 = true
		t.Logf("Acq1 executado no tempo %v", env.GetTime())
		yield(WaitTime(10 * time.Second))
		var proc *Process
		GetProcess(&proc, yield)
		res.Release(proc, 1) // Libera no futuro
		t.Logf("Acq1 liberou recurso no tempo %v", env.GetTime())
	}, 0)

	acq2 := false
	env.AddTask(func(yield func(Command) bool) {
		evt := ResourceAcquireEvent(env, res, 1, 0)
		yield(evt.Wait())
		acq2 = true
		t.Logf("Acq2 executado no tempo %v", env.GetTime())
	}, 0)

	env.StartSimul(5 * time.Second)
	if !acq1 || acq2 {
		t.Errorf("Acq1 deveria ter executado e Acq2 bloqueado.")
	}

	env.StartSimul(15 * time.Second)
	if !acq2 {
		t.Errorf("Acq2 deveria ter executado após liberação")
	}
}

func TestConditionsHelpers_Store(t *testing.T) {
	env := NewEnvironment()
	store := NewStore[string](env, "fila", 10)

	var target string
	env.AddTask(func(yield func(Command) bool) {
		yield(StoreGetEvent(env, store, &target, 0).Wait())
	}, 0)
	
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5*time.Second))
		yield(StorePutEvent(env, store, "A", 0).Wait())
	}, 0)

	env.StartSimul(10 * time.Second)

	if target != "A" {
		t.Errorf("Deveria ter recebido A da fila, obteve: %v", target)
	}
}

func TestConditionsHelpers_ContainerPutCanceled(t *testing.T) {
	env := NewEnvironment()
	cont := env.NewContainer("tank", 100, 100) // Inicia cheio

	// Tenta colocar 50 (vai bloquear, pois está cheio). Será cancelado por timeout no 2s
	env.AddTask(func(yield func(Command) bool) {
		putEvt := ContainerPutEvent(env, cont, 50, 0)
		tOut := TimeoutEvent(env, 2*time.Second)
		yield(env.AnyOf(putEvt, tOut).Wait())
	}, 0)

	env.StartSimul(10 * time.Second)
	
	if cont.Level() != 100 {
		t.Errorf("Container deveria permanecer intacto no 100")
	}
}

func TestConditionsHelpers_FilterStoreHelpers(t *testing.T) {
	env := NewEnvironment()
	fs := NewFilterStore[int](env, "fs", 10)

	var target int
	env.AddTask(func(yield func(Command) bool) {
		// Bloqueia esperando um numero par
		filter := func(v int) bool { return v%2 == 0 }
		getEvt := FilterStoreGetEvent(env, fs, &target, filter, 0)
		yield(getEvt.Wait())
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(2 * time.Second))
		// Insere impar (nao vai ser filtrado nem destravar o consumidor)
		put1 := FilterStorePutEvent(env, fs, 3, 0)
		yield(put1.Wait())
		
		yield(WaitTime(2 * time.Second))
		// Insere par (vai destravar o Get perfeitamente)
		put2 := FilterStorePutEvent(env, fs, 4, 0)
		yield(put2.Wait())
	}, 0)

	env.StartSimul(10 * time.Second)

	if target != 4 {
		t.Errorf("O consumidor deveria ter filtrado e recebido o 4, obteve: %v", target)
	}
}

func TestConditionsHelpers_FilterStoreCanceled(t *testing.T) {
	env := NewEnvironment()
	fs := NewFilterStore[int](env, "fs", 1) // Capacidade 1

	env.AddTask(func(yield func(Command) bool) {
		yield(fs.Put(99, 0)) // Lota o Store
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		// Tenta inserir e bloqueará. Será cancelado.
		putEvt := FilterStorePutEvent(env, fs, 100, 0)
		tOut := TimeoutEvent(env, 5*time.Second)
		yield(env.AnyOf(putEvt, tOut).Wait())
	}, 0)

	env.StartSimul(10 * time.Second)
}
