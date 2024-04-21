package rex

import (
	"log/slog"
	"os"
)

func InitLogger(verbose bool) {
	level := slog.LevelInfo

	if verbose {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))
}
