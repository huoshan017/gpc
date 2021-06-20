package gpc

import (
	"fmt"
	"reflect"
	"time"
)

type data struct {
	method         string
	param          interface{}
	result         interface{}
	resChan        chan error
	startTimerChan chan *time.Timer
}

const (
	GPC_CHANNEL_LEN     = 100
	GPC_CALL_TIMEOUT    = 1000 // 默認調用超時爲1000毫秒
	GPC_CALL_NO_TIMEOUT = -1   // 沒有超時
)

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// gpc的基础结构，封装了基础功能
type gpcBase struct {
	options        Options
	ch             chan *data
	callMethodFunc func(string, interface{}, interface{}) error
	closeChan      chan struct{}
}

// 初始化
func (g *gpcBase) init(callMethod func(string, interface{}, interface{}) error) {
	if g.options.chLen <= 0 {
		g.options.chLen = GPC_CHANNEL_LEN
	}
	g.ch = make(chan *data, g.options.chLen)
	if g.options.callTimeout == 0 {
		g.options.callTimeout = GPC_CALL_TIMEOUT
	}
	// 在这里给Run中调用的处理函数赋值，目前没有更好的方法，这算是最简单的做法了
	g.callMethodFunc = callMethod
	g.closeChan = make(chan struct{})
}

// 用于外部调用的方法，同步调用
func (g *gpcBase) Call(methodName string, param interface{}, result interface{}) (err error) {
	// 错误通道
	resChan := make(chan error)
	defer close(resChan)

	// 调用超时通道
	var startTimerChan chan *time.Timer
	if g.options.callTimeout >= 0 {
		startTimerChan = make(chan *time.Timer)
		defer close(startTimerChan)
	}

	// 调用数据
	d := &data{
		method:         methodName,
		param:          param,
		result:         result,
		resChan:        resChan,
		startTimerChan: startTimerChan,
	}
	g.ch <- d

	// 调用超时检测
	if startTimerChan != nil {
		// 取出计时器
		timer := <-startTimerChan
		// 等待結果的處理
		select {
		// todo 注意这里的time.Time通道会在超时后由计时器内部关闭
		case <-timer.C:
			err = fmt.Errorf("gpc: call method (%v) timeout", methodName)
			// 等待错误结果
		case err = <-resChan:
		}
	} else { // 只有错误结果
		err = <-resChan
	}

	return err
}

// 循环执行，为了不阻塞调用的goroutine，一般要加上go关键字再执行
func (g *gpcBase) Run() {
	isRun := true
	for isRun {
		select {
		case d := <-g.ch:
			// 开启一个计时器
			if d.startTimerChan != nil {
				d.startTimerChan <- time.NewTimer(time.Millisecond * GPC_CALL_TIMEOUT)
			}
			// 處理GPC調用
			err := g.callMethodFunc(d.method, d.param, d.result)
			d.resChan <- err
		case <-g.closeChan:
			isRun = false
		}
	}
}

// 关闭
func (g *gpcBase) Close() {
	close(g.closeChan)
}
