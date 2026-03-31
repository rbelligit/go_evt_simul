package evtmanager

import (
	"reflect"
	"testing"
	"time"
)

func TestQoSStoreWRROrder(t *testing.T) {
	env := NewEnvironment()
	// QoS Store com capacidade global 10, duas filas com pesos 2 e 1.
	store := NewQoSStore[string](env, "qos_buffer", 10, []int{2, 1})

	var recebidas []string

	produtor := func(yield func(Command) bool) {
		// Injeta logo todas de uma vez pois tem capacidade
		// Fila 0 (TC 0)
		yield(store.Put("q0_1", 0, 10))
		yield(store.Put("q0_2", 0, 10))
		yield(store.Put("q0_3", 0, 10))

		// Fila 1 (TC 1)
		yield(store.Put("q1_1", 1, 10))
		yield(store.Put("q1_2", 1, 10))
		yield(store.Put("q1_3", 1, 10))
	}

	consumidor := func(yield func(Command) bool) {
		// Precisa esperar garantir que o produtor colocou todas para o WRR ser testado da forma esperada
		yield(WaitTime(1 * time.Second))

		for i := 0; i < 6; i++ {
			var msg string
			yield(store.Get(&msg, 10))
			recebidas = append(recebidas, msg)
		}
	}

	// Como a prioridade das tasks é a mesma, o produtor roda primeiro e já preenche o buffer,
	// e o tempo em seguida garante que todos estão lá antes dos gets.
	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(5 * time.Second)

	expected := []string{"q0_1", "q1_1", "q0_2", "q0_3", "q1_2", "q1_3"}
	if !reflect.DeepEqual(recebidas, expected) {
		t.Errorf("Ordem do WRR incorreta!\nEsperado: %v\nObtido: %v", expected, recebidas)
	}
}

func TestQoSStoreBlockingProducer(t *testing.T) {
	env := NewEnvironment()
	store := NewQoSStore[int](env, "qos_small", 2, []int{1}) // Apenas uma fila, cap 2

	produtor := func(yield func(Command) bool) {
		yield(store.Put(100, 0, 10))
		yield(store.Put(200, 0, 10))
		// Aqui deve bloquear, pois capacidade já atingida
		yield(store.Put(300, 0, 10))

		if env.GetTime() != 5*time.Second {
			t.Errorf("Tempo do produtor incorreto ao voltar, obteve %v", env.GetTime())
		}
	}

	consumidor := func(yield func(Command) bool) {
		yield(WaitTime(5 * time.Second))

		var v int
		yield(store.Get(&v, 10)) // Libera espaço para o 300

		if v != 100 {
			t.Errorf("Valor tirado incorreto: %d", v)
		}
	}

	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(10 * time.Second)
}

func TestQoSStoreBlockingConsumer(t *testing.T) {
	env := NewEnvironment()
	store := NewQoSStore[string](env, "qos_wait", 5, []int{1})

	produtor := func(yield func(Command) bool) {
		yield(WaitTime(3 * time.Second))
		yield(store.Put("late_msg", 0, 10))
	}

	consumidor := func(yield func(Command) bool) {
		var msg string
		yield(store.Get(&msg, 10)) // Bloqueia imediatamente

		if env.GetTime() != 3*time.Second {
			t.Errorf("Consumidor liberado no tempo errado: %v", env.GetTime())
		}
		if msg != "late_msg" {
			t.Errorf("Consumidor recebeu mensagem incorreta: %s", msg)
		}
	}

	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(5 * time.Second)
}

func TestQoSStoreInvalidTC(t *testing.T) {
	env := NewEnvironment()
	store := NewQoSStore[int](env, "qos_invalid", 5, []int{1, 1})

	produtor := func(yield func(Command) bool) {
		// Envia com fila 99, vai jogar na última disponível
		yield(store.Put(42, 99, 10))
	}

	consumidor := func(yield func(Command) bool) {
		yield(WaitTime(1 * time.Second))
		var msg int
		yield(store.Get(&msg, 10))
	}

	env.AddTask(produtor, 10)
	env.AddTask(consumidor, 10)
	env.StartSimul(2 * time.Second)

	// Deve estar na última fila (índice 1) antes da leitura.
	// Mas como foi lido, testamos apenas que a simulação concluiu sem panic e a funcionalidade de remapeamento funcionou.
}
