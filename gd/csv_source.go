package gd

import (
	"errors"
	"github.com/fsnotify/fsnotify"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

func NewCsvSource(root string) (Source, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.New("create fs watcher fail")
	}
	update := make(chan DocChange, 1)
	source := &csvSource{
		watcher: watcher,
		update:  update,
	}
	source.initDirectory(root)
	go func() {
		for {
			event, ok := <-watcher.Events
			if !ok {
				return
			}
			var op Op
			if event.Has(fsnotify.Create) {
				op = Create
			} else if event.Has(fsnotify.Write) {
				op = Write
			} else if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
				op = Remove
			} else {
				continue
			}
			update <- DocChange{
				Path: path.Dir(event.Name),
				Name: path.Base(event.Name),
				Op:   op,
			}
		}
	}()
	return source, nil
}

type csvSource struct {
	watcher *fsnotify.Watcher
	update  chan DocChange
	dirMap  map[string]string
}

func (f *csvSource) Close() {
	_ = f.watcher.Close()
}

func (f *csvSource) GetDoc(name string) string {
	p, ok := f.dirMap[name]
	if !ok {
		slog.Error("fsSource get doc no dir")
		return ""
	}
	b, err := os.ReadFile(path.Join(p, name))
	if err != nil {
		slog.Error("fsSource get doc", "err", err)
		return ""
	}
	return string(b)
}

func (f *csvSource) Watch() <-chan DocChange {
	return f.update
}

func (f *csvSource) loop() {
	watcher := f.watcher
	for {
		event, ok := <-watcher.Events
		if !ok {
			return
		}
		var (
			op Op
			p  = path.Dir(event.Name)
			n  = path.Base(event.Name)
		)
		if event.Has(fsnotify.Create) {
			op = Create
			f.dirMap[n] = p
		} else if event.Has(fsnotify.Write) {
			op = Write
		} else if event.Has(fsnotify.Rename) || event.Has(fsnotify.Remove) {
			op = Remove
		} else {
			continue
		}
		f.update <- DocChange{
			Path: path.Dir(event.Name),
			Name: path.Base(event.Name),
			Op:   op,
		}
	}
}

func (f *csvSource) initDirectory(root string) {
	dirs := make([]string, 0, 10)
	f.dirMap = make(map[string]string)
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if err != nil {
				slog.Info("fsSource ignore", slog.String("path", p), slog.Any("err", err))
			} else {
				slog.Info("fsSource watch", slog.String("path", p))
				dirs = append(dirs, p)
			}
		} else {
			f.dirMap[d.Name()] = p
		}
		return nil
	})

	for _, dir := range dirs {
		_ = f.watcher.Add(dir)
	}
}
