package dcprotection

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fithisux/gopinger/pinglogic"
	"github.com/fithisux/orbit-dc-protector/utilities"
)

type Pingagent struct {
	updating_chan chan []utilities.OVPData
	Pingeraddress *net.UDPAddr
	conf          *pinglogic.TimedAttempts
	mutex         sync.Mutex
	Repinging     time.Duration
	pingtargets   []*net.UDPAddr
}

func CreatePingAgent(pingeraddress *net.UDPAddr, repinging time.Duration, conf *pinglogic.TimedAttempts) *Pingagent {
	pingagent := new(Pingagent)
	pingagent.pingtargets = nil
	pingagent.Pingeraddress = pingeraddress
	pingagent.updating_chan = make(chan []utilities.OVPData)
	pingagent.conf = conf
	pingagent.Repinging = repinging
	go ping_them(pingagent)
	return pingagent
}

func (pingagent *Pingagent) isAlive() bool {

	var pingtargets []*net.UDPAddr
	pingagent.mutex.Lock()
	pingtargets = pingagent.pingtargets
	pingagent.mutex.Unlock()
	if pingtargets != nil || len(pingtargets) > 0 {
		fmt.Println("Liveness check")
		_, ok := pinglogic.Active(pingagent.conf, pingagent.Pingeraddress, pingtargets)
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
		pingagent.mutex.Lock()
		pingagent.pingtargets = pingtargets
		pingagent.mutex.Unlock()
	}
}
