package evtmanager

import (
	"testing"
	"time"
)

func TestEvent_WaitAndSucceed(t *testing.T) {
	env := NewEnvironment()
	evt := env.NewEvent("TesteEvent")

	var tempoAcordou time.Duration
	processoEsperando := false

	// Processo A: Aguarda o evento
	env.AddTask(func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)
		
		processoEsperando = true
		
		// Aguarda o evento acontecer
		yield(evt.Wait())
		
		tempoAcordou = env.GetTime()
	}, 0)

	// Processo B: Dispara o evento no tempo 10
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(10 * time.Second))
		yield(evt.Succeed())
	}, 0)

	env.StartSimul(20 * time.Second)

	if !processoEsperando {
		t.Errorf("Processo não iniciou")
	}

	if tempoAcordou != 10*time.Second {
		t.Errorf("Processo deveria acordar no tempo 10s, mas acordou em %v", tempoAcordou)
	}
}

func TestEvent_WaitAlreadySucceeded(t *testing.T) {
	env := NewEnvironment()
	evt := env.NewEvent("TesteEventJáDisparado")

	var tempoAcordou time.Duration

	// Processo A disparou o evento no tempo 5
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5 * time.Second))
		yield(evt.Succeed())
	}, 0)

	// Processo B vai tentar aguardar no tempo 15, mas o evento já passou
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(15 * time.Second))
		yield(evt.Wait())
		tempoAcordou = env.GetTime()
	}, 0)

	env.StartSimul(20 * time.Second)

	if tempoAcordou != 15*time.Second {
		t.Errorf("Processo deveria continuar no tempo 15s sem bloquear, mas continuou em %v", tempoAcordou)
	}
}

func TestEvent_MultipleWaiters(t *testing.T) {
	env := NewEnvironment()
	evt := env.NewEvent("BroadcastEvent")

	contadorA := 0
	contadorB := 0

	// Processo A: Aguarda o evento
	env.AddTask(func(yield func(Command) bool) {
		yield(evt.Wait())
		contadorA++
	}, 0)

	// Processo B: Aguarda o evento
	env.AddTask(func(yield func(Command) bool) {
		yield(evt.Wait())
		contadorB++
	}, 0)

	// Processo C: Dispara o evento no tempo 5
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5 * time.Second))
		yield(evt.Succeed())
	}, 0)

	env.StartSimul(10 * time.Second)

	if contadorA != 1 {
		t.Errorf("Processo A não foi acordado")
	}
	if contadorB != 1 {
		t.Errorf("Processo B não foi acordado")
	}
}
