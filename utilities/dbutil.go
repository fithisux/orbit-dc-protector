/*
===========================================================================
ORBIT VM PROTECTOR GPL Source Code
Copyright (C) 2015 Vasileios Anagnostopoulos.
This file is part of the ORBIT VM PROTECTOR Source Code (?ORBIT VM PROTECTOR Source Code?).
ORBIT VM PROTECTOR Source Code is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
ORBIT VM PROTECTOR Source Code is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.
You should have received a copy of the GNU General Public License
along with ORBIT VM PROTECTOR Source Code.  If not, see <http://www.gnu.org/licenses/>.
In addition, the ORBIT VM PROTECTOR Source Code is also subject to certain additional terms. You should have received a copy of these additional terms immediately following the terms and conditions of the GNU General Public License which accompanied the Doom 3 Source Code.  If not, please request a copy in writing from id Software at the address below.
If you have questions concerning this license or the applicable additional terms, you may contact in writing Vasileios Anagnostopoulos, Campani 3 Street, Athens Greece, POBOX 11252.
===========================================================================
*/

package utilities

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"os"
	"time"
	//"log"
)

const (
	MongoTimeout      = 60
	AuthDatabase      = "orbitgoer"
	Reservoircapacity = 5
)

type PersistencyLayer struct {
	dbconf      *DBconfig
	gigasession *mgo.Session
	Ovpdata     *OVPData
}

func CreatePersistencyODP(exposeconfig *ExposeConfig, aconfig *DBconfig) *PersistencyLayer {
	p := new(PersistencyLayer)
	p.Ovpdata = nil
	p.dbconf = aconfig
	p.createEndpoints()
	p.InitializeODP(exposeconfig)
	return p
}

func CreatePersistencyOVP(exposeconfig *ExposeConfig, aconfig *DBconfig) *PersistencyLayer {
	p := new(PersistencyLayer)
	p.Ovpdata = nil
	p.dbconf = aconfig
	p.createEndpoints()
	p.InitializeOVP()
	return p
}

func (p *PersistencyLayer) grabSession() *mgo.Session {
	if mySession := p.gigasession.Copy(); mySession != nil {
		return mySession
	} else {
		panic("Failed to grab session")
	}
}

func (p *PersistencyLayer) createEndpoints() {
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    p.dbconf.Mongourls,
		Timeout:  MongoTimeout * time.Second,
		Database: AuthDatabase,
		Username: p.dbconf.AuthUsername,
		Password: p.dbconf.AuthPassword,
	}
	fmt.Println("Dialing")

	/*
		Trace := log.New(os.Stdout,"TRACE: ",log.Ldate|log.Ltime|log.Lshortfile)
		mgo.SetLogger(Trace)
		mgo.SetDebug(true)
	*/
	mongoSession, err := mgo.DialWithInfo(mongoDBDialInfo)
	fmt.Println("Dialed")
	if err != nil {
		panic("Failed to create mongo session: " + err.Error())
	}
	if mongoSession == nil {
		panic("Nil mongo session: " + err.Error())
	}
	fmt.Println("we are connected")
	p.gigasession = mongoSession
}

func (p *PersistencyLayer) Changeweight(amount int) {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"watchdog_ovp_weight": amount}},
		ReturnNew: true,
	}
	var characterization OVPData
	_, err := collection.Find(p.Ovpdata.OVPExpose).Apply(change, &characterization)
	if err != nil {
		panic("Changeweight found no objects " + err.Error())
	}
}

func (p *PersistencyLayer) Makefailed(expose *OVPExpose) {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"watchdog_ovp_operating": false}},
		ReturnNew: true,
	}
	var characterization OVPData
	_, err := collection.Find(expose).Apply(change, &characterization)
	if err != nil {
		panic("Makefailed found no objects " + err.Error())
	}
}

func (p *PersistencyLayer) Describe() *OVPData {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var characterization OVPData
	err := collection.Find(p.Ovpdata.OVPExpose).One(&characterization)

	if err != nil {
		if err.Error() == mgo.ErrNotFound.Error() {
			fmt.Println("Could not find me. Exiting...")
			os.Exit(0)
		} else {
			panic("Cannot describe because " + err.Error())
		}
	} else {
		if !characterization.Operating {
			panic("operating == false")
		}
	}
	return &characterization
}

func (p *PersistencyLayer) SamplePeers() []OVPExpose {
	mySession := p.grabSession()
	defer mySession.Close()
	x := make([]OVPExpose, Reservoircapacity)
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var characterization OVPData

	iter := collection.Find(bson.M{"watchdog_odp_dcid": p.Ovpdata.Dcid}).Iter()
	counter := 0
	index := 0
	for iter.Next(&characterization) {
		if characterization.OVPExpose != p.Ovpdata.OVPExpose {
			index++
			if counter < Reservoircapacity {
				x[counter] = characterization.OVPExpose
				counter++
			} else {
				j := rand.Intn(index)
				j = j + 1
				if j < Reservoircapacity {
					x[j] = characterization.OVPExpose
				}
			}
		}
	}

	if err := iter.Close(); err != nil {
		panic("Sampling produced an error " + err.Error())
	}

	if counter < Reservoircapacity {
		panic("Sampling produced insufficient elements")
	}

	return x
}

func (p *PersistencyLayer) InitializeOVP() {
	mySession := p.grabSession()
	defer mySession.Close()
	fmt.Println("searching for watchdogs in " + AuthDatabase)
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var characterization OVPData
	fmt.Println("expose desc start")
	fmt.Println("%+v", p.Ovpdata)
	fmt.Println("expose desc stop")

	if err := collection.Find(p.Ovpdata.OVPExpose).One(&characterization); err == nil {
		fmt.Println("found me")
		fmt.Printf("at epoch %d \n", characterization.Epoch)
		if characterization.Operating {
			panic("Cannot initialize operating")
		}
		characterization.Epoch++
		characterization.Weight = 0
		characterization.Operating = true
		if err1 := collection.Update(p.Ovpdata.OVPExpose, &characterization); err1 != nil {
			panic("Cannot update epoch while found " + err1.Error())
		}
		p.Ovpdata.Epoch = characterization.Epoch
	} else {
		fmt.Println("error")
		if err.Error() == mgo.ErrNotFound.Error() {
			fmt.Println("not found me")
			if err1 := collection.Insert(&p); err1 != nil {
				panic("Cannot initialize epoch " + err1.Error())
			}
			fmt.Println("inserted")
		} else {
			panic("general error")
		}
	}
}

func (p *PersistencyLayer) InitializeODP(exposeconfig *ExposeConfig) {
	mySession := p.grabSession()
	defer mySession.Close()
	fmt.Println("searching for watchdogs in " + AuthDatabase)
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var characterization OVPData

	fmt.Println("expose desc start")
	temp1 := fmt.Sprintf("%#v", exposeconfig.Ovpexpose)
	fmt.Println(temp1)
	fmt.Println("expose desc stop")
	if err := collection.Find(exposeconfig.Ovpexpose).One(&characterization); err == nil {
		//if err := collection.Find(exposeconfig.Ovpexpose).One(&characterization); err == nil {
		fmt.Println("found me")
		p.Ovpdata = &characterization
		if !characterization.Operating {
			panic("operating == false")
		}
	} else {
		panic("Cannot find me " + err.Error())
	}
}

func (p *PersistencyLayer) GetOVPPeers(bound int) []OVPExpose {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var servers []OVPData
	if err := collection.Find(bson.M{"watchdog_odp_dcid": p.Ovpdata.Dcid}).Sort("-watchdog_weight").All(&servers); err != nil {
		panic("general error at ovptargets")
	} else {
		counter := 0
		watchers := make([]OVPExpose, bound)
		for i := 0; i < len(servers); i++ {
			fmt.Println("OVP targets")
			if (servers[i].OVPExpose != p.Ovpdata.OVPExpose) && (counter < len(watchers)) {
				fmt.Println("Watched by %+v", servers[i].OVPExpose)
				watchers[counter] = servers[i].OVPExpose
				counter++
			}
		}
		if counter < len(watchers) {
			panic("not much pingers")
		}
		return watchers
	}
}

func (p *PersistencyLayer) GetODPPeers(dcid string) []OVPData {
	//fmt.Println("odppreers for " + dcid)
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var servers []OVPData
	if err := collection.Find(bson.M{"watchdog_ovp_dcid": dcid}).All(&servers); err != nil {
		panic("general error at ovptargets " + err.Error())
	} else {
		counter := 0
		peers := make([]OVPData, len(servers))
		for i := 0; i < len(servers); i++ {
			//fmt.Printf("Recovered by %#v \n", servers[i])
			if (servers[i].Odip != "") && (servers[i].OVPExpose != p.Ovpdata.OVPExpose) {
				//fmt.Printf("store by %#v \n", servers[i].OVPExpose)
				peers[counter] = servers[i]
				counter++
			}
		}
		if counter == 0 {
			return nil
		} else {
			return peers[:counter]
		}
	}
}

func (p *PersistencyLayer) GetRoute() string {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("routing")
	var candidate ODPRoute
	//fmt.Println("find route from "+p.Ovpdata.Dcid)
	if err := collection.Find(bson.M{"route_odp_src": p.Ovpdata.Dcid}).One(&candidate); err != nil {
		panic("general error at routes " + err.Error())
	}
	//fmt.Printf("%#v\n",candidate)
	return candidate.Dst
}
