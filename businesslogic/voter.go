package businesslogic

import (
	"encoding/json"
	"fmt"
	"time"
)

type Datacentergallop struct {
	Datacenter string `json:"Datacenter"`
}

func VotingProc(urls []string, datacenter string, votinginterval time.Duration) int {
	if len(urls) == 0 {
		return -1
	}

	reqbody, ok := json.Marshal(Datacentergallop{datacenter})
	if ok != nil {
		panic("Unmarshaller")
	}

	decision := 0
	c := make(chan Respondingdata)
	for _, url := range urls {
		fmt.Println("Request vote from " + url)
		go SendToPartner(url, reqbody, votinginterval, c)
	}
	building := -1 //i know you are dead
	candidates := 1
	opinion := new(DetectorStatus)
	counter := 0
	for resus := range c {
		fmt.Println("received something")
		if resus.Failure == nil {
			fmt.Println("received correct")
			if err := json.Unmarshal(resus.Data, opinion); err == nil {
				if opinion.Dcid == datacenter {
					if opinion.Alive {
						building++
					} else {
						building--
					}
					candidates++
				}
			} else {
				panic("Unmarshaller")
			}
		} else {
			fmt.Println("got failure")
		}
		counter++
		if counter == len(urls) {
			break
		}
	}

	close(c)

	active := 1 + ((len(urls) + 1) / 2)

	if candidates >= active {
		if candidates == -building {
			decision = -1
		} else if candidates == building {
			decision = 1
		}
	} else {
		panic("Not enough voters answered")
	}
	fmt.Printf("Vote == %d\n", decision)
	return decision
}
