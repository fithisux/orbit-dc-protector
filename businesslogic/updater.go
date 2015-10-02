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

package businesslogic

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

type ODPupdater struct {
	persistence    *utilities.PersistencyLayer
	updateinterval time.Duration
	Updates        chan *DBView
}

func CreateODPupdater(conf *utilities.ServerConfig) *ODPupdater {
	odpu := new(ODPupdater)
	odpu.persistence = utilities.CreatePersistencyODP(&conf.Exposeconfig, &conf.Dbconfig)
	odpu.updateinterval = time.Duration(conf.Detectorconfig.Updateinterval) * time.Millisecond
	odpu.Updates = make(chan *DBView)

	p := odpu.persistence
	go func() {
		ticker := time.NewTicker(odpu.updateinterval)
		for {
			select {
			case <-ticker.C:
				{
					fmt.Println("Updating")

					p.Describe()
					var pingers []utilities.OVPData = nil
					var voters []utilities.OVPData = nil

					dst := p.GetRoute()
					//fmt.Println("Got dst "+dst)
					if dst != "" {
						pingers = p.GetODPPeers(dst)
						voters = p.GetODPPeers(p.Ovpdata.Dcid)
					}
					odpu.Updates <- &DBView{dst, p.Ovpdata, pingers, voters}
					ticker = time.NewTicker(odpu.updateinterval)
				}
			}
		}
	}()
	return odpu
}
