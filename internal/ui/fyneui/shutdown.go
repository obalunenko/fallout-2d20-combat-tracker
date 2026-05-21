package fyneui

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"fyne.io/fyne/v2"
)

func installSignalShutdown(app fyne.App, onShutdown func()) func() {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case <-signals:
			onShutdown()
			fyne.Do(app.Quit)
		case <-done:
		}
	}()

	return func() {
		close(done)
		signal.Stop(signals)
	}
}

func shutdownOnce(fn func()) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			if fn != nil {
				fn()
			}
		})
	}
}
