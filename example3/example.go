package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/huoshan017/gpc"
)

type Player struct {
	id      int
	name    string
	guildId int
}

func NewPlayer(id int, name string) *Player {
	return &Player{
		id:   id,
		name: name,
	}
}

func (p *Player) SetGuild(guildId int) {
	p.guildId = guildId
}

func (p *Player) CreateGuild(id int, name string) *Guild {
	guild := NewGuild(id, name)
	guild.AddPlayer(p)
	return guild
}

type Guild struct {
	id        int
	name      string
	playerMgr map[int]*Player
	deleted   bool
}

func NewGuild(id int, name string) *Guild {
	return &Guild{
		id:        id,
		name:      name,
		playerMgr: make(map[int]*Player),
	}
}

func (g *Guild) AddPlayer(p *Player) bool {
	_, o := g.playerMgr[p.id]
	if o {
		return false
	}
	p.guildId = g.id
	g.playerMgr[p.id] = p
	return true
}

func (g *Guild) RemovePlayer(id int) bool {
	p, o := g.playerMgr[id]
	if !o {
		return false
	}
	delete(g.playerMgr, id)
	p.guildId = 0
	return true
}

func (g *Guild) HasPlayer(id int) bool {
	_, o := g.playerMgr[id]
	return o
}

func (g *Guild) GetPlayer(id int) *Player {
	return g.playerMgr[id]
}

func (g *Guild) GetPlayerList() []int {
	var playerList []int
	for k := range g.playerMgr {
		playerList = append(playerList, k)
	}
	return playerList
}

func (g *Guild) SetDelete() {
	g.deleted = true
}

type GuildProc struct {
	guild *Guild
}

func NewGuildProc(id int, name string) *GuildProc {
	return &GuildProc{
		guild: NewGuild(id, name),
	}
}

func (g *GuildProc) Tick(tick int) {

}

type AddArg struct {
	player *Player
}

type AddReply struct {
	res bool
}

func (g *GuildProc) Add(arg *AddArg, reply *AddReply) error {
	res := g.guild.AddPlayer(arg.player)
	reply.res = res
	return nil
}

type RemoveArg struct {
	id int
}

type RemoveReply struct {
	res bool
}

func (g *GuildProc) Remove(arg *RemoveArg, reply *RemoveReply) error {
	res := g.guild.RemovePlayer(arg.id)
	reply.res = res
	return nil
}

type HasArg struct {
	id int
}

type HasReply struct {
	res bool
}

func (g *GuildProc) Has(arg *HasArg, reply *HasReply) error {
	res := g.guild.HasPlayer(arg.id)
	reply.res = res
	return nil
}

type GetArg struct {
	id int
}

type GetReply struct {
	player *Player
}

func (g *GuildProc) Get(arg *GetArg, reply *GetReply) error {
	player := g.guild.GetPlayer(arg.id)
	reply.player = player
	return nil
}

type GuildManager struct {
	guildList map[int]*gpc.GPC
	locker    *sync.Mutex
}

func NewGuildManager() *GuildManager {
	return &GuildManager{
		guildList: make(map[int]*gpc.GPC),
		locker:    &sync.Mutex{},
	}
}

func (g *GuildManager) AddGuild(guild *GuildProc) bool {
	g.locker.Lock()
	defer g.locker.Unlock()

	if gpcTmp, o := g.guildList[guild.guild.id]; o {
		guildTmp, ok := gpcTmp.GetServ().(*GuildProc)
		if !ok {
			return false
		}
		// 未删除
		if !guildTmp.guild.deleted {
			return false
		}
	}
	gpcGuild, err := gpc.NewGPC(guild)
	if err != nil {
		log.Printf("NewGPC err: %v", err)
		return false
	}
	g.guildList[guild.guild.id] = gpcGuild
	go gpcGuild.Run()

	return true
}

func (g *GuildManager) RemoveGuild(id int) bool {
	g.locker.Lock()
	defer g.locker.Unlock()

	var gpcGuild *gpc.GPC
	var o bool
	if gpcGuild, o = g.guildList[id]; !o {
		return false
	}

	var guild *GuildProc
	guild, o = gpcGuild.GetServ().(*GuildProc)
	if !o {
		return false
	}
	guild.guild.SetDelete()

	return true
}

func (g *GuildManager) GetGuild(id int) *gpc.GPC {
	g.locker.Lock()
	defer g.locker.Unlock()

	gpcGuild := g.guildList[id]
	return gpcGuild
}

func main() {
	guildIdMax := 100
	guildMgr := NewGuildManager()
	for id := 1; id <= guildIdMax; id++ {
		guildMgr.AddGuild(NewGuildProc(id, fmt.Sprintf("guild_%v", id)))
	}

	idList := make([]int, guildIdMax)
	randGuildIdFunc := func() int {
		var c int
		id := rand.Int() % guildIdMax
		for idList[id] == id+1 {
			id += 1
			if id >= len(idList) {
				id = 0
			}
			c += 1
			if c >= len(idList) {
				break
			}
		}
		idList[id] = id + 1
		return id + 1
	}

	for pid := 1; pid <= 1000; pid++ {
		p := NewPlayer(pid, fmt.Sprintf("player_%v", pid))
		addArg := &AddArg{player: p}
		addReply := &AddReply{}
		removeArg := &RemoveArg{id: pid}
		removeReply := &RemoveReply{}
		go func(id int) {
			for {
				guildId := randGuildIdFunc()
				gpcGuild := guildMgr.GetGuild(guildId)
				if gpcGuild == nil {
					log.Printf("cant get guild with id %v from GuildManager", guildId)
					continue
				}
				err := gpcGuild.Call("GuildProc.Add", addArg, addReply)
				if err != nil {
					log.Printf("guild add player %v err: %v", id, err)
					continue
				}
				if addReply.res {
					err = gpcGuild.Call("GuildProc.Remove", removeArg, removeReply)
					if err != nil {
						log.Printf("guild remove player %v err: %v", id, err)
						continue
					}
					if !removeReply.res {
						log.Printf("remove player %v from guild %v err: %v", id, guildId, err)
						continue
					}
				}
				time.Sleep(time.Millisecond * 200)
			}
		}(pid)
	}

	for {
		time.Sleep(time.Second)
	}
}
