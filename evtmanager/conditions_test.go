package evtmanager

import (
	"testing"
	"time"
)

func TestConditions_AnyOf(t *testing.T) {
	env := NewEnvironment()
	
	e1 := env.NewEvent("E1")
	e1Canceled := false
	e1.AddCancelCallback(func() { e1Canceled = true })

	e2 := env.NewEvent("E2")
	e2Canceled := false
	e2.AddCancelCallback(func() { e2Canceled = true })

	var tempoAcordou time.Duration

	env.AddTask(func(yield func(Command) bool) {
		yield(env.AnyOf(e1, e2).Wait())
		tempoAcordou = env.GetTime()
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5 * time.Second))
		yield(e1.Succeed())
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(10 * time.Second))
		yield(e2.Succeed())
	}, 0)

	env.StartSimul(15 * time.Second)

	if tempoAcordou != 5*time.Second {
		t.Errorf("AnyOf deveria acordar no 5s, mas acordou %v", tempoAcordou)
	}
	if !e2Canceled {
		t.Errorf("O evento E2 deveria ter sido cancelado porque E1 disparou primeiro")
	}
	if e1Canceled {
		t.Errorf("O evento E1 não deveria ter sido cancelado (ele disparou)")
	}
}

func TestConditions_AllOf(t *testing.T) {
	env := NewEnvironment()
	e1 := env.NewEvent("E1")
	e2 := env.NewEvent("E2")

	var tempoAcordou time.Duration

	env.AddTask(func(yield func(Command) bool) {
		yield(env.AllOf(e1, e2).Wait())
		tempoAcordou = env.GetTime()
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(3 * time.Second))
		yield(e1.Succeed())
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(6 * time.Second))
		yield(e2.Succeed())
	}, 0)

	env.StartSimul(10 * time.Second)

	if tempoAcordou != 6*time.Second {
		t.Errorf("AllOf deveria aguardar todos e acordar no 6s, acordou %v", tempoAcordou)
	}
}

func TestConditions_Timeout(t *testing.T) {
	env := NewEnvironment()

	var tempoAcordou time.Duration
	timeoutFired := false

	env.AddTask(func(yield func(Command) bool) {
		timeoutEvt := env.Timeout(4 * time.Second)
		timeoutEvt.AddCallback(func() {
			timeoutFired = true
		})
		yield(timeoutEvt.Wait())
		tempoAcordou = env.GetTime()
	}, 0)

	env.StartSimul(10 * time.Second)

	if tempoAcordou != 4*time.Second {
		t.Errorf("Deveria acordar no tempo 4s, mas acordou %v", tempoAcordou)
	}
	if !timeoutFired {
		t.Errorf("O Callback de sucesso do Timeout deveria ter sido chamado")
	}
}

func TestConditions_TimeoutCanceled(t *testing.T) {
	env := NewEnvironment()

	e1 := env.NewEvent("E1")
	timeoutFired := false

	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(2 * time.Second))
		yield(e1.Succeed())
	}, 0)

	env.AddTask(func(yield func(Command) bool) {
		tOut := env.Timeout(5 * time.Second)
		tOut.AddCallback(func() {
			timeoutFired = true
		})
		yield(env.AnyOf(e1, tOut).Wait())
	}, 0)

	env.StartSimul(10 * time.Second)

	if timeoutFired {
		t.Errorf("O timeout de 5s não deveria disparar, pois foi cancelado pelo E1 no 2s")
	}
}
