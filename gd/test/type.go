package main

type ConfigName int

//go:generate stringer -type=ConfigName -trimprefix=Cfg
const (
	CfgTest = ConfigName(iota)
	Max
)

func (t ConfigName) Idx() int {
	return int(t)
}
