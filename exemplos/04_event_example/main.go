package main

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

func main() {
	env := evtmanager.NewEnvironment()

	// Cria um novo evento que servirá de "Gatilho" para os trabalhadores.
	// Por exemplo, uma remessa de peças chegou na fábrica.
	pecasChegaram := env.NewEvent("Pecas Chegaram")

	trabalhador := func(id int) evtmanager.TaskSimul {
		return func(yield func(evtmanager.Command) bool) {
			var p *evtmanager.Process
			evtmanager.GetProcess(&p, yield)

			fmt.Printf("[Tempo %v] Trabalhador %d aguardando as peças chegarem.\n", env.GetTime(), id)

			// O processo vai suspender sua execução aqui até que o evento `Succeed` seja chamado.
			yield(pecasChegaram.Wait())

			fmt.Printf("[Tempo %v] Trabalhador %d viu que as peças chegaram e começou a trabalhar!\n", env.GetTime(), id)

			// Trabalhador faz seu serviço
			yield(evtmanager.WaitTime(time.Duration(id) * time.Second))
			
			fmt.Printf("[Tempo %v] Trabalhador %d finalizou sua parte.\n", env.GetTime(), id)
		}
	}

	for i := 1; i <= 3; i++ {
		env.AddTask(trabalhador(i), 0)
	}

	fornecedor := func(yield func(evtmanager.Command) bool) {
		fmt.Printf("[Tempo %v] Fornecedor saiu para entregar as peças.\n", env.GetTime())
		
		yield(evtmanager.WaitTime(10 * time.Second))
		
		fmt.Printf("[Tempo %v] Fornecedor chegou na fábrica com as peças! Disparando evento...\n", env.GetTime())
		
		// Acorda todos os trabalhadores que estavam aguardando o evento simultaneamente!
		yield(pecasChegaram.Succeed())
	}

	env.AddTask(fornecedor, 0)

	fmt.Println("Iniciando Simulação de Eventos...")
	env.StartSimul(30 * time.Second)
	fmt.Println("Simulação Concluída!")
}
