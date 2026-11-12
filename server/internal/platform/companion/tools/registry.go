package tools

import (
	"context"
	"encoding/json"
)

type Context struct {
	UserID int64
}

type Result struct {
	Name    string
	Content string
}

type Handler func(ctx context.Context, tc *Context, args json.RawMessage) (Result, error)

type Registry struct {
	handlers map[string]Handler
	defs     []Definition
}

type Definition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

func NewRegistry() *Registry {
	return &Registry{handlers: map[string]Handler{}}
}

func (r *Registry) Register(def Definition, h Handler) {
	r.defs = append(r.defs, def)
	r.handlers[def.Name] = h
}

func (r *Registry) Definitions() []Definition {
	return r.defs
}

func (r *Registry) Execute(ctx context.Context, tc *Context, name string, args json.RawMessage) (Result, error) {
	h, ok := r.handlers[name]
	if !ok {
		return Result{Name: name, Content: `{"error":"unknown tool"}`}, nil
	}
	return h(ctx, tc, args)
}

func (r *Registry) LLMToolDefs() []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(r.defs))
	for _, d := range r.defs {
		out = append(out, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        d.Name,
				"description": d.Description,
				"parameters":  d.Parameters,
			},
		})
	}
	return out
}
