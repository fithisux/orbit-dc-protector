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
	MongoTimeout = 60
	AuthDatabase = "orbitgoer"
)

type PersistencyLayer struct {
	dbconfig  *DBconfig
	dbsession *mgo.Session
}

func CreatePersistencyLayer(dbconfig *DBconfig) *PersistencyLayer {
	p := new(PersistencyLayer)
	p.dbconfig = dbconfig
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    dbconfig.Mongourls,
		Timeout:  MongoTimeout * time.Second,
		Database: AuthDatabase,
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

func (p *PersistencyLayer) Describe(exposeconfig *ExposeConfig) *OVPData {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	descriptor := new(OVPData)
	if err := collection.Find(exposeconfig.Ovpexpose).One(&descriptor); err == nil {
		return descriptor
	} else {
		if err.Error() == mgo.ErrNotFound.Error() {
			return nil
		} else {
			panic("Cannot describe because " + err.Error())
		}
	}
}

func (p *PersistencyLayer) InitializeOVP(exposeconfig *ExposeConfig) *OVPData {
	fmt.Println("searching for ovp watchdog in " + AuthDatabase)
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var ovpdata *OVPData
	if ovpdata := p.Describe(exposeconfig); ovpdata != nil {
		fmt.Println("found me")
		fmt.Printf("at epoch %d \n", ovpdata.Epoch)
		if ovpdata.ODPExpose.Dcprotecting != exposeconfig.Odpexpose.Dcprotecting {
			panic("Cannot flip roles")
		}
		ovpdata.Epoch++
		if err := collection.Update(exposeconfig.Ovpexpose, ovpdata); err != nil {
			panic("Cannot update epoch " + err.Error())
		}
	} else {
		ovpdata := new(OVPData)
		ovpdata.Epoch = 1
		ovpdata.OVPExpose = exposeconfig.Ovpexpose
		ovpdata.ODPExpose = exposeconfig.Odpexpose
		if err := collection.Insert(ovpdata); err != nil {
			panic("Cannot start epoch " + err.Error())
		}
	}
	return ovpdata
}

func (p *PersistencyLayer) InitializeODP(exposeconfig *ExposeConfig) *OVPData {
	if ovpdata := p.Describe(exposeconfig); ovpdata != nil {
		return ovpdata
	} else {
		panic("Cannot find odp watchdog")
	}
	return nil
}

func (p *PersistencyLayer) GetOVPPeers(bound int, ovpdata *OVPData) []OVPExpose {
	mySession := p.grabSession()
	defer mySession.Close()
	x := make([]OVPExpose, bound)
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var ovpdescriptor OVPData
	iter := collection.Find(bson.M{"watchdog_ovp_dcid": ovpdata.Dcid, "watchdog_ovp_operating": true}).Iter()
	counter := 0
	index := 0
	for iter.Next(&ovpdescriptor) {
		if ovpdescriptor.OVPExpose != ovpdata.OVPExpose {
			index++
			if counter < bound {
				x[counter] = ovpdescriptor.OVPExpose
				counter++
			} else {
				j := rand.Intn(index)
				j = j + 1
				if j < bound {
					x[j] = ovpdescriptor.OVPExpose
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

func (p *PersistencyLayer) GetODPPeers(dcid string) []OVPData {
	fmt.Println("odppreers for " + dcid)
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("watchdogs")
	var servers []OVPData
	if err := collection.Find(bson.M{"watchdog_ovp_dcid": dcid, "watchdog_ovp_operating": true, "watchdog_ovp_dcprotecting": true}).All(&servers); err != nil {
		panic("general error at ovptargets " + err.Error())
	} else {
		return servers
	}
}

func (p *PersistencyLayer) GetRoute(dcid string) string {
	mySession := p.grabSession()
	defer mySession.Close()
	collection := mySession.DB(AuthDatabase).C("routing")
	var candidate ODPRoute
	//fmt.Println("find route from "+p.Ovpdata.Dcid)
	if err := collection.Find(bson.M{"route_odp_src": dcid}).One(&candidate); err != nil {
		panic("general error at routes " + err.Error())
	}
	//fmt.Printf("%#v\n",candidate)
	return candidate.Dst
}
