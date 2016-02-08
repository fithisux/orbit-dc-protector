package dcprotection

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Datacentergallop struct {
	Datacenter string `json:"Datacenter"`
}

func VotingProc(urls []string, datacenter string, votingthreshold float64, votingtimeout time.Duration) (int, int) {
	if len(urls) == 0 {
		return 0, -1
	}

	reqbody, ok := json.Marshal(Datacentergallop{datacenter})
	if ok != nil {
		panic("Unmarshaller")
	}

	responses := make(chan Respondingdata)
	for _, url := range urls {
		fmt.Println("Request vote from " + url)
		go SendToPartner(url, reqbody, votingtimeout, responses)
	}
	building := -1 //i know you are dead
	answers := 0
	opinion := new(DetectorStatus)
	for resus := range responses {
		fmt.Println("received something")
		if resus.Failure == nil {
			fmt.Println("received correct")
			if err := json.Unmarshal(resus.Data, opinion); err == nil {
				if opinion.Dcid == datacenter {
					if opinion.Aliveopinion {
						building++
					} else {
						building--
					}
					answers++
				}
			} else {
				panic("Unmarshaller")
			}
		} else {
			fmt.Println("got failure")
		}
	}

	close(responses)

	threshold := int(math.Ceil(0.5*float64(len(urls)) + 0.5))
	decision := 0
	if answers >= threshold {
		threshold = int(math.Ceil(votingthreshold * float64(answers)))
		if building < 0 {
			if -building >= threshold {
				decision = -1
			}
		} else {
			if building >= threshold {
				decision = 1
			}
		}
	} else {
		panic("Not enough voters answered")
	}
	fmt.Printf("Vote == %d\n", decision)
	return answers, decision
}
