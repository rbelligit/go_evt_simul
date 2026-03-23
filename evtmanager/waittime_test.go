package evtmanager

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestWaitTime(t *testing.T) {
	env := NewEnvironment()

	// Create a task that records its start time, waits for 5.0 time units, then records its end time
	task := func(yield func(Command) bool) {
		var i int
		for i = 0; i < 10; i++ {
			sts := yield(WaitTime(1 * time.Second))
			if !sts {
				break
			}
			expected := time.Duration(i+1) * time.Second
			if env.GetTime() != expected {
				t.Errorf("Expected time %v, got %v", expected, env.GetTime())
			}
			fmt.Fprintf(os.Stderr, "i=%d, time=%v\n", i, env.GetTime())
		}
		if i != 10 {
			t.Errorf("Expected 10 execution steps, got %d", i)
		}
	}

	env.AddTask(task, 0)
	err := env.StartSimul(10 * time.Second)

	if env.GetTime() != 10*time.Second {
		t.Errorf("Expected final time 10s, got %v", env.GetTime())
	}

	if err != nil {
		t.Fatalf("StartSimul returned error: %v", err)
	}

}
