package evtmanager

import (
	"testing"
	"time"
)

func TestStoreProducerConsumerFast(t *testing.T) {
	env := NewEnvironment()
	store := NewStore[string](env, "mensagens", 3)

	var recebidas []string

	produtor := func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)

		yield(WaitTime(time.Second))
		yield(store.Put("olá", 10))

		yield(WaitTime(time.Second))
		yield(store.Put("mundo", 10))
	}

	consumidor := func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)
		
		var msg string
		yield(store.Get(&msg, 10))
		recebidas = append(recebidas, msg)

		yield(store.Get(&msg, 10))
		recebidas = append(recebidas, msg)
	}

	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(5 * time.Second)

	if len(recebidas) != 2 || recebidas[0] != "olá" || recebidas[1] != "mundo" {
		t.Errorf("Falha na entrega das mensagens, recebido: %v", recebidas)
	}
}

func TestStoreProducerSuspendedWhenFull(t *testing.T) {
	env := NewEnvironment()
	store := NewStore[int](env, "buffer", 1) // capacidade 1, logo o segundo item bloqueia

	produtor := func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)

		yield(store.Put(100, 10))
		yield(store.Put(200, 10)) // Aqui deve bloquear até o consumidor tirar 1
		
		if env.GetTime() != 5*time.Second {
			t.Errorf("Tempo do produtor incorreto ao voltar, obteve %v", env.GetTime())
		}
	}

	consumidor := func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)

		yield(WaitTime(5 * time.Second)) // Fica inativo por 5 seg

		var v int
		yield(store.Get(&v, 10)) // tira o 100, o que fará o produtor colocar o 200
		
		if v != 100 {
			t.Errorf("Valor tirado incorreto: %d", v)
		}
	}

	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(10 * time.Second)
}

func TestStoreCapacityZero(t *testing.T) {
	env := NewEnvironment()
	store := NewStore[int](env, "sync_buffer", 0) // capacidade 0

	produtor := func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)

		// O produtor vai bloquear aqui imediatamente, pois capacidade 0
		yield(store.Put(42, 10))
		
		if env.GetTime() != 3*time.Second {
			t.Errorf("Produtor liberado no tempo errado: %v", env.GetTime())
		}
	}

	consumidor := func(yield func(Command) bool) {
		var p *Process
		GetProcess(&p, yield)

		yield(WaitTime(3 * time.Second)) 
		var val int
		// O consumidor fará hand-off direto
		yield(store.Get(&val, 10))

		if val != 42 {
			t.Errorf("Consumidor não recebeu: %v", val)
		}
	}

	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(5 * time.Second)
}
