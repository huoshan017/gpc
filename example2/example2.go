package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/huoshan017/gpc"
)

type friend struct {
	id   int
	name string
}

type FriendManager struct {
	friendList map[int]*friend
}

func newFriendManager() *FriendManager {
	return &FriendManager{
		friendList: make(map[int]*friend, 100),
	}
}

func (fm *FriendManager) add(f *friend) {
	fm.friendList[f.id] = f
}

func (fm *FriendManager) remove(id int) bool {
	_, o := fm.friendList[id]
	if !o {
		return false
	}
	delete(fm.friendList, id)
	return true
}

type FriendManagerProc struct {
	fm *FriendManager
}

func newFriendManagerProc() *FriendManagerProc {
	return &FriendManagerProc{
		fm: newFriendManager(),
	}
}

type Player struct {
	id        int
	name      string
	friendMgr *FriendManager
}

func NewPlayer(id int, name string) *Player {
	return &Player{
		id:        id,
		name:      name,
		friendMgr: newFriendManager(),
	}
}

func (p *Player) AddFriend(id int) bool {
	return true
}

type PlayerManager struct {
}

type AddArgs struct {
	newFriend *friend
}

type AddReplys struct {
}

func (f *FriendManagerProc) Add(arg *AddArgs, result *AddReplys) error {
	f.fm.add(arg.newFriend)
	return nil
}

type RemoveArgs struct {
	id int
}

type RemoveReplys struct {
}

func (f *FriendManagerProc) Remove(arg *RemoveArgs, result *RemoveReplys) error {
	if !f.fm.remove(arg.id) {
		return fmt.Errorf("remove friend %v failed", arg.id)
	}
	return nil
}

func main() {
	idLength := 100000
	gpcFriend, err := gpc.NewGPC(newFriendManagerProc(), gpc.ChannelLen(idLength))
	if err != nil {
		return
	}
	go gpcFriend.Run()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	// add goroutine
	go func() {
		for id := 1; id <= idLength; id++ {
			f := &friend{
				id:   id,
				name: fmt.Sprintf("f_%v", id),
			}
			gpcFriend.Call("FriendManagerProc.Add", &AddArgs{newFriend: f}, &AddReplys{})
		}
		wg.Done()
	}()

	// remove goroutine
	go func() {
		var result bool
		for id := idLength; id >= 1; id-- {
			gpcFriend.Call("FriendManagerProc.Remove", &RemoveArgs{id: id}, &RemoveReplys{})
			if result {
				log.Printf("FriendManagerProc Remove %v success", id)
			}
		}
		wg.Done()
	}()

	wg.Wait()
}
