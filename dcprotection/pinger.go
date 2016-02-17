package dcprotection

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/fithisux/gopinger/pinglogic"
	"github.com/fithisux/orbit-dc-protector/utilities"
)

type Pingagent struct {
	updating_chan chan []utilities.OPData
	pingconf      *pinglogic.PingConf
	pingtargets   []*net.UDPAddr
}

var pingagentmutex sync.Mutex

func CreatePingAgent(pingconf *pinglogic.PingConf) *Pingagent {
	pingagent := new(Pingagent)
	pingagent.pingtargets = nil
	pingagent.pingconf = pingconf
	pingagent.updating_chan = make(chan []utilities.OPData)
	go ping_them(pingagent)
	return pingagent
}

func (pingagent *Pingagent) isAlive() bool {

	var pingtargets []*net.UDPAddr
	pingagentmutex.Lock()
	pingtargets = pingagent.pingtargets
	pingagentmutex.Unlock()

	if pingtargets != nil || len(pingtargets) > 0 {
		fmt.Println("Liveness check")
		_, ok := pinglogic.Active(pingagent.pingconf, pingtargets)
		return len(ok.Answers) != 0
	} else {
		return false
	}
}

func ping_them(pingagent *Pingagent) {
	for lista := range pingagent.updating_chan {
		pingtargets := make([]*net.UDPAddr, len(lista))
		counter := 0
		for _, ovpdata := range lista {
			addressa := ovpdata.Odip + ":" + strconv.Itoa(ovpdata.Pingport)
			ra, err := net.ResolveUDPAddr("udp", addressa)
			//fmt.Println("Ping update on " + addressa)
			if err == nil {
				pingtargets[counter] = ra
				counter++
			}
		}
		pingtargets = pingtargets[:counter]
		pingagentmutex.Lock()
		pingagent.pingtargets = pingtargets
		pingagentmutex.Unlock()
	}
}
