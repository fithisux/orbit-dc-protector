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

package dcprotection

import (
	"fmt"
	"time"

	"github.com/fithisux/orbit-dc-protector/utilities"
)

type DBView struct {
	Dcid    string
	Me      *utilities.OVPData
	Pingers []utilities.OVPData
	Voters  []utilities.OVPData
}

type Landscapeupdater struct {
	persistencylayer *utilities.PersistencyLayer
	updateinterval   time.Duration
	Dbupdates        chan *DBView
	ovpdata          *utilities.OVPData
}

func CreateLandscapeupdater(conf *utilities.ServerConfig) *Landscapeupdater {
	landscapeupdater := new(Landscapeupdater)
	landscapeupdater.persistencylayer = utilities.CreatePersistencyLayer(&conf.Dbconfig)
	landscapeupdater.ovpdata = landscapeupdater.persistencylayer.InitializeODP(&conf.Exposeconfig)
	landscapeupdater.updateinterval = time.Duration(conf.Odpconfig.Updateinterval) * time.Millisecond
	landscapeupdater.Dbupdates = make(chan *DBView)

	go func() {
		ticker := time.NewTicker(landscapeupdater.updateinterval)
		for {
			select {
			case <-ticker.C:
				{
					fmt.Println("Updating")
					var pingers []utilities.OVPData = nil
					var voters []utilities.OVPData = nil
					dst := landscapeupdater.persistencylayer.GetRoute(landscapeupdater.ovpdata.Dcid)
					//fmt.Println("Got dst "+dst)
					if dst != "" {
						pingers = landscapeupdater.persistencylayer.GetODPPeers(dst)
						resultset := landscapeupdater.persistencylayer.GetODPPeers(landscapeupdater.ovpdata.Dcid)
						voters = make([]utilities.OVPData, len(resultset))
						index := 0
						for i := 0; i < len(voters); i++ {
							if resultset[i].OVPExpose != landscapeupdater.ovpdata.OVPExpose {
								voters[index] = resultset[i]
								index++
							}
						}
						voters = voters[:index]
					}
					landscapeupdater.Dbupdates <- &DBView{dst, landscapeupdater.ovpdata, pingers, voters}
					ticker = time.NewTicker(landscapeupdater.updateinterval)
				}
			}
		}
	}()
	return landscapeupdater
}
