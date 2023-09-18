package main

import (
	"game_tools/gd"
	"log/slog"
)

func main() {
	source, err := gd.NewFsSource("C:\\Users\\dongcheng\\Downloads")
	if err != nil {
		slog.Error("create fsSource", slog.Any("err", err))
	}
	update := source.Watch()
	for u := range update {
		slog.Info("update", slog.String("path", u.Path), slog.String("name", u.Name),
			slog.Int("op", int(u.Op)))
	}
}
