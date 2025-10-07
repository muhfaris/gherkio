package runner

import (
	"net/http"

	"github.com/muhfaris/gherkio/internal/loader"
)

type Context struct {
	Env         loader.Env
	Cat         loader.Catalog
	HTTP        *http.Client
	Store       map[string]any
	CurrentAuth string
}

func NewContext(env loader.Env, cat loader.Catalog) *Context {
	return &Context{
		Env:   env,
		Cat:   cat,
		HTTP:  &http.Client{Timeout: env.RequestTimeout()},
		Store: map[string]any{},
	}
}
