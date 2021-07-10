package main

import (
	"fmt"
	"time"

	"github.com/huoshan017/gpc"
)

type friend struct {
	id   int
	name string
}

type friendManager struct {
	friendList map[int]*friend
}

func newFriendManager() *friendManager {
	return &friendManager{
		friendList: make(map[int]*friend, 100),
	}
}

func (fm *friendManager) add(f *friend) {
	fm.friendList[f.id] = f
}

func (fm *friendManager) remove(id int) bool {
	_, o := fm.friendList[id]
	if !o {
		return false
	}
	delete(fm.friendList, id)
	return true
}

func (fm *friendManager) output() {
	//fmt.Printf("friend list: %v\n", *fm)
}

func (fm *friendManager) timeout() {
	time.Sleep(10 * time.Second)
}

type friendManagerWrapper struct {
	fm *friendManager
}

func newFriendManagerWrapper() *friendManagerWrapper {
	fmw := &friendManagerWrapper{}
	fmw.fm = newFriendManager()
	return fmw
}

func (fmw *friendManagerWrapper) add(param interface{}, result interface{}) error {
	fmw.fm.add(param.(*friend))
	return nil
}

func (fmw *friendManagerWrapper) remove(param interface{}, result interface{}) error {
	res := fmw.fm.remove(param.(int))
	*(result.(*bool)) = res
	return nil
}

func (fmw *friendManagerWrapper) output(param interface{}, result interface{}) error {
	fmw.fm.output()
	return nil
}

func (fmw *friendManagerWrapper) timeout(param interface{}, result interface{}) error {
	fmw.fm.timeout()
	return nil
}

func One() {
	handler := gpc.NewHandler()
	fmw := newFriendManagerWrapper()
	handler.RegisterHandle("add", fmw.add)
	handler.RegisterHandle("remove", fmw.remove)
	handler.RegisterHandle("output", fmw.output)
	handler.RegisterHandle("timeout", fmw.timeout)
	friendGpc := gpc.NewGPCFast(handler)
	idMax := 1000
	// add goroutine
	go func() {
		for id := 1; id <= idMax; id++ {
			f := &friend{
				id:   id,
				name: fmt.Sprintf("f_%v", id),
			}
			friendGpc.Go("add", f)
			fmt.Printf("add friend %v\n", id)
			friendGpc.Go("output", nil)
		}
	}()

	// remove goroutine
	go func() {
		var result bool
		for id := idMax; id >= 1; id-- {
			err := friendGpc.Call("remove", id, &result)
			if err != nil {
				panic(fmt.Sprintf("call remove %v panic", id))
			}
			if result {
				fmt.Printf("remove friend %v\n", id)
				friendGpc.Go("output", nil)
			}
		}
	}()

	go func() {
		time.Sleep(1 * time.Millisecond)
		friendGpc.Go("timeout", nil)
	}()

	go friendGpc.Run()

	for {
		time.Sleep(time.Millisecond)
	}
}

func main() {
	One()
}
