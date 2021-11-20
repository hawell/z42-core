package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/internal/api/handlers/auth"
	"github.com/hawell/z42/internal/api/handlers/recaptcha"
	"github.com/hawell/z42/internal/api/handlers/zone"
	"github.com/hawell/z42/internal/dnssec"
	"github.com/hawell/z42/internal/mailer"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/internal/upstream"
	jsoniter "github.com/json-iterator/go"
	"github.com/miekg/dns"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	serverConfig = Config{
		BindAddress:        "localhost:8080",
		ReadTimeout:        100,
		WriteTimeout:       100,
		MaxBodyBytes:       10000000,
		WebServer:          "z42.com",
		ApiServer:          "api.z42.com",
		NameServer:         "ns.z42.com.",
		HtmlTemplates:      "../../../templates/*.tmpl",
		RecaptchaSecretKey: "6LdNW6UcAAAAAL7M90WaPU2h4KwIveMuleVPMlkK",
		RecaptchaServer:    "http://127.0.0.1:9798",
	}
	connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"
	db            *database.DataBase
	client        *http.Client
	tokens        map[database.ObjectId]string
	users         = []database.User{
		{
			Email:    "apiUser1",
			Password: "apiUser1",
		},
		{
			Email:    "apiUser2",
			Password: "apiUser2",
		},
		{
			Email:    "apiUser3",
			Password: "apiUser3",
		},
	}
	recaptchaServer = recaptcha.NewMockServer("127.0.0.1:9798")
)

func TestAddZone(t *testing.T) {
	initialize(t)
	req := zone.NewZoneRequest{
		Name:            "example.com.",
		Enabled:         true,
		Dnssec:          true,
		CNameFlattening: false,
	}
	path := "/zones"

	// add zone
	body, err := jsoniter.Marshal(req)
	Expect(err).To(BeNil())
	resp := execRequest(users[0].Id, http.MethodPost, path, string(body))
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	z, err := db.GetZone(users[0].Id, "example.com.")
	Expect(err).To(BeNil())
	Expect(z.DS).To(MatchRegexp(`example.com.\s14400\sIN\sDS\s\d* 8 2 \w*`))
	z.DS = ""
	Expect(z).To(Equal(database.Zone{
		Id:              z.Id,
		Name:            "example.com.",
		Enabled:         true,
		Dnssec:          true,
		CNameFlattening: false,
		SOA:             *types.DefaultSOA("example.com."),
	}))
	rr, err := db.GetRecordSet(users[0].Id, "example.com.", "@", "ns")
	Expect(err).To(BeNil())
	ns := rr.Value.(*types.NS_RRSet)
	Expect(len(ns.Data)).To(Equal(2))
	Expect(ns.TtlValue).To(Equal(uint32(3600)))
	for i := range ns.Data {
		Expect(ns.Data[i].Host).To(MatchRegexp(`.*\.ns\.z42.com\.`))
	}

	// duplicate
	resp = execRequest(users[0].Id, http.MethodPost, path, string(body))
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// bad request
	body2 := `"name"="example.com.", "enabled"=true, "dnssec"=true, "cname_flattening"=false`
	resp = execRequest(users[0].Id, http.MethodPost, path, body2)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetZones(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	zone2Name := "zone2.com."
	_, err = addZone(users[0].Id, zone2Name)
	Expect(err).To(BeNil())
	zone3Name := "zone3.com."
	_, err = addZone(users[0].Id, zone3Name)
	Expect(err).To(BeNil())

	// get zones
	resp := execRequest(users[0].Id, http.MethodGet, "/zones", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response

	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 3,
			"items": [
				{"id": "zone1.com.", "enabled": true},
				{"id": "zone2.com.", "enabled": true},
				{"id": "zone3.com.", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// limit results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones?start=1&count=1", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 3,
			"items": [
				{"id": "zone2.com.", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// with q
	resp = execRequest(users[0].Id, http.MethodGet, "/zones?q=2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 1,
			"items": [
				{"id": "zone2.com.", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// empty results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones?q=asdas", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{"code": 200, "message": "successful", "data":{"items": [], "total": 0}}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// user with no zone
	resp = execRequest(users[1].Id, http.MethodGet, "/zones", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{"code": 200, "message": "successful", "data":{"items": [], "total": 0}}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// get zone
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	response.Data = &database.Zone{}
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data.(*database.Zone).DS).To(MatchRegexp(zone1Name + `\s14400\sIN\sDS\s\d* 8 2 \w*`))
	response.Data.(*database.Zone).DS = ""
	Expect(response.Data).To(Equal(&database.Zone{
		Name:            zone1Name,
		Enabled:         true,
		Dnssec:          false,
		CNameFlattening: false,
		SOA:             *types.DefaultSOA(zone1Name),
		DS:              "",
	}))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestUpdateZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// update zone
	resp := execRequest(users[0].Id, http.MethodPut, "/zones/"+zone1Name, `{"enabled": true, "dnssec":true, "cname_flattening": false, "soa": {"ttl": 300, "ns": "ns1.example.com.", "mbox": "admin.example.com.", "refresh": 44, "retry": 55, "expire": 66, "minttl": 100, "serial": 123456}}`)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	response.Data = &database.Zone{}
	err = json.Unmarshal(respBody, &response)
	Expect(err).To(BeNil())
	err = json.Unmarshal(respBody, &response)
	Expect(err).To(BeNil())
	Expect(response.Data.(*database.Zone).DS).To(MatchRegexp(zone1Name + `\s14400\sIN\sDS\s\d* 8 2 \w*`))
	response.Data.(*database.Zone).DS = ""
	Expect(response.Data).To(Equal(&database.Zone{
		Name:            zone1Name,
		Enabled:         true,
		Dnssec:          true,
		CNameFlattening: false,
		SOA: types.SOA_RRSet{
			GenericRRSet: types.GenericRRSet{TtlValue: 300},
			Ns:           "ns1.example.com.",
			MBox:         "admin.example.com.",
			Data:         nil,
			Refresh:      44,
			Retry:        55,
			Expire:       66,
			MinTtl:       100,
			Serial:       types.DefaultSOA(zone1Name).Serial + 1,
		},
		DS: "",
	}))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPut, "/zones/"+"invalid.none.", `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodPut, "/zones/"+zone1Name, `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestDeleteZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com"
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodDelete, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete non-existing zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestAddLocation(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// add location
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// duplicate
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+"invalid.none."+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// bad request
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `name: "www", enabled: true`)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetLocations(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	zone2Name := "zone2.com."
	_, err = addZone(users[0].Id, zone2Name)
	Expect(err).To(BeNil())
	zone3Name := "zone3.com."
	_, err = addZone(users[0].Id, zone3Name)
	Expect(err).To(BeNil())
	l1 := "www1"
	_, err = addLocation(users[0].Id, zone1Name, l1)
	Expect(err).To(BeNil())
	l2 := "www2"
	_, err = addLocation(users[0].Id, zone1Name, l2)
	Expect(err).To(BeNil())
	l3 := "www3"
	_, err = addLocation(users[0].Id, zone1Name, l3)
	Expect(err).To(BeNil())
	l4 := "www4"
	_, err = addLocation(users[0].Id, zone2Name, l4)
	Expect(err).To(BeNil())
	l5 := "www5"
	_, err = addLocation(users[0].Id, zone2Name, l5)
	Expect(err).To(BeNil())
	l6 := "www6"
	_, err = addLocation(users[0].Id, zone2Name, l6)
	Expect(err).To(BeNil())

	// get locations
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 4,
			"items": [
				{"id": "@", "enabled": true},
				{"id": "www1", "enabled": true},
				{"id": "www2", "enabled": true},
				{"id": "www3", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// another zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone2Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 4,
			"items": [
				{"id": "@", "enabled": true},
				{"id": "www4", "enabled": true},
				{"id": "www5", "enabled": true},
				{"id": "www6", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// zone with no location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone3Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`{"code":200,"message":"successful","data":{"items":[{"id":"@","enabled":true}],"total":1}}`)))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none."+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// limit results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations?start=1&count=1", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 4,
			"items": [
				{"id": "www1", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// with q
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations?q=2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 1,
			"items": [
				{"id": "www2", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// empty results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations?q=asdasd", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{"code":200,"message":"successful","data":{"total": 0, "items":[]}}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetLocation(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())

	// get location
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var l zone.GetLocationResponse
	err = json.Unmarshal(body, &l)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+"invalid", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none"+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestUpdateLocation(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())

	// update location
	resp := execRequest(users[0].Id, http.MethodPut, "/zones/"+zone1Name+"/locations/"+location1, `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"name": "www",
			"enabled": false
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodPut, "/zones/"+zone1Name+"/locations/"+location1, `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPut, "/zones/"+"invalid.none."+"/locations/"+location1, `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodPut, "/zones/"+zone1Name+"/locations/"+"invalid", `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestDeleteLocation(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete location
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete non-existing location
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+"invalid", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete non-existing zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+"invalid.none."+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestAddRecordSet(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	path := "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets"
	body := `{"type": "a", "enabled": true, "value": {"ttl": 300, "records": [{"ip": "1.2.3.4"}]}}`

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// add record set
	resp = execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// duplicate
	resp = execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets", body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+"invalid.none."+"/locations/"+location1+"/rrsets", body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// bad request
	body = `ttl: 300, records: {"ip": "1.2.3.4"}`
	resp = execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestAddLongText(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "default._domainkey"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	path := "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets"
	body := `{"type":"txt", "enabled":true, "value":{"ttl":300, "records":[{"text":"v=DKIM1;h=sha256;k=rsa;p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyApM1TW9+8LDXKGuqSbPUvLM5KN4+UraYalUXoZzX8JB33qxRrp/rMJfpx1RUei+cvw7WRFhLwZ5Ue0yNxQZ+RXsK0MYXGmdcqPLERu1GwJ61w4TEVJyox/++OoO/R/pa/cR/OS2i9d7tjqU8BZCB8o2MF0Sg9FN+FqFB3MzGgWrm2/kChBjA8QffIoSx7T8JuTlEf7pEf03gIIrMy4aYJxw+D0J777B0szlYdKyLRy7WqCcfzJNQU8AXtVX0IlmEdxkqst5IKzKpa3rjwYJ9/MtifcDWV47rdJEQ28Gi3HTmEXZ8L52ZukP1GztPg8hP5h5Y27GCx6WwC6zdlCz1wIDAQAB"}]}}`
	resp := execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	record, err := db.GetRecordSet(users[0].Id, zone1Name, location1, "txt")
	Expect(err).To(BeNil())
	Expect(record.Type).To(Equal("txt"))
	Expect(record.Enabled).To(BeTrue())
	Expect(record.Value).To(Equal(&types.TXT_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data:         []types.TXT_RR{{Text: "v=DKIM1;h=sha256;k=rsa;p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyApM1TW9+8LDXKGuqSbPUvLM5KN4+UraYalUXoZzX8JB33qxRrp/rMJfpx1RUei+cvw7WRFhLwZ5Ue0yNxQZ+RXsK0MYXGmdcqPLERu1GwJ61w4TEVJyox/++OoO/R/pa/cR/OS2i9d7tjqU8BZCB8o2MF0Sg9FN+FqFB3MzGgWrm2/kChBjA8QffIoSx7T8JuTlEf7pEf03gIIrMy4aYJxw+D0J777B0szlYdKyLRy7WqCcfzJNQU8AXtVX0IlmEdxkqst5IKzKpa3rjwYJ9/MtifcDWV47rdJEQ28Gi3HTmEXZ8L52ZukP1GztPg8hP5h5Y27GCx6WwC6zdlCz1wIDAQAB"}},
	}))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetRecordSets(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	location2 := "www2"
	_, err = addLocation(users[0].Id, zone1Name, location2)
	Expect(err).To(BeNil())
	r1 := "a"
	v1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, v1)
	Expect(err).To(BeNil())
	r2 := "aaaa"
	v2 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("::1"),
			},
		},
	}
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r2, v2)
	Expect(err).To(BeNil())
	zone2Name := "zone2.com."
	_, err = addZone(users[0].Id, zone2Name)
	Expect(err).To(BeNil())
	location3 := "www3"
	_, err = addLocation(users[0].Id, zone2Name, location3)
	Expect(err).To(BeNil())
	r3 := "aname"
	v3 := &types.ANAME_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Location:     "aname.example.com.",
	}
	_, err = addRecordSet(users[0].Id, zone2Name, location3, r3, v3)
	Expect(err).To(BeNil())
	r4 := "cname"
	v4 := &types.CNAME_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Host:         "x.example.com.",
	}
	_, err = addRecordSet(users[0].Id, zone2Name, location3, r4, v4)
	Expect(err).To(BeNil())

	// get record sets
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 2,
			"items": [
				{"id": "a", "enabled": true},
				{"id": "aaaa", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// another zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone2Name+"/locations/"+location3+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"total": 2,
			"items": [
				{"id": "aname", "enabled": true},
				{"id": "cname", "enabled": true}
			]
		}
	}`))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// location with no record sets
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location2+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(MatchJSON([]byte(`{"code":200,"message":"successful","data":{"items":[],"total":0}}`)))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none"+"/locations/"+location1+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetRecordSet(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	r1 := "a"
	v1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, v1)
	Expect(err).To(BeNil())

	// get record set
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(string(body)).To(MatchJSON(
		`{"code":200,"message":"successful","data":{"value":{"ttl":300,"filter":{},"health_check":{},"records":[{"ip":"1.2.3.4"}]},"enabled":true}}`,
	))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing record set
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+"tlsa", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none"+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestGetRecordSetWithEmptyRecords(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	recordsets := []struct {
		Type     string
		Expected string
	}{
		{"a", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"filter":{},"health_check":{},"records":[]},"enabled":true}}`},
		{"aaaa", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"filter":{},"health_check":{},"records":[]},"enabled":true}}`},
		{"txt", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
		{"ns", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
		{"mx", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
		{"srv", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
		{"caa", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
		{"tlsa", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
		{"ds", `{"code":200,"message":"successful","data":{"value":{"ttl":0,"records":[]},"enabled":true}}`},
	}
	for _, r := range recordsets {
		v := types.TypeStrToRRSet(r.Type)
		_, err = addRecordSet(users[0].Id, zone1Name, location1, r.Type, v)
		Expect(err).To(BeNil())
		resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r.Type, "")
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(string(body)).To(MatchJSON(r.Expected), r.Type)
		err = resp.Body.Close()
		Expect(err).To(BeNil())
	}
}

func TestUpdateRecordSet(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	r1 := "a"
	v1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, v1)
	Expect(err).To(BeNil())
	path := "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets/" + r1
	body := `{"enabled": true, "value": {"ttl": 400, "records": [{"ip": "1.2.3.5"}]}}`

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// update record set
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	resp = execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(string(respBody)).To(Equal(
		`{"code":200,"message":"successful","data":{"value":{"ttl":400,"filter":{},"health_check":{},"records":[{"ip":"1.2.3.5"}]},"enabled":true}}`,
	))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing zone
	path = "/zones/" + "invalid.none" + "/locations/" + location1 + "/rrsets/" + r1
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing location
	path = "/zones/" + zone1Name + "/locations/" + "invalid" + "/rrsets/" + r1
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// non-existing record set
	path = "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets/" + "tlsa"
	body = `{"enabled": true, "value": {"ttl": 400, "records": [{"ip": "::1"}]}}`
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// invalid record set
	path = "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets/" + "xxx"
	body = `{"enabled": true, "value": {"ttl": 400, "records": [{"ip": "::1"}]}}`
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestDeleteRecordSet(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	location1 := "www"
	_, err = addLocation(users[0].Id, zone1Name, location1)
	Expect(err).To(BeNil())
	r1 := "a"
	v1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, v1)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete record set
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete non-existing record set
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+"tlsa", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete non-existing zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+"invalid.none"+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// delete non-existing location
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestSignup(t *testing.T) {
	initialize(t)

	// add new user
	body := `{"email": "user1@example.com", "password": "password", "recaptcha_token": "123456"}`
	path := "/auth/signup"
	resp := execRequest("", http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	err := resp.Body.Close()
	Expect(err).To(BeNil())

	// check new user status is pending
	user, err := db.GetUser("user1@example.com")
	Expect(err).To(BeNil())
	Expect(user.Email).To(Equal("user1@example.com"))
	Expect(user.Status).To(Equal(database.UserStatusPending))

}

func TestVerify(t *testing.T) {
	initialize(t)

	_, code, err := db.AddUser(database.NewUser{
		Email:    "user2@example.com",
		Password: "12345678",
		Status:   database.UserStatusPending,
	})
	Expect(err).To(BeNil())

	// verify user
	path := "/auth/verify?code=" + code
	resp := execRequest(users[0].Id, http.MethodPost, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_, err = ioutil.ReadAll(resp.Body)
	// check user status is active
	user, err := db.GetUser("user2@example.com")
	Expect(err).To(BeNil())
	Expect(user.Email).To(Equal("user2@example.com"))
	Expect(user.Status).To(Equal(database.UserStatusActive))
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestRecover(t *testing.T) {
	initialize(t)

	path := "/auth/recover"
	body := fmt.Sprintf(`{"email": "%s", "recaptcha_token": "123456"}`, users[0].Email)
	resp := execRequest("", http.MethodPost, path, body)
	b, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK), string(b))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// should have a verification of type recover
	_, err = db.GetVerification(users[0].Id, database.VerificationTypeRecover)
	Expect(err).To(BeNil())

	// duplicate request
	resp = execRequest("", http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	// should overwrite existing code
	_, err = db.GetVerification(users[0].Id, database.VerificationTypeRecover)
	Expect(err).To(BeNil())
}

func TestReset(t *testing.T) {
	initialize(t)

	path := "/auth/recover"
	body := fmt.Sprintf(`{"email": "%s", "recaptcha_token": "123456"}`, users[0].Email)
	resp := execRequest("", http.MethodPost, path, body)
	b, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK), string(b))
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	code, err := db.GetVerification(users[0].Id, database.VerificationTypeRecover)
	Expect(err).To(BeNil())

	path = "/auth/reset"
	body = fmt.Sprintf(`{"password": "password2", "code": "%s", "recaptcha_token": "123456"}`, code)
	resp = execRequest(users[0].Id, http.MethodPatch, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	_, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	_, err = login(users[0].Email, "password2")
	Expect(err).To(BeNil())
}

func TestExportZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	path := "/zones/zone1.com./export"
	resp := execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	b, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())

	z, err := db.GetZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	Expect(z.DS).To(MatchRegexp(zone1Name + `\s14400\sIN\sDS\s\d* 8 2 \w*`))
	z.DS = ""
	Expect(z).To(Equal(database.Zone{
		Id:              z.Id,
		Name:            zone1Name,
		Enabled:         true,
		Dnssec:          false,
		CNameFlattening: false,
		SOA:             *types.DefaultSOA(zone1Name),
	}))
	rr, err := db.GetRecordSet(users[0].Id, zone1Name, "@", "ns")
	Expect(err).To(BeNil())
	ns := rr.Value.(*types.NS_RRSet)
	Expect(len(ns.Data)).To(Equal(2))
	Expect(ns.TtlValue).To(Equal(uint32(3600)))
	for i := range ns.Data {
		Expect(ns.Data[i].Host).To(MatchRegexp(`.*\.ns\.z42.com\.`))
	}

	parser := dns.NewZoneParser(strings.NewReader(string(b)), "", "")
	parser.SetIncludeAllowed(false)
	for fileRR, ok := parser.Next(); ok; fileRR, ok = parser.Next() {
		if fileRR.Header().Rrtype == dns.TypeSOA || (fileRR.Header().Rrtype == dns.TypeNS && fileRR.Header().Name == zone1Name) {
			continue
		}
		var label string
		if fileRR.Header().Name == zone1Name {
			label = "@"
		} else {
			label = strings.TrimSuffix(fileRR.Header().Name, "."+zone1Name)
		}
		dbRRset, err := db.GetRecordSet(users[0].Id, zone1Name, label, types.TypeToString(fileRR.Header().Rrtype))
		Expect(err).To(BeNil())
		dbRRs := dbRRset.Value.Value(fileRR.Header().Name)
		Expect(fileRR).To(BeElementOf(dbRRs))
	}
}

func TestImportZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	file := "testdata/" + zone1Name
	f, err := os.Open(file)
	Expect(err).To(BeNil())
	fw, err := w.CreateFormFile("file", file)
	Expect(err).To(BeNil())
	_, err = io.Copy(fw, f)
	Expect(err).To(BeNil())
	err = w.Close()
	Expect(err).To(BeNil())
	url := generateURL("/zones/" + zone1Name + "/import")
	req, err := http.NewRequest("POST", url, &b)
	Expect(err).To(BeNil())
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tokens[users[0].Id])
	req.Close = true
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	err = f.Close()
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(string(respBody)).To(MatchJSON(`{"code":200,"message":"successful"}`), string(respBody))

	z, err := db.GetZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	Expect(z.DS).To(MatchRegexp(zone1Name + `\s14400\sIN\sDS\s\d* 8 2 \w*`))
	z.DS = ""
	Expect(z).To(Equal(database.Zone{
		Id:              z.Id,
		Name:            zone1Name,
		Enabled:         true,
		Dnssec:          false,
		CNameFlattening: false,
		SOA:             *types.DefaultSOA(zone1Name),
	}))
	rr, err := db.GetRecordSet(users[0].Id, zone1Name, "@", "ns")
	Expect(err).To(BeNil())
	ns := rr.Value.(*types.NS_RRSet)
	Expect(len(ns.Data)).To(Equal(2))
	Expect(ns.TtlValue).To(Equal(uint32(3600)))
	for i := range ns.Data {
		Expect(ns.Data[i].Host).To(MatchRegexp(`.*\.ns\.z42.com\.`))
	}

	f, err = os.Open(file)
	Expect(err).To(BeNil())
	parser := dns.NewZoneParser(f, "", "")
	parser.SetIncludeAllowed(false)
	for fileRR, ok := parser.Next(); ok; fileRR, ok = parser.Next() {
		if fileRR.Header().Rrtype == dns.TypeSOA || (fileRR.Header().Rrtype == dns.TypeNS && fileRR.Header().Name == zone1Name) {
			continue
		}
		var label string
		if fileRR.Header().Name == zone1Name {
			label = "@"
		} else {
			label = strings.TrimSuffix(fileRR.Header().Name, "."+zone1Name)
		}
		dbRRset, err := db.GetRecordSet(users[0].Id, zone1Name, label, types.TypeToString(fileRR.Header().Rrtype))
		Expect(err).To(BeNil())
		dbRRs := dbRRset.Value.Value(fileRR.Header().Name)
		Expect(fileRR).To(BeElementOf(dbRRs))
	}
}

func TestAddAPIKey(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	body := `{"name": "api_key_1", "zone_name": "zone1.com.", "scope": "acme", "enabled": true}`
	path := "/auth/api_keys"
	resp := execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	respBody, err := io.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	var response handlers.Response
	response.Data = &auth.NewAPIKeyResponse{}
	err = jsoniter.Unmarshal(respBody, &response)
	Expect(err).To(BeNil())
	Expect(response.Data.(*auth.NewAPIKeyResponse).Key).NotTo(BeEmpty())
	response.Data.(*auth.NewAPIKeyResponse).Key = ""
	Expect(response.Data).To(Equal(&auth.NewAPIKeyResponse{
		Name:     "api_key_1",
		Key:      "",
		ZoneName: "zone1.com.",
		Scope:    "acme",
		Enabled:  true,
	}))
}

func TestGetAPIKeys(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	zone2Name := "zone2.com."
	zone3Name := "zone3.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	_, err = addZone(users[0].Id, zone2Name)
	Expect(err).To(BeNil())
	_, err = addZone(users[1].Id, zone3Name)
	Expect(err).To(BeNil())

	path := "/auth/api_keys"
	resp := execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := io.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	Expect(respBody).To(MatchJSON(`{
	  "code": 200,
	  "message": "successful",
	  "data": []
	}`))

	_, err = addAPIKey(users[0].Id, zone1Name, "key_1", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone1Name, "key_2", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone1Name, "key_3", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone2Name, "key_4", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone2Name, "key_5", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[1].Id, zone3Name, "key_1", "acme")
	Expect(err).To(BeNil())

	path = "/auth/api_keys"
	resp = execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err = io.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(respBody, &response)
	Expect(err).To(BeNil())
	Expect(respBody).To(MatchJSON(`{
	  "code": 200,
	  "message": "successful",
	  "data": [
		{
		  "name": "key_1",
		  "scope": "acme",
		  "zone_name": "zone1.com.",
		  "enabled": true
		},
		{
		  "name": "key_2",
		  "scope": "acme",
		  "zone_name": "zone1.com.",
		  "enabled": true
		},
		{
		  "name": "key_3",
		  "scope": "acme",
		  "zone_name": "zone1.com.",
		  "enabled": true
		},
		{
		  "name": "key_4",
		  "scope": "acme",
		  "zone_name": "zone2.com.",
		  "enabled": true
		},
		{
		  "name": "key_5",
		  "scope": "acme",
		  "zone_name": "zone2.com.",
		  "enabled": true
		}
	  ]
	}
	`))
}

func TestGetAPIKey(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	zone2Name := "zone2.com."
	zone3Name := "zone3.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())
	_, err = addZone(users[0].Id, zone2Name)
	Expect(err).To(BeNil())
	_, err = addZone(users[1].Id, zone3Name)
	Expect(err).To(BeNil())

	_, err = addAPIKey(users[0].Id, zone1Name, "key_1", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone1Name, "key_2", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone1Name, "key_3", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone2Name, "key_4", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[0].Id, zone2Name, "key_5", "acme")
	Expect(err).To(BeNil())
	_, err = addAPIKey(users[1].Id, zone3Name, "key_1", "acme")
	Expect(err).To(BeNil())

	path := "/auth/api_keys/key_1"
	resp := execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := io.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(respBody, &response)
	Expect(err).To(BeNil())
	Expect(respBody).To(MatchJSON(`{
	  "code": 200,
	  "message": "successful",
	  "data": {
        "name": "key_1",
        "scope": "acme",
        "zone_name": "zone1.com.",
        "enabled": true
      }
	}
	`))
}

func TestUpdateAPIKey(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	_, err = addAPIKey(users[0].Id, zone1Name, "key_1", "acme")
	Expect(err).To(BeNil())

	path := "/auth/api_keys/key_1"
	body := `{"scope": "acme", "enabled": false}`
	resp := execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	_, err = io.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())
}

func TestDeleteAPIKey(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	_, err = addAPIKey(users[0].Id, zone1Name, "key_1", "acme")
	Expect(err).To(BeNil())

	path := "/auth/api_keys/key_1"
	resp := execRequest(users[0].Id, http.MethodDelete, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	_, err = db.GetAPIKey(users[0].Id, "key_1")
	Expect(err).To(MatchError(database.ErrNotFound))
}

func TestGetActiveNS(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	path := "/zones/" + zone1Name + "/active_ns"
	resp := execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := io.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = resp.Body.Close()
	Expect(err).To(BeNil())
	Expect(respBody).To(MatchJSON(`{
		"code": 200,
		"message": "successful",
		"data": {
			"rcode":0,
			"hosts":["ns1.namefind.com.","ns2.namefind.com."]
		}
	}`))
}

func TestMain(m *testing.M) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableKeepAlives = true
	client = &http.Client{Transport: t, Timeout: time.Minute}
	recaptchaServer.HandlerFunc = func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		resp := recaptcha.Response{
			Success:     true,
			Score:       1.0,
			Action:      "login",
			ChallengeTS: time.Now(),
			Hostname:    "localhost:8080",
			ErrorCodes:  nil,
		}
		b, _ := jsoniter.Marshal(&resp)
		if _, err := writer.Write(b); err != nil {
			panic(err)
		}

	}
	go recaptchaServer.Start()
	var err error
	db, err = database.Connect(connectionStr)
	if err != nil {
		panic(err)
	}
	s := NewServer(
		&serverConfig,
		db,
		&mailer.Mock{
			SendEMailVerificationFunc: func(toName string, toEmail string, code string) error {
				return nil
			},
			SendPasswordResetFunc: func(toName string, toEmail string, code string) error {
				return nil
			},
		},
		upstream.NewUpstream([]upstream.Config{{
			Ip:       "1.1.1.1",
			Port:     53,
			Protocol: "udp",
			Timeout:  2000,
		}}),
		zap.L(),
	)
	go func() {
		err := s.ListenAndServer()
		if !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	err = db.Clear(true)
	if err != nil {
		panic(err)
	}
	tokens = make(map[database.ObjectId]string)
	for i := range users {
		users[i].Id, _, err = db.AddUser(database.NewUser{Email: users[i].Email, Password: users[i].Email, Status: database.UserStatusActive})
		if err != nil {
			panic(err)
		}
		token, err := login(users[i].Email, users[i].Password)
		if err != nil {
			panic(err)
		}
		tokens[users[i].Id] = token
	}
	m.Run()
	err = s.Shutdown()
	if err != nil {
		panic(err)
	}
	err = db.Close()
	if err != nil {
		panic(err)
	}
}

func generateURL(path string) string {
	return "http://" + serverConfig.BindAddress + path
}

func login(user string, password string) (string, error) {
	url := generateURL("/auth/login")
	body := strings.NewReader(fmt.Sprintf(`{"email":"%s", "password": "%s", "recaptcha_token": "123456"}`, user, password))
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return "", err
	}
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("3")
		return "", err
	}

	loginResp := make(map[string]interface{})
	err = json.Unmarshal(respBody, &loginResp)
	if err != nil {
		return "", err
	}
	if loginResp["code"].(float64) != 200 {
		fmt.Println(loginResp)
		return "", errors.New("login failed")
	}
	return loginResp["token"].(string), nil
}

func initialize(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(false)
	Expect(err).To(BeNil())
}

func addZone(userId database.ObjectId, zone string) (database.ObjectId, error) {
	Keys, err := dnssec.GenerateKeys(zone)
	Expect(err).To(BeNil())
	return db.AddZone(userId, database.NewZone{
		Name:    zone,
		Enabled: true,
		SOA:     *types.DefaultSOA(zone),
		Keys:    Keys,
		NS:      *types.GenerateNS(serverConfig.NameServer),
	})
}

func addLocation(userId database.ObjectId, zoneName string, location string) (database.ObjectId, error) {
	return db.AddLocation(userId, database.NewLocation{ZoneName: zoneName, Location: location, Enabled: true})
}

func addRecordSet(userId database.ObjectId, zoneName string, location string, recordType string, recordset types.RRSet) (database.ObjectId, error) {
	return db.AddRecordSet(userId, database.NewRecordSet{ZoneName: zoneName, Location: location, Enabled: true, Type: recordType, Value: recordset})
}

func addAPIKey(userId database.ObjectId, zoneName string, name string, scope string) (string, error) {
	return db.AddAPIKey(userId, database.APIKeyItem{
		Name:     name,
		Scope:    scope,
		ZoneName: zoneName,
		Enabled:  true,
	})
}

func execRequest(userId database.ObjectId, method string, path string, body string) *http.Response {
	url := generateURL(path)
	reqBody := strings.NewReader(body)
	req, err := http.NewRequest(method, url, reqBody)
	Expect(err).To(BeNil())
	req.Close = true
	req.Header.Add("Content-Type", "application/json")
	if userId != "" {
		req.Header.Add("Authorization", "Bearer "+tokens[userId])
	}
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	return resp
}
