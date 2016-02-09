package dcprotection

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/fithisux/gopinger/pinglogic"
	"github.com/fithisux/orbit-dc-protector/utilities"
)

type DetectorStatus struct {
	Dcid          string
	Aliveopinion  bool
	Alivedecision bool
	Parked        bool
}

type DetectorState struct {
	DetectorStatus
	landcsapeupdater *Landscapeupdater
	dbview           *DBView
}

var odpmutex sync.Mutex

type ODPdetector struct {
	state           *DetectorState
	odpconfig       *utilities.ODPconfig
	pinger          *Pingagent
	detectoraddress *net.UDPAddr
}

func CreateODPdetector(landscapeupdater *Landscapeupdater, serverconfig *utilities.ServerConfig) *ODPdetector {
	odpdetector := new(ODPdetector)
	odpdetector.state = new(DetectorState)
	odpdetector.state.Parked = true       //explicit
	odpdetector.state.Aliveopinion = true //explicit
	odpdetector.state.Dcid = ""           //explicit
	odpdetector.state.landcsapeupdater = landscapeupdater
	odpdetector.odpconfig = &serverconfig.Odpconfig
	detectoraddress, err := net.ResolveUDPAddr("udp", serverconfig.Opconfig.Odip+":"+strconv.Itoa(serverconfig.Opconfig.Pingport))
	if err != nil {
		panic(err.Error())
	}
	go pinglogic.Passive(detectoraddress)
	odpdetector.detectoraddress = detectoraddress
	return odpdetector
}

func (opdetector *ODPdetector) GetOpinion() *DetectorStatus {
	opinion := new(DetectorStatus)
	odpmutex.Lock()
	*opinion = opdetector.state.DetectorStatus
	odpmutex.Unlock()
	return opinion
}

func (opdetector *ODPdetector) Run() {
	pinger := CreatePingAgent(opdetector.detectoraddress, opdetector.odpconfig.Repinginterval, &opdetector.odpconfig.Pingattempts)
	state := opdetector.state

	var urls []string
	var dcid string

	go func() {
		for somedbview := range state.landcsapeupdater.Dbupdates {
			odpmutex.Lock()
			if state.Parked {
				if somedbview.Dcid != "" {
					if (state.dbview == nil) || (state.dbview.Dcid != somedbview.Dcid) {
						state.dbview = somedbview
						state.Parked = false
						state.Dcid = somedbview.Dcid
						state.Aliveopinion = true
						pinger.updating_chan <- state.dbview.Pingers
					}
				}
			} else {
				if state.dbview.Dcid == somedbview.Dcid {
					state.dbview = somedbview
					pinger.updating_chan <- state.dbview.Pingers
				} else {
					panic("change dst datacenter while not parked")
				}
			}
			odpmutex.Unlock()

		}
	}()

	suspicion := utilities.Createdcsuspicion()
	for {
		mustwait := false
		odpmutex.Lock()
		mustwait = state.Parked
		odpmutex.Unlock()

		if mustwait {
			fmt.Println("we are parked")
			time.Sleep(opdetector.odpconfig.Landscapeupdateinterval)
			continue
		}

		startping := time.Now()
		alive := pinger.isAlive()
		elapsedping := time.Since(startping)
		fmt.Printf("Elapsed ping : %s\n", elapsedping)

		odpmutex.Lock()
		state.Aliveopinion = alive
		odpmutex.Unlock()
		suspicion.Update(alive)
		if alive {
			fmt.Println("we are alive")
			continue
		} else {
			fmt.Println("we are down")
			fmt.Println("create voters")
			odpmutex.Lock()
			urls = make([]string, len(state.dbview.Voters))
			for i := 0; i < len(urls); i++ {
				urls[i] = "http://" + state.dbview.Voters[i].Ovip + ":" + strconv.Itoa(state.dbview.Voters[i].Voteport)
				urls[i] += "/odp/opinion"
			}
			dcid = state.dbview.Dcid
			odpmutex.Unlock()
			fmt.Println("voters created")
			if answers, decision := VotingProc(urls, dcid, opdetector.odpconfig.Votingthreshold, opdetector.odpconfig.Votingtimeout); decision == -1 {
				fmt.Println("datacenter announcement for " + dcid)
				fmt.Println("after " + strconv.Itoa(answers) + " answers out of " + strconv.Itoa(len(urls)))
				cmd := exec.Command("./faildc", dcid)
				var out bytes.Buffer
				cmd.Stdout = &out
				err := cmd.Run()
				if err != nil {
					fmt.Println("Notified? " + err.Error())
				}
				odpmutex.Lock()
				state.Parked = true
				odpmutex.Unlock()
				_, elapsedsuspicion := suspicion.Converged()
				fmt.Printf("Elapsed suspicion : %s\n", elapsedsuspicion)
			} else {
				fmt.Println("We are undecided")
				continue //undecided
			}
		}

	}
}
