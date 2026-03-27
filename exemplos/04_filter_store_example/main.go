package main

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

type LogMsg struct {
	Level string
	Text  string
}

func main() {
	env := evtmanager.NewEnvironment()

	// Cria um FilterStore para mensagens de log com capacidade 10
	logBus := evtmanager.NewFilterStore[*LogMsg](env, "LogBus", 10)

	// Produtor: Gera logs de diferentes níveis
	producer := func(yield func(evtmanager.Command) bool) {
		msgs := []LogMsg{
			{Level: "INFO", Text: "Sistema iniciado"},
			{Level: "DEBUG", Text: "Carregando módulos..."},
			{Level: "ERROR", Text: "Falha ao conectar no DB"},
			{Level: "INFO", Text: "Usuário logado"},
			{Level: "ERROR", Text: "Timeout requisição API"},
		}

		for _, m := range msgs {
			yield(evtmanager.WaitTime(2 * time.Second))
			msgCopy := m // Cópia para o ponteiro
			fmt.Printf("[Tempo %v] Gerando log: [%s] %s\n", env.GetTime(), msgCopy.Level, msgCopy.Text)
			yield(logBus.Put(&msgCopy, 0))
		}
	}

	// Consumidor 1: Apenas lê logs de ERROR
	errorConsumer := func(yield func(evtmanager.Command) bool) {
		for {
			var msg *LogMsg
			filterError := func(m *LogMsg) bool { return m.Level == "ERROR" }
			
			// Fica bloqueado até chegar um ERROR
			yield(logBus.Get(&msg, filterError, 0))
			
			fmt.Printf("[Tempo %v] 🚨 ALERTA CRÍTICO: %s\n", env.GetTime(), msg.Text)
		}
	}

	// Consumidor 2: Apenas lê logs INFO
	infoConsumer := func(yield func(evtmanager.Command) bool) {
		for {
			var msg *LogMsg
			filterInfo := func(m *LogMsg) bool { return m.Level == "INFO" }
			
			// Fica bloqueado até chegar um INFO
			yield(logBus.Get(&msg, filterInfo, 0))
			
			fmt.Printf("[Tempo %v] ℹ️ INFORMAÇÃO: %s\n", env.GetTime(), msg.Text)
		}
	}

	env.AddTask(producer, 0)
	env.AddTask(errorConsumer, 0)
	env.AddTask(infoConsumer, 0)

	fmt.Println("Iniciando Simulação do FilterStore...")
	// Inicia simulação com limite de 15 segundos
	env.StartSimul(15 * time.Second)
	fmt.Println("Simulação Concluída!")
}
