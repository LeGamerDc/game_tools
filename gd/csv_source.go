package gd

import (
	"errors"
	"github.com/fsnotify/fsnotify"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func NewCsvSource(root string) (Source, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.New("create fs watcher fail")
	}
	update := make(chan []DocChange, 1)
	source := &csvSource{
		watcher: watcher,
		update:  update,
	}
	source.initDirectory(root)
	go source.loop()
	return source, nil
}

type csvSource struct {
	watcher *fsnotify.Watcher
	update  chan []DocChange
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
	b, err := os.ReadFile(path.Join(p, name, "csv"))
	if err != nil {
		slog.Error("fsSource get doc", "err", err)
		return ""
	}
	return string(b)
}

func (f *csvSource) Watch() <-chan []DocChange {
	return f.update
}

func (f *csvSource) loop() {
	var (
		wait    <-chan time.Time
		watcher = f.watcher
		updates []DocChange
	)
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			var (
				op Op
				p  = path.Dir(event.Name)
				n  = path.Base(event.Name)
			)
			if event.Has(fsnotify.Create) && isDir(event.Name) {
				_ = f.watcher.Add(event.Name)
			} else if strings.HasSuffix(n, ".csv") {
				n = n[:len(n)-4]
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
				u := DocChange{
					Path: p,
					Name: n,
					Op:   op,
				}
				updates = append(updates, u)
				if wait == nil {
					wait = time.After(time.Second * 10)
				}
			}
		case <-wait:
			slog.Info("fsSource updates", slog.Any("updates", updates))
			f.update <- updates
			updates = make([]DocChange, 0, 8)
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

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
