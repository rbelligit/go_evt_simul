# GoSim-Iter 🚀
### Discrete Event Simulation Engine for Go 1.23+

**GoSim-Iter** é um motor de simulação de eventos discretos (DES) de alta performance, projetado para ser leve, determinístico e extremamente escalável. Diferente de outras bibliotecas que utilizam goroutines e canais para cada processo, esta biblioteca utiliza os novos **Iteradores nativos do Go 1.23 (`iter` package)** para gerenciar o fluxo de execução.

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

## 💻 Exemplo de Implementação

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

---

## 🛠️ Status das Funcionalidades

| Funcionalidade | Status | Descrição |
| :--- | :---: | :--- |
| **Event Loop** | ✅ | Motor principal com avanço de tempo instantâneo. |
| **WaitTime** | ✅ | Suspensão de processos por duração específica. |
| **Resources** | ✅ | Gerenciamento de capacidade com fila de prioridade e FIFO. |
| **Iter.Pull Integration** | ✅ | Ciclo de vida completo (Created -> Running -> Terminated). |
| **Stores** | 🚧 | **Em desenvolvimento** (Fila de dados/pacotes para redes). |
| **WaitAny/All** | 🚧 | **Planejado** (Sincronização complexa de eventos). |

---

## 📦 Requisitos

Certifique-se de estar usando o **Go 1.23** ou superior, pois a biblioteca depende da funcionalidade de iteradores (`iter` package) e do suporte a `range-over-func`.