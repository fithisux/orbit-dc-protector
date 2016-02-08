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
// orbit-vm-protector.go
package main

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/fithisux/gopinger/pinglogic"
	"github.com/fithisux/orbit-dc-protector/dcprotection"
	"github.com/fithisux/orbit-dc-protector/utilities"
)

var detector *dcprotection.ODPdetector

func dcprotector_opinion(request *restful.Request, response *restful.Response) { //stop a stream
	log.Printf("Inside dcprotector_opinion")
	response.WriteEntity(detector.GetOpinion())
}

func main() {

	conf, err := utilities.Parsetoconf()
	if err != nil {
		panic(err.Error())
	}

	landscapeupdater := dcprotection.CreateLandscapeupdater(conf)
	detector = dcprotection.CreateODPdetector(landscapeupdater, &conf)
	go detector.Run()
	wsContainer := restful.NewContainer()
	log.Printf("Registering")
	ws := new(restful.WebService)
	ws.Path("/odp").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/opinion").To(dcprotector_opinion)).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	wsContainer.Add(ws)

	// Add container filter to enable CORS
	/*
		cors := restful.CrossOriginResourceSharing{
			ExposeHeaders:  []string{"X-My-Header"},
			AllowedHeaders: []string{"Content-Type"},
			CookiesAllowed: false,
			Container:      wsContainer}
		wsContainer.Filter(cors.Filter)

		// Add container filter to respond to OPTIONS
		wsContainer.Filter(wsContainer.OPTIONSFilter)
	*/

	log.Printf("start listening on localhost:%d\n", conf.Opconfig.Voteport)
	server := &http.Server{Addr: conf.Opconfig.Ovip + ":" + strconv.Itoa(conf.Opconfig.Voteport), Handler: wsContainer}
	log.Fatal(server.ListenAndServe())
}
