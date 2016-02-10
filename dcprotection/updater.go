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

type DetectorOpinion struct {
	Dcid         string
	Aliveopinion bool
}

type DBView struct {
	DetectorOpinion
	Me      *utilities.OPData
	Pingers []utilities.OPData
	Voters  []utilities.OPData
}

type Landscapeupdater struct {
	persistencylayer *utilities.PersistencyLayer
	updateinterval   time.Duration
	Dbupdates        chan *DBView
	opdata           *utilities.OPData
}

func CreateLandscapeupdater(conf *utilities.ServerConfig) *Landscapeupdater {
	landscapeupdater := new(Landscapeupdater)
	landscapeupdater.persistencylayer = utilities.CreatePersistencyLayer(&conf.Dbconfig)
	landscapeupdater.opdata = landscapeupdater.persistencylayer.InitializeODP(&conf.Opconfig)
	landscapeupdater.updateinterval = conf.Odpconfig.Landscapeupdateinterval
	landscapeupdater.Dbupdates = make(chan *DBView)

	go func() {
		ticker := time.NewTicker(landscapeupdater.updateinterval)
		for {
			select {
			case <-ticker.C:
				{
					fmt.Println("Updating")
					var pingers []utilities.OPData = nil
					var voters []utilities.OPData = nil
					dst := landscapeupdater.persistencylayer.GetRoute(landscapeupdater.opdata.Dcid)
					operating := landscapeupdater.persistencylayer.GetDatacenterState(dst)
					fmt.Printf("Got dst %s \n", dst)
					fmt.Println("Got operating %b \n", operating)
					if dst != "" {
						pingers = landscapeupdater.persistencylayer.GetODPPeers(dst)
						resultset := landscapeupdater.persistencylayer.GetODPPeers(landscapeupdater.opdata.Dcid)
						voters = make([]utilities.OPData, len(resultset))
						index := 0
						for i := 0; i < len(voters); i++ {
							if resultset[i].OPConfig != landscapeupdater.opdata.OPConfig {
								voters[index] = resultset[i]
								index++
							}
						}
						voters = voters[:index]
					}
					landscapeupdater.Dbupdates <- &DBView{DetectorOpinion{dst, operating}, landscapeupdater.opdata, pingers, voters}
					ticker = time.NewTicker(landscapeupdater.updateinterval)
				}
			}
		}
	}()
	return landscapeupdater
}
