package dcprotection

import (
	"bytes"
	"fmt"
	"net"
	"sync"

	"github.com/fithisux/gopinger/pinglogic"
	"github.com/fithisux/orbit-dc-protector/utilities"
	//"os"
	"os/exec"
	"strconv"
	"time"
)

type ParsedDetectorconfig struct {
	Pingattempts            *pinglogic.TimedAttempts
	Landscapeupdateinterval time.Duration
	Repinginterval          time.Duration
	Votingtimeout           time.Duration
	Detectoraddress         *net.UDPAddr
}

type DetectorStatus struct {
	Dcid         string
	Aliveopinion bool
	Parked       bool
}

type DetectorState struct {
	DetectorStatus
	landcsapeupdater *Landscapeupdater
	dbview           *DBView
}

var odpmutex sync.Mutex

type ODPdetector struct {
	state  *DetectorState
	config *ParsedDetectorconfig
	pinger *Pingagent
}

func CreateODPdetector(detectoraddress *net.UDPAddr, landscapeupdater *Landscapeupdater, odpconfig *utilities.ODPconfig) *ODPdetector {
	odpdetector := new(ODPdetector)
	odpdetector.state = new(DetectorState)
	odpdetector.state.Parked = true       //explicit
	odpdetector.state.Aliveopinion = true //explicit
	odpdetector.state.Dcid = ""           //explicit
	odpdetector.state.landcsapeupdater = landscapeupdater
	odpdetector.config = new(ParsedDetectorconfig)
	odpdetector.config.Landscapeupdateinterval = landscapeupdater.updateinterval
	odpdetector.config.Repinginterval = time.Duration(odpconfig.Repinginterval) * time.Millisecond
	odpdetector.config.Votingtimeout = time.Duration(odpconfig.Votinginterval) * time.Millisecond
	tempdur := time.Duration(odpconfig.Pingattempts.Timeout) * time.Millisecond
	odpdetector.config.Pingattempts = &pinglogic.TimedAttempts{tempdur, odpconfig.Pingattempts.Retries}
	odpdetector.config.Detectoraddress = detectoraddress
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
	pinger := CreatePingAgent(opdetector.config.Detectoraddress, opdetector.config.Repinginterval, opdetector.config.Pingattempts)
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

	continuousdown := false
	startsuspicion := time.Now()
	for {
		mustwait := false
		odpmutex.Lock()
		mustwait = state.Parked
		odpmutex.Unlock()

		if mustwait {
			fmt.Println("we are parked")
			time.Sleep(opdetector.config.Landscapeupdateinterval)
			continue
		}

		startping := time.Now()
		alive := pinger.isAlive()
		elapsedping := time.Since(startping)
		fmt.Printf("Elapsed ping : %s\n", elapsedping)

		odpmutex.Lock()
		state.Aliveopinion = alive
		odpmutex.Unlock()
		if alive {
			fmt.Println("we are alive")
			time.Sleep(opdetector.config.Repinginterval)
			continuousdown = false
			continue
		} else {
			fmt.Println("we are down")
			if !continuousdown {
				startsuspicion = time.Now()
				continuousdown = true
			}
		}

		fmt.Println("create voters")
		odpmutex.Lock()
		urls = make([]string, len(state.dbview.Voters))
		for i := 0; i < len(urls); i++ {
			urls[i] = "http://" + state.dbview.Voters[i].Ovip + ":" + strconv.Itoa(state.dbview.Voters[i].Voteport)
			urls[i] += "/dcprotector/opinion"
		}
		dcid = state.dbview.Dcid
		odpmutex.Unlock()
		fmt.Println("voters created")

		if decision := VotingProc(urls, dcid, opdetector.config.Votingtimeout); decision == -1 {
			fmt.Println("datacenter announcement for " + dcid)
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
			elapsedsuspicion := time.Since(startsuspicion)
			fmt.Printf("Elapsed suspicion : %s\n", elapsedsuspicion)
		} else {
			fmt.Println("We are undecided")
			continue //undecided
		}
	}
}
