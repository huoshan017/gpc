package gpc

import (
	"fmt"
	"reflect"
	"sync/atomic"
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

// 设置参数的接口
type IGPC interface {
	SetChannelLen(length int)
	SetCallTimeout(timeout int)
}

// gpc的基础结构，封装了基础功能
type gpcBase struct {
	ch             chan *data
	chLen          int
	callTimeout    int // 毫秒
	isRun          int32
	callMethodFunc func(string, interface{}, interface{}) error
}

func (g *gpcBase) SetChannelLen(length int) {
	g.chLen = length
}

func (g *gpcBase) SetCallTimeout(timeout int) {
	g.callTimeout = timeout
}

func (g *gpcBase) Init() {
	if g.chLen <= 0 {
		g.chLen = GPC_CHANNEL_LEN
	}
	g.ch = make(chan *data, g.chLen)
	if g.callTimeout == 0 {
		g.callTimeout = GPC_CALL_TIMEOUT
	}
	g.isRun = 1
}

func (g *gpcBase) Call(methodName string, param interface{}, result interface{}) (err error) {
	resChan := make(chan error)
	var startTimerChan chan *time.Timer
	// 需要调用超时
	if g.callTimeout >= 0 {
		startTimerChan = make(chan *time.Timer)
	}

	d := &data{
		method:         methodName,
		param:          param,
		result:         result,
		resChan:        resChan,
		startTimerChan: startTimerChan,
	}
	g.ch <- d

	// 有调用超时检测
	if startTimerChan != nil {
		// 取出计时器
		timer := <-startTimerChan
		// 等待結果的處理
		select {
			// todo 注意这里的time.Time通道会在超时后有计时器内部关闭
		case <-timer.C:
			err = fmt.Errorf("gpc: call method (%v) timeout", methodName)
			// 等待结果
		case err = <-resChan:
		}
	} else {
		err = <-resChan
	}

	return err
}

func (g *gpcBase) Run() {
	for atomic.LoadInt32(&g.isRun) > 0 {
		d := <-g.ch
		// 开启一个计时器
		if d.startTimerChan != nil {
			d.startTimerChan <- time.NewTimer(time.Millisecond * GPC_CALL_TIMEOUT)
		}
		// 處理GPC調用
		err := g.callMethodFunc(d.method, d.param, d.result)
		d.resChan <- err
	}
}

func (g *gpcBase) Stop() {
	atomic.StoreInt32(&g.isRun, 0)
}
