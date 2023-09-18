package gd

import (
	"game_tools/internal/resync"
	"strings"
	"time"
)

type store struct {
	once resync.Once
	doc  interface{}
}

func (s *store) Reset() {
	s.once.Reset()
}

func (s *store) Load(key Key, loader func(name string) string,
	parser func(raw string) interface{},
) interface{} {
	s.once.Do(func() {
		raw := loader(key.String())
		s.doc = parser(raw)
	})
	return s.doc
}

type Key interface {
	Idx() int
	String() string
}

type Row struct {
	store  store
	deps   []int
	parser func(string) interface{}
}

type Gdd struct {
	row    []Row
	source Source
	suffix string

	mapper map[string]int
}

func NewGdd(max int, source Source, suffix string) *Gdd {
	return &Gdd{
		row:    make([]Row, max),
		source: source,
		suffix: suffix,
		mapper: make(map[string]int),
	}
}

func (gdd *Gdd) Register(key Key, loader func(string) interface{}, deps ...Key) {
	idx := key.Idx()
	d := make([]int, 0, len(deps))
	for _, dep := range deps {
		d = append(d, dep.Idx())
	}
	gdd.row[idx] = Row{
		deps:   d,
		parser: loader,
	}
	gdd.mapper[key.String()] = key.Idx()
}

func (gdd *Gdd) GetDoc(key Key) (doc interface{}) {
	c := &gdd.row[key.Idx()]
	return c.store.Load(key, gdd.source.GetDoc, c.parser)
}

func (gdd *Gdd) reset(key Key) {
	c := &gdd.row[key.Idx()]
	c.store.Reset()
}

func (gdd *Gdd) check(name string) (tag string, ok bool) {
	if gdd.suffix == "" {
		return name, true
	}
	if strings.HasSuffix(name, gdd.suffix) {
		return name[:len(name)-len(gdd.suffix)], true
	}
	return "", false
}

func (gdd *Gdd) Start() {
	var (
		wait   <-chan time.Time
		update = gdd.source.Watch()
	)
	for {
		select {
		case u, ok := <-update:
			if !ok {
				return
			}
			tag, ok := gdd.check(u.Name)

		}
	}
}
