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
// measurements.go
package utilities

import (
	"time"
)

type ProtectorMeasurements struct {
	DCProtector bool
	Ovip        string
	Dcid        string
	Timestamp   time.Time
	Windowvalue int
	Juststarted bool
	Epoch       int
}
type DCMeasurments struct {
	ProtectorMeasurements
	Targetdcid             string
	Internalfalsepositives int
	Externalfalsepositives int
	TimeToPing             int
	TimeToVote             int
	Pinged                 int
	Votings                int
}

type VMDetection struct {
	Vmid  string
	Ovip  string
	Epoch int
}

type VMMeasurments struct {
	Registeredservers int
	Detections        []VMDetection
}

type Dcsuspicion struct {
	continuousdown bool
	startsuspicion time.Time
}

func Createdcsuspicion() *Dcsuspicion {
	dd := new(Dcsuspicion)
	dd.Reset()
	return dd
}

func (dd *Dcsuspicion) Reset() {
	dd.continuousdown = false
	dd.startsuspicion = time.Now()
}

func (dd *Dcsuspicion) Update(status bool) {
	if status {
		dd.Reset()
	} else {
		if !dd.continuousdown {
			dd.continuousdown = true
		}
	}
}

func (dd *Dcsuspicion) Converged() (bool, time.Duration) {
	return dd.continuousdown, time.Since(dd.startsuspicion)
}
