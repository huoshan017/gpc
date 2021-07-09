package gpc

type Options struct {
	chLen       int
	callTimeout int   // 毫秒
	tickMs      int32 // 定时器函数的调用间隔
}

func (option *Options) SetChannelLen(length int) {
	option.chLen = length
}

func (option *Options) SetCallTimeout(timeout int) {
	option.callTimeout = timeout
}

func (option *Options) SetTickMs(tickMs int32) {
	option.tickMs = tickMs
}

type GPCOption func(Options)

func ChannelLen(length int) GPCOption {
	return func(option Options) {
		option.SetChannelLen(length)
	}
}

func CallTimeout(timeout int) GPCOption {
	return func(option Options) {
		option.SetCallTimeout(timeout)
	}
}

func NoCallTimeout() GPCOption {
	return func(option Options) {
		option.SetCallTimeout(GPC_CALL_NO_TIMEOUT)
	}
}

func TickMs(tickMs int32) GPCOption {
	return func(option Options) {
		option.SetTickMs(tickMs)
	}
}