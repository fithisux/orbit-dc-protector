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

var odpmutex sync.Mutex

type ODPdetector struct {
	landcsapeupdater *Landscapeupdater
	dbview           *DBView
	opinion          *utilities.DetectorOpinion
	odpconfig        *utilities.ODPconfig
	pinger           *Pingagent
	serverconfig     *utilities.ServerConfig
	pingconf         *pinglogic.PingConf
	Backchannel      chan *pinglogic.Backcall
}

func CreateODPdetector(landscapeupdater *Landscapeupdater, serverconfig *utilities.ServerConfig) *ODPdetector {
	odpdetector := new(ODPdetector)
	odpdetector.serverconfig = serverconfig
	odpdetector.landcsapeupdater = landscapeupdater
	odpdetector.opinion = new(utilities.DetectorOpinion)
	odpdetector.opinion.Aliveopinion = true //explicit
	odpdetector.opinion.Dcid = ""           //explicit
	odpdetector.odpconfig = &serverconfig.Odpconfig
	odpdetector.pingconf = new(pinglogic.PingConf)
	if detectoraddress, err := net.ResolveUDPAddr("udp", serverconfig.Opconfig.Odip+":"+strconv.Itoa(serverconfig.Opconfig.Pingport)); err != nil {
		panic(err.Error())
	} else {
		odpdetector.pingconf.Timingconf = &serverconfig.Odpconfig.Pingattempts
		odpdetector.pingconf.Backaddress = detectoraddress
		odpdetector.Backchannel = make(chan *pinglogic.Backcall)
		odpdetector.pingconf.Backchannel = odpdetector.Backchannel
	}
	go pinglogic.Passive(odpdetector.pingconf.Backaddress, odpdetector.Backchannel)
	return odpdetector
}

func (opdetector *ODPdetector) GetOpinion() *utilities.DetectorOpinion {
	opinion := new(utilities.DetectorOpinion)
	odpmutex.Lock()
	*opinion = *opdetector.opinion
	odpmutex.Unlock()
	return opinion
}

func (opdetector *ODPdetector) Run() {
	pinger := CreatePingAgent(opdetector.pingconf)

	go func() {
		for somedbview := range opdetector.landcsapeupdater.Dbupdates {
			odpmutex.Lock()
			if opdetector.dbview == nil { //parked
				if somedbview.Dcid != "" && somedbview.Aliveopinion {
					fmt.Println("started")
					opdetector.dbview = somedbview
					*opdetector.opinion = somedbview.DetectorOpinion
					pinger.updating_chan <- opdetector.dbview.Pingers
				} else {
					fmt.Println("still waiting")
				}
			} else {
				if opdetector.dbview.DetectorOpinion == somedbview.DetectorOpinion {
					opdetector.dbview = somedbview
					pinger.updating_chan <- opdetector.dbview.Pingers
				} else {
					panic("change opinion while not parked")
				}
			}
			odpmutex.Unlock()
		}
	}()

	suspicion := utilities.Createdcsuspicion()
	for {
		odpmutex.Lock()
		mustwait := (opdetector.dbview == nil)
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
		opdetector.opinion.Aliveopinion = alive
		odpmutex.Unlock()
		suspicion.Update(alive)
		if alive {
			fmt.Println("we are alive")
			continue
		} else {
			fmt.Println("we are down")
			fmt.Println("create voters")
			odpmutex.Lock()
			dcid := opdetector.dbview.Dcid
			urls := make([]string, len(opdetector.dbview.Voters))
			for i := 0; i < len(urls); i++ {
				urls[i] = "http://" + opdetector.dbview.Voters[i].Ovip + ":" + strconv.Itoa(opdetector.dbview.Voters[i].Voteport)
				urls[i] += "/odp/opinion"
			}
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

				dcdetection := new(utilities.DCDetection)
				dcdetection.Reporter_ovip = opdetector.serverconfig.Opconfig.Odip
				dcdetection.Reporter_dcid = opdetector.serverconfig.Opconfig.Dcid
				dcdetection.Dcid = dcid
				dcdetection.Timestamp = time.Now()
				opdetector.landcsapeupdater.persistencylayer.InsertDCDetection(dcdetection)

				odpmutex.Lock()
				opdetector.dbview = nil
				odpmutex.Unlock()
				opdetector.landcsapeupdater.persistencylayer.SetDatacenterFailed(opdetector.opinion.Dcid)
				_, elapsedsuspicion := suspicion.Converged()
				fmt.Printf("Elapsed suspicion : %s\n", elapsedsuspicion)
			} else {
				fmt.Println("We are undecided")
				continue //undecided
			}
		}

	}
}
