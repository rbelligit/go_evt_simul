package evtmanager

import (
	"testing"
	"time"
)

// Msg é uma estrutura auxiliar para simular pacotes de rede ou mensagens de sistema.
type Msg struct {
	Type string
	ID   int
}

// TestFilterStoreBasic testa se um processo consegue pegar um item específico
// do meio do buffer, ignorando os outros que não combinam com o filtro.
func TestFilterStoreBasic(t *testing.T) {
	env := NewEnvironment()
	store := NewFilterStore[*Msg](env, "NetworkBus", 10)

	// Produtor: Coloca 3 mensagens diferentes no tempo 0
	env.AddTask(func(yield func(Command) bool) {
		yield(store.Put(&Msg{Type: "PING", ID: 1}, 0))
		yield(store.Put(&Msg{Type: "DADO", ID: 2}, 0))
		yield(store.Put(&Msg{Type: "ACK", ID: 3}, 0))
	}, 0)

	var msgRecebida *Msg

	// Consumidor: Quer EXATAMENTE o pacote "ACK", ignorando "PING" e "DADO"
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5))

		filtroAck := func(m *Msg) bool { return m.Type == "ACK" }
		yield(store.Get(&msgRecebida, filtroAck, 0))
	}, 0)

	env.StartSimul(10)

	if msgRecebida == nil || msgRecebida.Type != "ACK" {
		t.Errorf("O consumidor deveria ter pegado o ACK, mas pegou: %v", msgRecebida)
	}
	// O buffer original tinha 3 itens. Tiramos 1. Sobram 2.
	if len(store.items) != 2 {
		t.Errorf("Deveriam sobrar 2 itens no buffer (PING e DADO), mas sobraram: %d", len(store.items))
	}
	// Confirma que os itens restantes são os corretos (FIFO mantido para o resto)
	if store.items[0].Type != "PING" || store.items[1].Type != "DADO" {
		t.Errorf("A ordem dos itens restantes foi corrompida: %v", store.items)
	}
}

// TestFilterStoreBlockingGet testa o cenário onde o consumidor chega antes da mensagem.
// Ele deve ficar suspenso até que uma mensagem que passe no filtro dele seja inserida.
func TestFilterStoreBlockingGet(t *testing.T) {
	env := NewEnvironment()
	store := NewFilterStore[*Msg](env, "TopicBus", 10)

	var msgRecebida *Msg
	var tempoRecebimento time.Duration

	// Consumidor: Chega no tempo 0 querendo um pacote "ALARME"
	env.AddTask(func(yield func(Command) bool) {
		filtroAlarme := func(m *Msg) bool { return m.Type == "ALARME" }
		yield(store.Get(&msgRecebida, filtroAlarme, 0))
		tempoRecebimento = env.GetTime()
	}, 0)

	// Produtor: Fica gerando pacotes. O Alarme só sai no tempo 15.
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5))
		yield(store.Put(&Msg{Type: "INFO", ID: 1}, 0)) // Consumidor ignora

		yield(WaitTime(5))
		yield(store.Put(&Msg{Type: "INFO", ID: 2}, 0)) // Consumidor ignora

		yield(WaitTime(5))
		yield(store.Put(&Msg{Type: "ALARME", ID: 99}, 0)) // Consumidor deve acordar aqui!
	}, 0)

	env.StartSimul(20)

	if tempoRecebimento != 15 {
		t.Errorf("O consumidor deveria ter acordado no tempo 15, mas acordou em %d", tempoRecebimento)
	}
	if msgRecebida.ID != 99 {
		t.Errorf("A mensagem recebida deveria ser o ID 99, foi %d", msgRecebida.ID)
	}
}

// TestFilterStoreBlockingPut testa o Backpressure quando o buffer enche.
func TestFilterStoreBlockingPut(t *testing.T) {
	env := NewEnvironment()
	store := NewFilterStore[*Msg](env, "SmallBuffer", 1) // Capacidade apenas 1!

	var tempoSucesso time.Duration

	// Produtor: Tenta colocar dois pacotes. O segundo vai bloquear.
	env.AddTask(func(yield func(Command) bool) {
		yield(store.Put(&Msg{Type: "A", ID: 1}, 0))
		yield(store.Put(&Msg{Type: "B", ID: 2}, 0)) // Bloqueia aqui!
		tempoSucesso = env.GetTime()
	}, 0)

	// Consumidor: Demora 10 segundos para limpar o buffer
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(10))
		var m *Msg
		yield(store.Get(&m, nil, 0)) // Passar 'nil' funciona como um Store normal (pega qualquer um)
	}, 0)

	env.StartSimul(20)

	// O produtor só consegue empurrar a mensagem B após o tempo 10
	if tempoSucesso != 10 {
		t.Errorf("O produtor deveria ter sido desbloqueado no tempo 10, mas foi em %d", tempoSucesso)
	}
	// O buffer deve conter a mensagem B agora
	if len(store.items) != 1 || store.items[0].Type != "B" {
		t.Errorf("O buffer deveria conter a mensagem B, mas tem: %v", store.items)
	}
}

// TestFilterStoreMultipleConsumers testa múltiplos consumidores bloqueados
// esperando por mensagens diferentes. O produtor deve acordar o consumidor correto.
func TestFilterStoreMultipleConsumers(t *testing.T) {
	env := NewEnvironment()
	store := NewFilterStore[*Msg](env, "EventBus", 10)

	logEventos := make([]string, 0)

	// Consumidor 1 (Espera por "AZUL")
	env.AddTask(func(yield func(Command) bool) {
		var m *Msg
		filtroAzul := func(m *Msg) bool { return m.Type == "AZUL" }
		yield(store.Get(&m, filtroAzul, 0))
		logEventos = append(logEventos, "Recebeu Azul")
	}, 0)

	// Consumidor 2 (Espera por "VERMELHO")
	env.AddTask(func(yield func(Command) bool) {
		var m *Msg
		filtroVermelho := func(m *Msg) bool { return m.Type == "VERMELHO" }
		yield(store.Get(&m, filtroVermelho, 0))
		logEventos = append(logEventos, "Recebeu Vermelho")
	}, 0)

	// Produtor
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5))
		// Publica vermelho primeiro. Deverá acordar o Consumidor 2, ignorando o Consumidor 1.
		yield(store.Put(&Msg{Type: "VERMELHO", ID: 1}, 0))

		yield(WaitTime(5))
		// Publica azul depois. Deverá acordar o Consumidor 1.
		yield(store.Put(&Msg{Type: "AZUL", ID: 2}, 0))
	}, 0)

	env.StartSimul(15)

	if len(logEventos) != 2 {
		t.Fatalf("Esperava 2 eventos processados, ocorreram %d", len(logEventos))
	}
	if logEventos[0] != "Recebeu Vermelho" || logEventos[1] != "Recebeu Azul" {
		t.Errorf("A ordem de recebimento está errada: %v", logEventos)
	}
}
