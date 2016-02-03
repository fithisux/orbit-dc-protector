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
// jsonutil.go
package utilities

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
)

type ODPRoute struct {
	Src string `bson:"route_odp_src"`
	Dst string `bson:"route_odp_dst"`
}

type ODPExpose struct {
	Odip     string `bson:"watchdog_odp_ip" json:"watchdog_odp_ip"`
	Voteport int    `bson:"watchdog_odp_voteport" json:"watchdog_odp_voteport"`
	Pingport int    `bson:"watchdog_odp_pingport" json:"watchdog_odp_pingport"`
}

type OVPExpose struct {
	Ovip         string `bson:"watchdog_ovp_ip" json:"watchdog_ovp_ip"`
	Serfport     int    `bson:"watchdog_ovp_serfport" json:"watchdog_ovp_serfport"`
	Announceport int    `bson:"watchdog_ovp_announceport"  json:"watchdog_ovp_announceport"`
	Dcid         string `bson:"watchdog_ovp_dcid" json:"watchdog_ovp_dcid"`
}

func (oe *OVPExpose) Name() string {
	bb, err := json.Marshal(oe)
	if err != nil {
		panic(err)
	}

	return string(bb)
}

type OVPData struct {
	OVPExpose    `bson:",inline"`
	Epoch        int  `bson:"watchdog_ovp_epoch" json:"watchdog_ovp_epoch"`
	Operating    bool `bson:"watchdog_ovp_operating" json:"watchdog_ovp_operating"`
	Dcprotecting bool `bson:"watchdog_ovp_dcprotecting" json:"watchdog_ovp_dcprotecting"`
	ODPExpose    `bson:",inline"`
}

type OrbitAttempts struct {
	Retries int `json:"retries"`
	Timeout int `json:"timeout"`
}

type Detectorconfig struct {
	Pingattempts   OrbitAttempts `json:"pingattempts"`
	Updateinterval int64         `json:"updateinterval"`
	Repinginterval int64         `json:"repinginterval"`
	Votinginterval int64         `json:"votinginterval"`
}

type DBconfig struct {
	Mongourls    []string `json:"mongourls"`
	AuthUsername string   `json:"username"`
	AuthPassword string   `json:"password"`
}

type ExposeConfig struct {
	Ovpexpose OVPExpose `json:"ovpexpose"`
	Odpexpose ODPExpose `json:"odpexpose"`
}

type ServerConfig struct {
	Exposeconfig   ExposeConfig   `json:"exposeconfig"`
	Detectorconfig Detectorconfig `json:"detectorconfig"`
	Dbconfig       DBconfig       `json:"dbconfig"`
	Numofwatchers  int            `json:"numofpeers"`
	Minwatchers    int            `json:"minpeers"`
}

var jsonfile *string

func init() {
	log.Println("Inside init")
	jsonfile = flag.String("jsonfile", "", "the json configuration file")
	if jsonfile == nil {
		panic("shitty jsonfile")
	}
}

func validateJson(content []byte) (*ServerConfig, error) {

	var err error

	var data ServerConfig
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, err
	}

	if data.Numofwatchers <= 0 {
		return nil, errors.New("option : Numofwatchers is a positive integer.")
	}

	if data.Detectorconfig.Updateinterval <= 0 {
		return nil, errors.New("option : Updateinterval is a positive integer.")
	}

	if data.Detectorconfig.Votinginterval <= 0 {
		return nil, errors.New("option : Votinginterval is a positive integer.")
	}

	if data.Detectorconfig.Repinginterval <= 0 {
		return nil, errors.New("option : Repinginterval is a positive integer.")
	}

	if data.Detectorconfig.Pingattempts.Retries <= 0 {
		return nil, errors.New("option : Retries is a positive integer.")
	}

	if data.Detectorconfig.Pingattempts.Timeout <= 0 {
		return nil, errors.New("option : Timeout is a positive integer.")
	}

	if data.Dbconfig.Mongourls == nil || len(data.Dbconfig.Mongourls) == 0 {
		return nil, errors.New("zero mongourls")
	}

	return &data, nil
}

func Parsetoconf() (*ServerConfig, error) {
	flag.Parse()

	if info, err := os.Stat(*jsonfile); err == nil && !info.IsDir() {
		content, err2 := ioutil.ReadFile(*jsonfile)
		if err2 != nil {
			return nil, err2
		} else {
			return validateJson(content)
		}
	} else {
		if err != nil {
			return nil, err
		} else {
			return nil, errors.New("jsonpath is not a file")
		}
	}
}
