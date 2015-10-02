package businesslogic

import (
	"sync"
	"bytes"
	"fmt"
	"github.com/fithisux/gopinger/pinglogic"
	"github.com/fithisux/orbit-dc-protector/utilities"
	"net"
	//"os"
	"os/exec"
	"strconv"
	"time"
)

type PasedDetectorconfig struct {
	Pingattempts   *pinglogic.TimedAttempts
	Updateinterval time.Duration
	Repinginterval time.Duration
	Votinginterval time.Duration
	Me             *net.UDPAddr
}


type DetectorStatus struct {
	Dcid   string
	Alive  bool
	Parked bool
}

type DetectorState struct {
	DetectorStatus
	odpu          *ODPupdater
	Workingdbview *DBView
	mu            sync.Mutex
}

type OPDetector struct {
	state  *DetectorState
	config *PasedDetectorconfig
	pinger *Pingagent
}

func CreateOPDetector(ra *net.UDPAddr, odpu *ODPupdater, detectorconfig *utilities.Detectorconfig) *OPDetector {
	opd := new(OPDetector)
	opd.state = new(DetectorState)
	opd.state.Parked = true //explicit
	opd.state.Alive = true  //explicit
	opd.state.Dcid = ""     //explicit
	opd.state.odpu = odpu
	opd.config = new(PasedDetectorconfig)
	opd.config.Updateinterval = odpu.updateinterval
	opd.config.Repinginterval = time.Duration(detectorconfig.Repinginterval) * time.Millisecond
	opd.config.Votinginterval = time.Duration(detectorconfig.Votinginterval) * time.Millisecond
	tempdur := time.Duration(detectorconfig.Pingattempts.Timeout) * time.Millisecond
	opd.config.Pingattempts = &pinglogic.TimedAttempts{tempdur, detectorconfig.Pingattempts.Retries}
	opd.config.Me = ra
	return opd
}

func (opdetector *OPDetector) GetOpinion() *DetectorStatus {
	opinion := new(DetectorStatus)
	opdetector.state.mu.Lock()
	*opinion = opdetector.state.DetectorStatus
	opdetector.state.mu.Unlock()
	return opinion
}

func (opdetector *OPDetector) Run() {
	pinger := CreatePingAgent(opdetector.config.Me, opdetector.config.Repinginterval, opdetector.config.Pingattempts)
	state := opdetector.state

	var urls []string
	var dcid string

	go func() {
		for somedbview := range state.odpu.Updates {
			state.mu.Lock()
			if state.Parked {
				if somedbview.Dcid != "" {
					if (state.Workingdbview == nil) || (state.Workingdbview.Dcid != somedbview.Dcid) {
						state.Workingdbview = somedbview
						state.Parked = false
						state.Dcid = somedbview.Dcid
						state.Alive = true
						pinger.updating_chan <- state.Workingdbview.Pingers
					}
				}
			} else {
				if state.Workingdbview.Dcid == somedbview.Dcid {
					state.Workingdbview = somedbview
					pinger.updating_chan <- state.Workingdbview.Pingers
				} else {
					panic("change dst datacenter while not parked")
				}
			}
			state.mu.Unlock()

		}
	}()

	for {
		mustwait := false
		state.mu.Lock()
		mustwait = state.Parked
		state.mu.Unlock()

		if mustwait {
			fmt.Println("we are parked")
			time.Sleep(opdetector.config.Updateinterval)
			continue
		}

		alive := pinger.isAlive()
		state.mu.Lock()
		state.Alive = alive
		state.mu.Unlock()
		if alive {
			fmt.Println("we are alive")
			time.Sleep(opdetector.config.Repinginterval)
			continue
		} else {
			fmt.Println("we are down")
			//os.Exit(0)
		}

		fmt.Println("create voters")
		state.mu.Lock()
		urls = make([]string, len(state.Workingdbview.Voters))
		for i := 0; i < len(urls); i++ {
			urls[i] = "http://" + state.Workingdbview.Voters[i].Odip + ":" + strconv.Itoa(state.Workingdbview.Voters[i].Voteport)
			urls[i] += "/dcprotector/opinion"
		}
		dcid = state.Workingdbview.Dcid
		state.mu.Unlock()
		fmt.Println("voters created")

		if decision := VotingProc(urls, dcid, opdetector.config.Votinginterval); decision == -1 {
			fmt.Println("datacenter announcement for " + dcid)
			cmd := exec.Command(*utilities.Notifyaction, dcid)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				fmt.Println("Notified? " + err.Error())
			}
			state.mu.Lock()
			state.Parked = true
			state.mu.Unlock()
		} else {
			fmt.Println("We are undecided")
			continue //undecided
		}
	}
}
