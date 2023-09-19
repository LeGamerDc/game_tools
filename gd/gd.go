package gd

import (
	"game_tools/internal/resync"
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
	store   store
	deps    []int
	affects []int
	parser  func(string) interface{}
}

type Gdd struct {
	row    []Row
	source Source

	mapper map[string]int
}

func NewGdd(max int, source Source) *Gdd {
	return &Gdd{
		row:    make([]Row, max),
		source: source,
		mapper: make(map[string]int),
	}
}

func (gdd *Gdd) Start() {
	gdd.buildDeps()
	go gdd.loop()
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

func (gdd *Gdd) buildDeps() {
	for idx := range gdd.row {
		a := &gdd.row[idx]
		for _, dep := range a.deps {
			b := &gdd.row[dep]
			b.affects = append(b.affects, idx)
		}
	}
}

func (gdd *Gdd) dfs(root int, set map[int]struct{}) {
	set[root] = struct{}{}
	c := &gdd.row[root]
	for _, affect := range c.affects {
		gdd.dfs(affect, set)
	}
}

func (gdd *Gdd) reset(idx int) {
	c := &gdd.row[idx]
	c.store.Reset()
}

func (gdd *Gdd) loop() {
	update := gdd.source.Watch()
	for us := range update {
		affect := make(map[int]struct{}, 2*len(us))
		for _, u := range us {
			if idx, ok := gdd.mapper[u.Name]; ok {
				gdd.dfs(idx, affect)
			}
		}
		for idx := range affect {
			gdd.reset(idx)
		}
	}
}
