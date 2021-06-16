package gpc

import (
	"fmt"
)

type Handler struct {
	handlerMap map[string]func(param interface{}, result interface{}) error
}

func NewHandler() *Handler {
	return &Handler{
		handlerMap: make(map[string]func(param interface{}, result interface{}) error),
	}
}

func (h *Handler) RegisterHandle(method string, handleFunc func(param interface{}, result interface{}) error) {
	h.handlerMap[method] = handleFunc
}

func (h *Handler) Handle(method string, param interface{}, result interface{}) error {
	handle, o := h.handlerMap[method]
	if !o {
		return fmt.Errorf("gpc: Not found handle with method name %v", method)
	}
	return handle(param, result)
}

// struct for fast goroutine procedure call
type GPCFast struct {
	handler *Handler
	gpcBase
}

// create new GPC
func NewGPCFast(handler *Handler, options ...GPCOption) *GPCFast {
	gpc := &GPCFast{
		handler: handler,
	}
	for _, option := range options {
		option(gpc)
	}
	gpc.Init()
	// 在这里给Run中调用的处理函数赋值，有点丑，目前只能这么做，算是最简单的做法了
	gpc.callMethodFunc = gpc.callMethod
	return gpc
}

// Run中调用的处理函数，因为go无法支持在一个类型中的调用接口达到虚函数的效果
func (g *GPCFast) callMethod(method string, param interface{}, result interface{}) error {
	return g.handler.Handle(method, param, result)
}
