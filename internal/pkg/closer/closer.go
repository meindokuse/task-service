// Package closer собирает хуки завершения работы и выполняет их в порядке LIFO,
// чтобы ресурсы, захваченные последними (например, HTTP-сервер), освобождались первыми.
package closer

import (
	"context"
	"sync"
)

type Closer struct {
	mu    sync.Mutex
	funcs []func(ctx context.Context) error
}

func New() *Closer {
	return &Closer{}
}

func (c *Closer) Add(f func(ctx context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.funcs = append(c.funcs, f)
}

// Close выполняет каждый зарегистрированный хук в обратном порядке регистрации,
// накапливая ошибки (не прерываясь на первой из них).
func (c *Closer) Close(ctx context.Context) error {
	c.mu.Lock()
	funcs := make([]func(ctx context.Context) error, len(c.funcs))
	copy(funcs, c.funcs)
	c.mu.Unlock()

	var err error
	for i := len(funcs) - 1; i >= 0; i-- {
		if cerr := funcs[i](ctx); cerr != nil {
			err = cerr
		}
	}
	return err
}
