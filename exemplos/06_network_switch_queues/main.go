package main

import (
	"github.com/rbelligit/go_evt_simul/components"
	"github.com/rbelligit/go_evt_simul/evtmanager"
)

type PackageT struct {
	MacSrc   [6]byte
	MacDst   [6]byte
	Priority int
	nBytes   int
}

func (p *PackageT) GetPriority() int {
	return p.Priority
}

func (p *PackageT) GetOutPort() int {
	return -1
}

func (p *PackageT) GetPackageBytes() int {
	return p.nBytes
}

func (p *PackageT) GetMacAddressSrc() [6]byte {
	return p.MacSrc
}

func (p *PackageT) GetMacAddressDst() [6]byte {
	return p.MacDst
}

func main() {
	env := evtmanager.NewEnvironment()

	switch_ := components.NewSwitch[*PackageT](env, 2, 100, 10, []int{8, 4, 2, 1})
	_ = switch_
}
