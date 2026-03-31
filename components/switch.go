package components

import (
	"fmt"
	"time"

	"github.com/rbelligit/go_evt_simul/evtmanager"
)

// Inputs  _______________________
// -->     |                     |
// -->     |                     |
// -->     |                     |
// -->     |                     | -->
// -->     |                     | -->
//         |_____________________| -->
//
// As entradas não tem prioridade, mas as saídas tem
// os pacotes tem número de bytes e endereço destino que pode ser usado para roteamento

// Simulates entries in 8 priority queues and output of package in one queue

// macTableEntry representa uma entrada na tabela MAC do switch,
// armazenando a porta associada e o tempo do último registro.
type macTableEntry struct {
	port     int
	lastTime time.Duration
}

// Switch representa um switch genérico simulado que recebe e roteia
// pacotes para portas de saída baseando-se no endereço MAC e prioridade.
type Switch[PacketType PackageTypeInt] struct {
	nPorts int

	maxSizeQueue int
	// switch só tem filas nas saídas, assim que o pacote chega é roteado para uma fila de saída
	QueuesOut   []*evtmanager.QoSStore[PacketType] // uma saída, com prioridade
	UsedSizeOut [][]int

	MacTable   map[[6]byte]macTableEntry
	forgetTime time.Duration

	env *evtmanager.Environment
}

// Interface que define um pacote
// Se o pacote retornar -1 em GetOutPort(), ele será roteado internamente pelo MAC address, conforme aprendizado de um switch
// se for >= 0 e < numero máximo de saídas, ele será enviado apenas para esta saída.
type PackageTypeInt interface {
	GetPriority() int
	GetOutPort() int
	GetPackageBytes() int
	GetMacAddressDst() [6]byte
	GetMacAddressSrc() [6]byte
}

// NewSwitch cria e inicializa um novo Switch com os parâmetros de simulação,
// número de portas de entrada e saída, tamanho máximo da fila e tempo de expiração da tabela MAC.
func NewSwitch[PacketType PackageTypeInt](env *evtmanager.Environment, nPorts int,
	maxSizeQueue int, forgetTime time.Duration, weights []int) *Switch[PacketType] {
	s := &Switch[PacketType]{

		QueuesOut:    make([]*evtmanager.QoSStore[PacketType], nPorts),
		UsedSizeOut:  make([][] /*saida*/ /*pesos*/ int, nPorts),
		maxSizeQueue: maxSizeQueue,
		nPorts:       nPorts,
		MacTable:     make(map[[6]byte]macTableEntry),
		env:          env,
		forgetTime:   forgetTime,
	}

	for i := 0; i < nPorts; i++ {
		s.QueuesOut[i] = evtmanager.NewQoSStore[PacketType](env, fmt.Sprintf("Out%d", i), 10000, weights)
		s.UsedSizeOut[i] = make([]int, len(weights))
		for j := 0; j < len(weights); j++ {
			s.UsedSizeOut[i][j] = 0
		}
	}
	return s
}

// AddPacketIn insere o pacote na fila de saída determinada, se a fila não for achada, faz broadcast
// Retorna true se o pacote foi inserido, false caso contrário
func (s *Switch[PacketType]) AddPacketIn(portIndex int, p PacketType, yield func(evtmanager.Command) bool) bool {
	if portIndex < 0 || portIndex >= s.nPorts {
		return false
	}

	outPort := p.GetOutPort()
	if outPort >= s.nPorts {
		return false // Descarta pacotes destinados a portas que não existem
	}

	macOr := p.GetMacAddressSrc()
	s.MacTable[macOr] = macTableEntry{port: portIndex, lastTime: s.env.GetTime()}
	if outPort < 0 {
		// pegar o pacote de saída baseado no MAC address
		macDst := p.GetMacAddressDst()
		outPort = s.getOutPort(macDst)
		// Caso o lookup retorne uma porta aprendida que não exista como fila de saída
		if outPort >= s.nPorts {
			return false
		}
	}

	prio := p.GetPriority()
	numTcs := len(s.UsedSizeOut[0])
	if prio < 0 {
		prio = 0
	} else if prio >= numTcs {
		prio = numTcs - 1
	}

	nBytes := p.GetPackageBytes()

	if outPort < 0 {
		// broadcast
		AtLeastOneNotSent := false
		for i := 0; i < s.nPorts; i++ {
			if s.UsedSizeOut[i][prio]+nBytes > s.maxSizeQueue {
				AtLeastOneNotSent = true
				continue
			}

			s.UsedSizeOut[i][prio] += nBytes
			yield(s.QueuesOut[i].Put(p, prio, 1))
		}
		return !AtLeastOneNotSent
	} else {
		if s.UsedSizeOut[outPort][prio]+nBytes > s.maxSizeQueue {
			return false
		}
		s.UsedSizeOut[outPort][prio] += nBytes
		yield(s.QueuesOut[outPort].Put(p, prio, 1))
		return true
	}
}

// getOutPort consulta a tabela MAC para encontrar a porta de saída de um endereço MAC de destino.
// Se a entrada tiver expirado de acordo com forgetTime, ela é removida e a função retorna -1.
func (s *Switch[PacketType]) getOutPort(macDst [6]byte) int {
	entry, ok := s.MacTable[macDst]
	if !ok {
		return -1
	}
	if s.env.GetTime()-entry.lastTime > s.forgetTime {
		delete(s.MacTable, macDst)
		return -1
	}
	return entry.port
}

// GetOutputQueue expõe a fila de saída (Store) dada uma porta destino e uma classe de prioridade (0 a 7),
// sendo útil para a coleta de pacotes processados pelo switch pelos atores do destino.
func (s *Switch[PacketType]) GetOutputQueue(outPort int) *evtmanager.QoSStore[PacketType] {
	if outPort < 0 || outPort >= s.nPorts {
		return nil
	}
	return s.QueuesOut[outPort]
}
