package gpc

type GPCOption func(IGPC)

func ChannelLen(length int) GPCOption {
	return func(g IGPC) {
		g.SetChannelLen(length)
	}
}

func CallTimeout(timeout int) GPCOption {
	return func(g IGPC) {
		g.SetCallTimeout(timeout)
	}
}

func NoCallTimeout() GPCOption {
	return func(g IGPC) {
		g.SetCallTimeout(GPC_CALL_NO_TIMEOUT)
	}
}
