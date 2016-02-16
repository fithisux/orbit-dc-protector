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
	"math/rand"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	MongoTimeout  = 60
	OrbitDatabase = "orbitgoer"
)

type PersistencyLayer struct {
	dbconfig  *DBconfig
	dbsession *mgo.Session
}

func CreatePersistencyLayer(dbconfig *DBconfig) *PersistencyLayer {
	p := new(PersistencyLayer)
	p.dbconfig = dbconfig
	fmt.Println("connecting")
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    dbconfig.Mongourls,
		Timeout:  MongoTimeout * time.Second,
		Database: "admin",
		Username: p.dbconfig.AuthUsername,
		Password: p.dbconfig.AuthPassword,
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
	p.dbsession = mongoSession
	return p
}

func (p *PersistencyLayer) grabSession() *mgo.Session {
	if mySession := p.dbsession.Copy(); mySession != nil {
		return mySession
	} else {
		panic("Failed to grab session")
	}
}

func (p *PersistencyLayer) Describe(opconfig *OPConfig) *OPData {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("watchdogs")
	descriptor := new(OPData)
	if err := collection.Find(opconfig).One(&descriptor); err == nil {
		return descriptor
	} else {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		} else {
			panic("Cannot describe because " + err.Error())
		}
	}
}

func (p *PersistencyLayer) InitializeOVP(opconfig *OPConfig) *OPData {
	fmt.Println("searching for ovp watchdog in " + OrbitDatabase)
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("watchdogs")
	var opdata *OPData
	if opdata = p.Describe(opconfig); opdata != nil {
		fmt.Println("found me")
		fmt.Printf("at epoch %d \n", opdata.Epoch)
		if opdata.Dcprotecting != opconfig.Dcprotecting {
			panic("Cannot flip roles")
		}
		opdata.Epoch++
		if err := collection.Update(opconfig, opdata); err != nil {
			panic("Cannot update epoch " + err.Error())
		}
	} else {
		opdata = new(OPData)
		opdata.Epoch = 1
		opdata.OPConfig = *opconfig
		if err := collection.Insert(opdata); err != nil {
			panic("Cannot start epoch " + err.Error())
		}
	}
	return opdata
}

func (p *PersistencyLayer) InitializeODP(opconfig *OPConfig) *OPData {
	if ovpdata := p.Describe(opconfig); ovpdata != nil {
		return ovpdata
	} else {
		panic("Cannot find odp watchdog")
	}
	return nil
}

func (p *PersistencyLayer) GetOVPPeers(bound int, opconfig *OPConfig) []OPData {
	mySession := p.grabSession()
	defer mySession.Close()
	x := make([]OPData, bound)
	collection := mySession.DB(OrbitDatabase).C("watchdogs")
	var opdescriptor OPData
	iter := collection.Find(bson.M{"watchdog_ovp_dcid": opconfig.Dcid}).Iter()
	counter := 0
	index := 0
	for iter.Next(&opdescriptor) {
		if opdescriptor.OPConfig != *opconfig {
			index++
			if counter < bound {
				x[counter] = opdescriptor
				counter++
			} else {
				j := rand.Intn(index)
				j = j + 1
				if j < bound {
					x[j] = opdescriptor
				}
			}
		}
	}

	if err := iter.Close(); err != nil {
		panic("Sampling produced an error " + err.Error())
	}

	if counter < bound {
		panic("Sampling produced insufficient elements")
	}

	return x
}

func (p *PersistencyLayer) GetODPPeers(dcid string) []OPData {
	fmt.Println("odppreers for " + dcid)
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("watchdogs")
	var servers []OPData
	if err := collection.Find(bson.M{"watchdog_ovp_dcid": dcid, "watchdog_ovp_dcprotecting": true}).All(&servers); err != nil {
		panic("general error at ovptargets " + err.Error())
	} else {
		return servers
	}
}

func (p *PersistencyLayer) GetRoute(dcid string) string {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("routing")
	var candidate OPRoute
	fmt.Println("find route from " + dcid)
	if err := collection.Find(bson.M{"route_odp_src": dcid}).One(&candidate); err != nil {
		panic("general error at GetRoute " + err.Error())
	}
	//fmt.Printf("%#v\n",candidate)
	return candidate.Dst
}

func (p *PersistencyLayer) GetDatacenterState(dcid string) *DetectorOpinion {
	if dcid == "" {
		return &DetectorOpinion{"", true}
	}
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("datacenters")
	var candidate DetectorOpinion
	fmt.Println("find state of " + dcid)
	if err := collection.Find(bson.M{"datacenter_id": dcid}).One(&candidate); err != nil {
		panic("general error at GetDatacenterState " + err.Error())
	}
	//fmt.Printf("%#v\n",candidate)
	return &candidate
}

func (p *PersistencyLayer) SetDatacenterFailed(dcid string) {
	if dcid == "" {
		return
	}
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("datacenters")
	fmt.Println("make failed " + dcid)
	if err := collection.Update(bson.M{"datacenter_id": dcid}, bson.M{"$set": bson.M{"datacenter_operating": false}}); err != nil {
		panic("general error at SetDatacenterFailed " + err.Error())
	}
}

func (p *PersistencyLayer) InsertVMDetection(vmdetection *VMDetection) {
	if vmdetection == nil {
		return
	}
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("vmdetections")
	if err := collection.Insert(vmdetection); err != nil {
		panic("general error at InsertVMDetection " + err.Error())
	}
}

func (p *PersistencyLayer) InsertDCDetection(dcdetection *DCDetection) {
	if dcdetection == nil {
		return
	}
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(OrbitDatabase).C("dcdetections")
	if err := collection.Insert(dcdetection); err != nil {
		panic("general error at InsertDCDetection " + err.Error())
	}
}
