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
	errChan        chan error
	startTimerChan chan *time.Timer
}

const (
	GPC_CHANNEL_LEN     = 100
	GPC_CALL_TIMEOUT    = 1000 // 默認調用超時爲1000毫秒
	GPC_CALL_NO_TIMEOUT = -1   // 沒有超時
	GPC_TICK_MS         = 10   // 定时器函数调用间隔
)

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

// gpc的基础结构，封装了基础功能
type gpcBase struct {
	options        Options
	ch             chan *data
	callMethodFunc func(string, interface{}, interface{}) error
	goMethodFunc   func(string, interface{})
	tickMethodFunc func(tick int32)
	closeChan      chan struct{}
}

// 初始化
func (g *gpcBase) init(
	callMethod func(string, interface{}, interface{}) error,
	goMethod func(string, interface{}),
	tickMethod func(tick int32)) {
	if g.options.chLen <= 0 {
		g.options.chLen = GPC_CHANNEL_LEN
	}
	g.ch = make(chan *data, g.options.chLen)
	if g.options.callTimeout == 0 {
		g.options.callTimeout = GPC_CALL_TIMEOUT
	}
	if g.options.tickMs == 0 {
		g.options.tickMs = GPC_TICK_MS
	}
	// 在这里给Run中调用的处理函数赋值，目前没有更好的方法，这算是最简单的做法了
	g.callMethodFunc = callMethod
	g.goMethodFunc = goMethod
	g.tickMethodFunc = tickMethod
	g.closeChan = make(chan struct{})
}

// 外部调用无返回值的方法
func (g *gpcBase) Go(methodName string, param interface{}) {
	// 调用数据
	d := &data{
		method: methodName,
		param:  param,
	}
	g.ch <- d
}

// 用于外部调用的方法，同步调用
func (g *gpcBase) Call(methodName string, param interface{}, result interface{}) (err error) {
	if result == nil {
		panic("gpc: Call result param cant be nil")
	}

	// 错误通道
	errChan := make(chan error)
	defer close(errChan)

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
		errChan:        errChan,
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
		case err = <-errChan:
		}
	} else { // 只有错误结果
		err = <-errChan
	}

	return err
}

// 循环执行，为了不阻塞调用的goroutine，一般要加上go关键字再执行
func (g *gpcBase) Run() {
	var ticker *time.Ticker
	if g.tickMethodFunc != nil {
		ticker = time.NewTicker(time.Duration(g.options.tickMs * int32(time.Millisecond)))
	}
	lastTime := time.Now()
	isRun := true
	if ticker != nil {
		for isRun {
			select {
			case d := <-g.ch:
				g.process(d)
			case <-ticker.C:
				now := time.Now()
				tick := now.Sub(lastTime)
				g.tickMethodFunc(int32(tick.Milliseconds()))
				lastTime = now
			case <-g.closeChan:
				isRun = false
			}
		}
	} else {
		for isRun {
			select {
			case d := <-g.ch:
				g.process(d)
			case <-g.closeChan:
				isRun = false
			}
		}
	}
}

func (g *gpcBase) process(d *data) {
	if d.result != nil {
		// 开启一个计时器
		if d.startTimerChan != nil {
			d.startTimerChan <- time.NewTimer(time.Millisecond * GPC_CALL_TIMEOUT)
		}
		// 處理GPC調用
		err := g.callMethodFunc(d.method, d.param, d.result)
		d.errChan <- err
	} else {
		g.goMethodFunc(d.method, d.param)
	}
}

// 关闭
func (g *gpcBase) Close() {
	close(g.closeChan)
}
