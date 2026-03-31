package main

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

type Package struct {
	ID   int
	Data string
}

func main() {
	env := evtmanager.NewEnvironment()

	// Store with capacity 2. If it's full, Put will block.
	// If it's empty, Get will block.
	store := evtmanager.NewStore[Package](env, "Warehouse", 3)

	producer := func(yield func(evtmanager.Command) bool) {
		for i := 1; i <= 5; i++ {
			pkg := Package{ID: i, Data: fmt.Sprintf("Data-%d", i)}
			fmt.Printf("[Time %v] Producer wants to store package %d.\n", env.GetTime(), pkg.ID)

			// Priority 10
			yield(store.Put(pkg, 10))

			fmt.Printf("[Time %v] Producer successfully stored package %d.\n", env.GetTime(), pkg.ID)
			yield(evtmanager.WaitTime(1 * time.Second))
		}
	}

	consumer := func(id int) evtmanager.TaskSimul {
		return func(yield func(evtmanager.Command) bool) {
			for i := 0; i < 3; i++ { // Each consumer wants 3 packages
				// Consumer takes longer to process, causing the store to fill up
				yield(evtmanager.WaitTime(3 * time.Second))

				var pkg Package
				fmt.Printf("[Time %v] Consumer %d waiting for package.\n", env.GetTime(), id)

				// Priority 10
				yield(store.Get(&pkg, 10))

				fmt.Printf("[Time %v] Consumer %d got package %d: %s.\n", env.GetTime(), id, pkg.ID, pkg.Data)
			}
		}
	}

	env.AddTask(producer, 0)
	// Two consumers means 6 packages needed, but producer only makes 5.
	// We'll see how it stops when simulation time limits out or ends.
	env.AddTask(consumer(1), 0)
	env.AddTask(consumer(2), 0)

	fmt.Println("Starting Store Simulation...")

	// Start simulation. We cap at 30 seconds.
	env.StartSimul(30 * time.Second)
	
	fmt.Println("Simulation Complete!")
	fmt.Printf("Total Simulated Time: %v\n", env.GetTime())
}
