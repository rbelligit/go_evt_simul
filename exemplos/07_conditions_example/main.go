package main

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

func main() {
	env := evtmanager.NewEnvironment()

	fmt.Println("============== EXEMPLO DE CONDIÇÕES (ANYOF, TIMEOUT) ==============")

	// Vamos simular dois trabalhadores tentando obter uso de uma mesma máquina especial (Resource).
	// Mas eles são impacientes! Se a máquina demorar muito, eles desistem (Timeout).
	maquina := env.NewResource("Maquina de Solda", 1)

	// O Trabalhador 1 é paciente e chega primeiro (No tempo 0)
	trabalhador1 := func(yield func(evtmanager.Command) bool) {
		fmt.Printf("[Tempo %v] Trabalhador 1 chega na fábrica e vai usar a máquina.\n", env.GetTime())
		
		// Espera a máquina ficar disponível 
		acqEvt := evtmanager.ResourceAcquireEvent(env, maquina, 1, 0)
		if !yield(acqEvt.Wait()) {
			return
		}

		fmt.Printf("[Tempo %v] Trabalhador 1 pegou a máquina! Vai trabalhar por 10 segundos.\n", env.GetTime())
		if !yield(evtmanager.WaitTime(10 * time.Second)) {
			return
		}
		
		fmt.Printf("[Tempo %v] Trabalhador 1 terminou. Liberando a máquina.\n", env.GetTime())
		
		var p *evtmanager.Process
		evtmanager.GetProcess(&p, yield)
		maquina.Release(p, 1)
	}

	// O Trabalhador 2 chega logo depois, mas é impaciente.
	trabalhador2 := func(id int, paciencia time.Duration) evtmanager.TaskSimul {
		return func(yield func(evtmanager.Command) bool) {
			// Começa a tentar no tempo 2s
			if !yield(evtmanager.WaitTime(2 * time.Second)) {
				return
			}
			
			fmt.Printf("[Tempo %v] Trabalhador %d chegou e quer a máquina. Só vai aguardar por %v!\n", env.GetTime(), id, paciencia)

			// 1. Cria o evento de aguardar a máquina
			esperaMaquina := evtmanager.ResourceAcquireEvent(env, maquina, 1, 0)
			
			// 2. Cria o evento do relógio correndo (Timeout)
			esperaRelogio := evtmanager.TimeoutEvent(env, paciencia)

			// 3. Usa o Meta-Evento AnyOf para o processo suspender e aguardar "O que acontecer primeiro"
			// MAGIA: Quando um dos eventos ocorrer, o outro será cancelado automaticamente e de forma limpa!
			if !yield(env.AnyOf(esperaMaquina, esperaRelogio).Wait()) {
				return
			}

			// Verifica qual deles disparou para decidir o que aconteceu
			if esperaRelogio.IsTriggered() {
				fmt.Printf("[Tempo %v] Trabalhador %d diz: O tempo acabou! Desisto de usar a máquina e vou almoçar.\n", env.GetTime(), id)
			} else {
				fmt.Printf("[Tempo %v] Trabalhador %d diz: Aleluia! A máquina liberou a tempo. Trabalhando...\n", env.GetTime(), id)
				
				if !yield(evtmanager.WaitTime(5 * time.Second)) {
					return
				}
				var p *evtmanager.Process
				evtmanager.GetProcess(&p, yield)
				maquina.Release(p, 1)
				fmt.Printf("[Tempo %v] Trabalhador %d terminou.\n", env.GetTime(), id)
			}
		}
	}

	env.AddTask(trabalhador1, 0)
	
	// O Trabalhador 2 tem 5 segundos de paciência. Como chegou no tempo 2, vai desistir no tempo 7.
	env.AddTask(trabalhador2(2, 5*time.Second), 0)

	// O Trabalhador 3 tem 20 segundos de paciência. Ele vai conseguir a máquina depois que o T1 liberar no 10.
	env.AddTask(trabalhador2(3, 20*time.Second), 0)

	env.StartSimul(30 * time.Second)
	fmt.Println("============== SIMULAÇÃO CONCLUÍDA ==============")
}
