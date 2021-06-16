package gpc

import (
	"fmt"
)

// 调用处理器
type Handler struct {
	handlerMap map[string]func(param interface{}, result interface{}) error
}

// 创建调用处理器
func NewHandler() *Handler {
	return &Handler{
		handlerMap: make(map[string]func(param interface{}, result interface{}) error),
	}
}

// 注册一个处理函数
func (h *Handler) RegisterHandle(method string, handleFunc func(param interface{}, result interface{}) error) {
	h.handlerMap[method] = handleFunc
}

// 外部调用处理
func (h *Handler) Handle(method string, param interface{}, result interface{}) error {
	handle, o := h.handlerMap[method]
	if !o {
		return fmt.Errorf("gpc: Not found handle with method name %v", method)
	}
	return handle(param, result)
}

// 快速gpc结构
type GPCFast struct {
	handler *Handler
	gpcBase
}

// 创建一个快速的gpc
func NewGPCFast(handler *Handler, options ...GPCOption) *GPCFast {
	gpc := &GPCFast{
		handler: handler,
	}
	for _, option := range options {
		option(gpc)
	}
	gpc.Init(gpc.callMethod)
	return gpc
}

// Run中调用的处理函数，因为go无法支持在一个类型中的方法中调用接口达到虚函数的效果
func (g *GPCFast) callMethod(method string, param interface{}, result interface{}) error {
	return g.handler.Handle(method, param, result)
}
