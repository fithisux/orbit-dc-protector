package businesslogic

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

var (
	wg sync.WaitGroup
)

type Respondingdata struct {
	Failure error
	Data    []byte
}

type Tempresponder struct {
	r   *http.Response
	err error
}

// main is not changed
func SendToPartner(url string, reqbody []byte, timeout time.Duration, datanotification chan Respondingdata) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	wg.Add(1)
	go work(url, reqbody, ctx, datanotification)
	wg.Wait()
}

func work(url string, reqbody []byte, ctx context.Context, respbody chan Respondingdata) {
	defer wg.Done()

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	// anonymous struct to pack and unpack data in the channel
	c := make(chan struct {
		r   *http.Response
		err error
	}, 1)

	req, _ := http.NewRequest("GET", url, bytes.NewReader(reqbody))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	go func() {
		resp, err := client.Do(req)
		c <- Tempresponder{resp, err}
	}()

	select {
	case <-ctx.Done():
		tr.CancelRequest(req)
		fmt.Println("Wait for client.Do")
		<-c // Wait for client.Do
		fmt.Println("Cancel the context")
		respbody <- Respondingdata{ctx.Err(), nil}
		return
	case ok := <-c:
		fmt.Println("Got one")
		err := ok.err
		resp := ok.r
		if err != nil {
			fmt.Println("Error ", err)
			respbody <- Respondingdata{err, nil}
			return
		}
		fmt.Println("tight")
		defer resp.Body.Close()
		out, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Server Response: %s\n", out)
		fmt.Println("status " + ok.r.Status + " " + string(out))

		if ok.r.Status == "200 OK" {
			fmt.Println("it is ok status")
			respbody <- Respondingdata{nil, out}
			return
		} else {
			fmt.Println("it is not ok status")
			respbody <- Respondingdata{errors.New(ok.r.Status), nil}
			return
		}
	}
}
