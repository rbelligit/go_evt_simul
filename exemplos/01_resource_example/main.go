package main

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

func main() {
	// Create a new simulation environment
	env := evtmanager.NewEnvironment()

	// Create a limited resource (e.g., 2 cashiers in a bank)
	cashiers := env.NewResource("Cashiers", 2)

	// Define a customer process
	customer := func(id int) evtmanager.TaskSimul {
		return func(yield func(evtmanager.Command) bool) {
			var p *evtmanager.Process
			evtmanager.GetProcess(&p, yield)

			fmt.Printf("[Tempo %v] Cliente %d chegou no banco.\n", env.GetTime(), id)

			// Try to acquire 1 cashier, with priority 1
			// This will block the customer process if both cashiers are busy
			cashiers.Acquire(p, 1, 1, yield)

			fmt.Printf("[Tempo %v] Cliente %d começou a ser atendido.\n", env.GetTime(), id)

			// Simulate the time it takes to serve the customer (e.g., 5 seconds)
			yield(evtmanager.WaitTime(5 * time.Second))

			// Release the cashier
			cashiers.Release(p, 1)

			fmt.Printf("[Tempo %v] Cliente %d terminou e saiu do banco.\n", env.GetTime(), id)
		}
	}

	// Add customers arriving at different times
	for i := 1; i <= 5; i++ {
		// Capture the loop variable
		id := i
		// Create a process that will wait before adding the customer task
		spawner := func(yield func(evtmanager.Command) bool) {
			// Wait i seconds before the customer arrives
			yield(evtmanager.WaitTime(time.Duration(id) * time.Second))
			// Add the actual customer process right now (relative to when spawner finishes waiting)
			env.AddTask(customer(id), 0)
		}
		env.AddTask(spawner, 0)
	}

	fmt.Println("Inciando a simulação...")
	
	// Start the simulation with a maximum time limit
	env.StartSimul(30 * time.Second)

	fmt.Println("Simulação concluída!")
	fmt.Printf("Tempo total simulado: %v\n", env.GetTime())
}
