package gpc

import (
	"fmt"
	"sync"
	"testing"
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
	//fmt.Printf("friend list: %v", *fm)
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

func TestFriend(t *testing.T) {
	handler := NewHandler()
	fmw := newFriendManagerWrapper()
	handler.RegisterHandle("add", fmw.add)
	handler.RegisterHandle("remove", fmw.remove)
	handler.RegisterHandle("output", fmw.output)

	idMax := 1000000
	friendGpc := NewGPCFast(handler, ChannelLen(idMax))
	go friendGpc.Run()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// add goroutine
	go func() {
		for id := 1; id <= idMax; id++ {
			f := &friend{
				id:   id,
				name: fmt.Sprintf("f_%v", id),
			}
			friendGpc.Call("add", f, nil)
			friendGpc.Call("output", nil, nil)
		}
		wg.Done()
	}()

	// remove goroutine
	go func() {
		var result bool
		for id := idMax; id >= 1; id-- {
			friendGpc.Call("remove", id, &result)
			if result {
				friendGpc.Call("output", nil, nil)
			}
		}
		wg.Done()
	}()

	wg.Wait()
}

type FriendManagerProc struct {
	fm *friendManager
}

func newFriendManagerProc() *FriendManagerProc {
	return &FriendManagerProc{
		fm: newFriendManager(),
	}
}

type AddArgs struct {
	f *friend
}

type AddResult struct {
}

func (f *FriendManagerProc) Add(arg *AddArgs, result *AddResult) error {
	f.fm.add(arg.f)
	return nil
}

type RemoveArgs struct {
	id int
}

type RemoveResult struct {
	res bool
}

func (f *FriendManagerProc) Remove(arg *RemoveArgs, result *RemoveResult) error {
	if !f.fm.remove(arg.id) {
		result.res = false
	} else {
		result.res = true
	}
	return nil
}

type OutputArgs struct {
}

type OutputResult struct {
}

func (f *FriendManagerProc) Output(arg *OutputArgs, result *OutputResult) error {
	return nil
}

func TestFriend2(t *testing.T) {
	idLength := 1000000
	gpc := NewGPC(ChannelLen(idLength))
	err := gpc.Register(newFriendManagerProc())
	if err != nil {
		t.Error(err)
		return
	}
	go gpc.Run()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// add goroutine
	go func() {
		for id := 1; id <= idLength; id++ {
			f := &friend{
				id:   id,
				name: fmt.Sprintf("f_%v", id),
			}
			err := gpc.Call("FriendManagerProc.Add", &AddArgs{f: f}, &AddResult{})
			if err != nil {
				t.Error(err)
			}
			err = gpc.Call("FriendManagerProc.Output", &OutputArgs{}, &OutputResult{})
			if err != nil {
				t.Error(err)
			}
		}
		wg.Done()
	}()

	// remove goroutine
	go func() {
		for id := idLength; id >= 1; id-- {
			result := &RemoveResult{}
			err := gpc.Call("FriendManagerProc.Remove", &RemoveArgs{id: id}, result)
			if err != nil {
				t.Error(err.Error())
			}
			if result.res {
				err = gpc.Call("FriendManagerProc.Output", &OutputArgs{}, &OutputResult{})
				if err != nil {
					t.Error(err)
				}
			}
		}
		wg.Done()
	}()
	wg.Wait()
}
