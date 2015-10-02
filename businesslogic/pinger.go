package businesslogic

import (
	"fmt"
	"github.com/fithisux/gopinger/pinglogic"
	"net"
	"strconv"
	"sync"
	"time"
	"github.com/fithisux/orbit-dc-protector/utilities"
)

type Pingagent struct {
	updating_chan chan []utilities.OVPData
	Me            *net.UDPAddr
	conf          *pinglogic.TimedAttempts
	Mu            sync.Mutex
	Repinging     time.Duration
	pinged        []*net.UDPAddr
}

func CreatePingAgent(me *net.UDPAddr, repinging time.Duration, conf *pinglogic.TimedAttempts) *Pingagent {
	pa := new(Pingagent)
	pa.pinged = nil
	pa.Me = me
	pa.updating_chan = make(chan []utilities.OVPData)
	pa.conf = conf
	pa.Repinging = repinging
	go ping_them(pa)
	return pa
}

func (pa *Pingagent) isAlive() bool {

	var pinged []*net.UDPAddr
	pa.Mu.Lock()
	pinged = pa.pinged
	pa.Mu.Unlock()
	if pinged != nil || len(pinged) > 0 {
		fmt.Println("Liveness check")
		_, ok := pinglogic.Active(pa.conf, pa.Me, pinged)
		return ok
	} else {
		return false
	}
}

func ping_them(pa *Pingagent) {

	for lista := range pa.updating_chan {
		pinged := make([]*net.UDPAddr, len(lista))
		counter := 0
		for _, ovpdata := range lista {
			addressa := ovpdata.Odip + ":" + strconv.Itoa(ovpdata.Pingport)
			ra, err := net.ResolveUDPAddr("udp", addressa)
			//fmt.Println("Ping update on " + addressa)
			if err == nil {
				pinged[counter] = ra
				counter++
			}
		}
		pinged = pinged[:counter]
		pa.Mu.Lock()
		pa.pinged = pinged
		pa.Mu.Unlock()
	}
}
