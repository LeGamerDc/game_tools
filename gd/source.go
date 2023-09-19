package gd

type Source interface {
	GetDoc(name string) string
	Watch() <-chan []DocChange
	Close()
}

type DocChange struct {
	Path string
	Name string
	Op   Op
}

type Op uint32

const (
	Create Op = iota
	Write
	Remove
)
