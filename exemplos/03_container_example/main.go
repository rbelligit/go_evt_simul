package main

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

func main() {
	env := evtmanager.NewEnvironment()

	// Container com capacidade para 1000 litros, iniciando com 200 litros.
	reservatorio := env.NewContainer("Reservatório", 1000, 200)

	// Consumidor (ex: uma plantação sendo irrigada)
	irrigacao := func(yield func(evtmanager.Command) bool) {
		for i := 1; i <= 3; i++ {
			fmt.Printf("[Tempo %v] Iniciando irrigação %d. Nível atual: %.2f\n", 
				env.GetTime(), i, reservatorio.Level())
			
			// Tenta consumir 150 litros
			// Vai bloquear se não tiver o suficiente
			yield(reservatorio.Get(150, 1))
			
			fmt.Printf("[Tempo %v] Irrigação %d concluída! Nível atual: %.2f\n", 
				env.GetTime(), i, reservatorio.Level())
			
			// Espera 5 segundos até a próxima irrigação
			yield(evtmanager.WaitTime(5 * time.Second))
		}
	}

	// Fornecedor (ex: caminhão pipa)
	caminhaoPipa := func(yield func(evtmanager.Command) bool) {
		// Demora 8 segundos para chegar
		yield(evtmanager.WaitTime(8 * time.Second))
		
		fmt.Printf("[Tempo %v] Caminhão pipa chegou com 500 litros!\n", env.GetTime())
		
		// Abastece o reservatório
		yield(reservatorio.Put(500, 1))
		
		fmt.Printf("[Tempo %v] Caminhão pipa descarregou. Nível atual: %.2f\n", env.GetTime(), reservatorio.Level())
	}

	env.AddTask(irrigacao, 0)
	env.AddTask(caminhaoPipa, 0)

	fmt.Println("Iniciando Simulação de Container...")
	env.StartSimul(30 * time.Second)
	fmt.Println("Simulação Concluída!")
	fmt.Printf("Tempo total: %v, Nível final: %.2f\n", env.GetTime(), reservatorio.Level())
}
