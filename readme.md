# GoSim-Iter 🚀
### Discrete Event Simulation Engine for Go 1.23+

[🇺🇸 Read in English](README_en.md)

**GoSim-Iter** é um motor de simulação de eventos discretos (DES) de alta performance, projetado para ser leve, determinístico e extremamente escalável. Inspirado fortemente pelo funcionamento e API da consagrada biblioteca Python **SimPy**, o projeto adapta esses conceitos clássicos de simulação para o ecossistema moderno do Go. Diferente de outras bibliotecas que utilizam goroutines e canais para cada processo, esta biblioteca utiliza os novos **Iteradores nativos do Go 1.23 (`iter` package)** para gerenciar o fluxo de execução.
Ainda em versão pré-alfa

## 🌟 Diferenciais Técnicos

- **Corrotinas sobre Goroutines:** Ao usar `iter.Pull`, cada processo simulado consome apenas o espaço necessário para sua stack de execução, sem o overhead de agendamento do runtime do Go.
- **Kernel de Simulação Não Bloqueante:** O `Environment` gerencia o tempo virtual de forma atômica. Nenhuma operação de rede ou recurso trava a thread real do sistema.
- **Priority Queue com Index Tracking:** Implementação customizada de `container/heap` que permite operações de `Fix` (reordenamento) em $O(\log n)$, essencial para interrupções e preempção.
- **Desempate Determinístico:** Recursos (`Resources`) utilizam um contador de ordem de chegada (`indiceOrd`) para garantir que processos com a mesma prioridade sejam atendidos em ordem FIFO estrita.

---

## 🏗️ Arquitetura do Sistema

A biblioteca é dividida em quatro pilares fundamentais:

1.  **Environment:** O relógio mestre e o agendador de eventos.
2.  **Process:** A unidade de execução (Task). Ela "cede" o controle para o Environment através de comandos.
3.  **Commands:** Funções que definem a próxima ação do processo (esperar tempo, solicitar recurso, sinalizar evento).
4.  **PendingAgents:** Uma interface flexível que permite que qualquer objeto (um Processo ou um Evento de Recurso) seja colocado  em filas de espera. Ainda em analise esta implementação.

---

## 💻 Exemplos de Implementação

O projeto conta com exemplos executáveis completos na pasta `exemplos/`, demonstrando cenários reais de uso:

- **[01_resource_example](./exemplos/01_resource_example/)**: Demonstra o uso de `Environment`, `WaitTime` e limites de capacidade através de `Resources` (ex: clientes aguardando atendimento num banco).
- **[02_store_example](./exemplos/02_store_example/)**: Exemplo do padrão Produtor/Consumidor utilizando `Store` para troca segura de pacotes e dados.
- **[03_container_example](./exemplos/03_container_example/)**: Mostra o gerenciamento de níveis contínuos/quantitativos com `Container` (ex: abastecimento e consumo de um reservatório).
- **[04_event_example](./exemplos/04_event_example/)**: Demonstra sincronização entre múltiplos processos utilizando `Event` (ex: trabalhadores aguardando a chegada de peças).
- **[05_filter_store_example](./exemplos/04_filter_store_example/)**: Demonstra o uso de `FilterStore` com consumidores aguardando itens específicos baseados em funções de filtro.

Para executá-los localmente:
```bash
go run ./exemplos/01_resource_example/main.go
go run ./exemplos/02_store_example/main.go
go run ./exemplos/03_container_example/main.go
go run ./exemplos/04_event_example/main.go
go run ./exemplos/04_filter_store_example/main.go
```

### Recursos e Tempo
Escreva seus processos de simulação de forma sequencial, como se fossem threads reais, utilizando o `yield` para suspender a execução:

```go
func meuProcesso(yield func(Command) bool) {
    // 1. Acessar o processo atual (helper para capturar o contexto)
    var p *Process
    GetProcess(&p, yield)

    // 2. Aguardar tempo virtual
    yield(WaitTime(10 * time.Second))

    // 3. Tentar adquirir um recurso limitado
    // Parâmetros: processo, quantidade, prioridade, yield
    res.Acquire(p, 1, 10, yield) 
    
    // O código aqui só executa quando o recurso for liberado para este processo
    fmt.Println("Recurso adquirido em:", p.GetEnv().GetTime())

    yield(WaitTime(5 * time.Second)) // Simula uso do recurso

    // 4. Liberar recurso
    res.Release(p, 1)
}
```

### Filas de Dados genéricas (`Store[T]`)
Troque mensagens blocantes entre corrotinas com segurança estrita da simulação:

```go
func meuProdutor(yield func(Command) bool) {
    yield(store.Put("Pacote A", 10)) // Suspende processo se Buffer > Capacidade configurada
}

func meuConsumidor(yield func(Command) bool) {
    var pacote string
    yield(store.Get(&pacote, 10)) // Suspende processo se Buffer == 0
    fmt.Println("Processado:", pacote)
}
```

---

## 🛠️ Status das Funcionalidades

| Funcionalidade | Status | Descrição |
| :--- | :---: | :--- |
| **Event Loop** | ✅ | Motor principal com avanço de tempo instantâneo. |
| **WaitTime** | ✅ | Suspensão de processos por duração específica. |
| **Resources** | ✅ | Gerenciamento de capacidade com fila de prioridade e FIFO. |
| **Iter.Pull Integration** | ✅ | Ciclo de vida completo (Created -> Running -> Terminated). |
| **Stores** | ✅ | Fila genérica (`Store[T]`) de dados/pacotes com suspensão por backpressure. |
| **Containers** | ✅ | (Gerenciamento de recursos contínuos, ex: fluidos, baterias). |
| **Events (Triggers)** | ✅ | (Sinalização manual simples de sucesso/falha entre processos). |
| **FilterStores** | ✅ | Filas de dados com extração bloqueante baseada em funções de filtro. |
| **WaitAny/All** | 🚧 | **Planejado** (Sincronização complexa de multiplos eventos Condition/AllOf/AnyOf). |
| **Preemptive Resources** | 🚧 | **Planejado** (Recursos com interrupção forçada/preempção de processos de baixa prioridade). |

---

## 📦 Requisitos

Certifique-se de estar usando o **Go 1.23** ou superior, pois a biblioteca depende da funcionalidade de iteradores (`iter` package) e do suporte a `range-over-func`.
