package evtmanager

import (
	"testing"
	"time"
)

// TestContainerBasic testa a adição e remoção simples sem bloqueios.
func TestContainerBasic(t *testing.T) {
	env := NewEnvironment()
	// Reservatório de 100 litros, começa vazio (0)
	tank := env.NewContainer("Tanque", 100, 0)

	// Caminhão: Coloca 60 litros no tempo 0
	env.AddTask(func(yield func(Command) bool) {
		yield(tank.Put(60, 0))
		if tank.Level() != 60 {
			t.Errorf("Esperava 60 no tanque, mas tem %f", tank.Level())
		}
	}, 0)

	// Carro: Retira 20 litros no tempo 5
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5))
		yield(tank.Get(20, 0))
		if tank.Level() != 40 {
			t.Errorf("Esperava 40 no tanque após retirada, mas tem %f", tank.Level())
		}
	}, 0)

	env.StartSimul(10)

	if tank.Level() != 40 {
		t.Errorf("Nível final incorreto. Esperava 40, obteve %f", tank.Level())
	}
}

// TestContainerBlockingGet testa se um processo fica suspenso aguardando material.
func TestContainerBlockingGet(t *testing.T) {
	env := NewEnvironment()
	tank := env.NewContainer("CaixaDagua", 500, 0)
	var tempoAbastecimento time.Duration

	// Casa: Tenta gastar 100L no tempo 0 (vai bloquear pois está vazio)
	env.AddTask(func(yield func(Command) bool) {
		yield(tank.Get(100, 0))
		tempoAbastecimento = env.GetTime() // Registra quando conseguiu a água
	}, 0)

	// Sabesp: Manda 200L apenas no tempo 12
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(12))
		yield(tank.Put(200, 0))
	}, 0)

	env.StartSimul(20)

	// A casa só deve ter conseguido a água no tempo 12
	if tempoAbastecimento != 12 {
		t.Errorf("A casa deveria receber água em T=12, mas recebeu em %d", tempoAbastecimento)
	}
	// Sobram 100L no tanque (200 - 100)
	if tank.Level() != 100 {
		t.Errorf("Esperava 100L sobrando, obteve %f", tank.Level())
	}
}

// TestContainerBlockingPut testa o Backpressure (bloqueio por falta de espaço).
func TestContainerBlockingPut(t *testing.T) {
	env := NewEnvironment()
	// Tanque de 100L, já começa CHEIO (100)
	tank := env.NewContainer("Silo", 100, 100)
	var tempoDescarga time.Duration

	// Fazendeiro: Tenta colocar mais 50L no tempo 0 (vai bloquear por falta de espaço)
	env.AddTask(func(yield func(Command) bool) {
		yield(tank.Put(50, 0))
		tempoDescarga = env.GetTime() // Registra quando conseguiu descarregar
	}, 0)

	// Caminhão: Leva 80L embora no tempo 15
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(15))
		yield(tank.Get(80, 0)) // Libera 80L de espaço
	}, 0)

	env.StartSimul(30)

	// O fazendeiro só deve ter conseguido colocar os 50L após o caminhão sair em T=15
	if tempoDescarga != 15 {
		t.Errorf("O fazendeiro deveria descarregar em T=15, mas foi em %d", tempoDescarga)
	}
	// Nível final = 100 (inicial) - 80 (caminhão) + 50 (fazendeiro) = 70
	if tank.Level() != 70 {
		t.Errorf("Nível final deveria ser 70, mas é %f", tank.Level())
	}
}

// TestContainerCascading Wakeup testa se um único Put grande acorda múltiplos processos Get.
func TestContainerCascading(t *testing.T) {
	env := NewEnvironment()
	tank := env.NewContainer("Bateria", 100, 0)

	carro1Acordou := false
	carro2Acordou := false

	// Carro 1 quer 30 (bloqueia)
	env.AddTask(func(yield func(Command) bool) {
		yield(tank.Get(30, 0))
		carro1Acordou = true
	}, 0)

	// Carro 2 quer 40 (bloqueia)
	env.AddTask(func(yield func(Command) bool) {
		yield(tank.Get(40, 0))
		carro2Acordou = true
	}, 0)

	// Carregador injeta 100 no tempo 5
	env.AddTask(func(yield func(Command) bool) {
		yield(WaitTime(5))
		yield(tank.Put(100, 0))
	}, 0)

	env.StartSimul(10)

	if !carro1Acordou || !carro2Acordou {
		t.Errorf("Ambos os carros deveriam ter acordado. Carro1: %v, Carro2: %v", carro1Acordou, carro2Acordou)
	}
	// Nível final = 100 - 30 - 40 = 30
	if tank.Level() != 30 {
		t.Errorf("O nível final deveria ser 30, mas é %f", tank.Level())
	}
}
