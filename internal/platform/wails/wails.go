// Package wailsplatform — реализация app.Platform для десктопной оболочки на Wails.
package wailsplatform

import (
	"context"
	"sync"

	wr "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Platform хранит контекст Wails, который появляется только при старте окна.
type Platform struct {
	mu  sync.RWMutex
	ctx context.Context
}

// SetContext вызывается из OnStartup оболочки до запуска ядра.
func (p *Platform) SetContext(ctx context.Context) {
	p.mu.Lock()
	p.ctx = ctx
	p.mu.Unlock()
}

func (p *Platform) context() context.Context {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ctx
}

func (p *Platform) Emit(event string, data ...any) {
	if ctx := p.context(); ctx != nil {
		wr.EventsEmit(ctx, event, data...)
	}
}

func (p *Platform) PickFile(title string) (string, error) {
	return wr.OpenFileDialog(p.context(), wr.OpenDialogOptions{
		Title:   title,
		Filters: []wr.FileFilter{{DisplayName: "All Files", Pattern: "*"}},
	})
}

func (p *Platform) OpenURL(url string) {
	if ctx := p.context(); ctx != nil {
		wr.BrowserOpenURL(ctx, url)
	}
}
