package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers"
	"github.com/hawell/z42/internal/api/handlers/zone"
	"github.com/hawell/z42/internal/types"
	"github.com/hawell/z42/pkg/hiredis"
	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var (
	serverConfig = Config{
		BindAddress:  "localhost:8080",
		ReadTimeout:  10,
		WriteTimeout: 10,
	}
	redisConfig = hiredis.Config{
		Address:  "127.0.0.1:6379",
		Net:      "tcp",
		DB:       0,
		Password: "",
		Prefix:   "test_",
		Suffix:   "_test",
		Connection: hiredis.ConnectionConfig{
			MaxIdleConnections:   10,
			MaxActiveConnections: 10,
			ConnectTimeout:       500,
			ReadTimeout:          500,
			IdleKeepAlive:        30,
			MaxKeepAlive:         0,
			WaitForConnection:    false,
		},
	}
	connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"
	db            *database.DataBase
	redis         *hiredis.Redis
	tokens        map[database.ObjectId]string
	client        http.Client
	users         = []database.User{
		{
			Email:    "user1",
			Password: "user1",
		},
		{
			Email:    "user2",
			Password: "user2",
		},
		{
			Email:    "user3",
			Password: "user3",
		},
	}
)

func TestAddZone(t *testing.T) {
	initialize(t)
	req := zone.NewZoneRequest{
		Name:            "example.com.",
		Enabled:         true,
		Dnssec:          true,
		CNameFlattening: false,
		SOA:             types.SOA_RRSet{
			GenericRRSet: types.GenericRRSet{TtlValue: 3600},
			Ns:           "ns1.example.com.",
			MBox:         "mail.example.com.",
			Refresh:      3600,
			Retry:        3600,
			Expire:       3600,
			MinTtl:       3600,
			Serial:       3600,
		},
		NS:              types.NS_RRSet{
			GenericRRSet: types.GenericRRSet{TtlValue: 3600},
			Data:         []types.NS_RR{
				{Host: "ns1.example.com."},
				{Host: "ns2.example.com."},
			},
		},
	}
	path := "/zones"

	// add zone
	body, err := jsoniter.Marshal(req)
	Expect(err).To(BeNil())
	resp := execRequest(users[0].Id, http.MethodPost, path, string(body))
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// duplicate
	resp = execRequest(users[0].Id, http.MethodPost, path, string(body))
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// bad request
	body2 := `"name"="example.com.", "enabled"=true, "dnssec"=true, "cname_flattening"=false`
	resp = execRequest(users[0].Id, http.MethodPost, path, body2)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
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
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "zone1.com.",
		},
		{
			"id": "zone2.com.",
		},
		{
			"id": "zone3.com.",
		},
	}))

	// limit results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones?start=1&count=1", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "zone2.com.",
		},
	}))

	// with q
	resp = execRequest(users[0].Id, http.MethodGet, "/zones?q=2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "zone2.com.",
		},
	}))

	// empty results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones?q=asdas", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(BeEmpty())

	// user with no zone
	resp = execRequest(users[1].Id, http.MethodGet, "/zones", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(BeEmpty())
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
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(Equal(map[string]interface{}{
		"name":             "zone1.com.",
		"enabled":          false,
		"dnssec":           false,
		"cname_flattening": false,
	}))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestUpdateZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// update zone
	resp := execRequest(users[0].Id, http.MethodPut, "/zones/"+zone1Name, `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(respBody, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(Equal(map[string]interface{}{
		"name":             "zone1.com.",
		"enabled":          true,
		"dnssec":           true,
		"cname_flattening": false,
	}))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPut, "/zones/"+"invalid.none.", `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodPut, "/zones/"+zone1Name, `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestDeleteZone(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com"
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodDelete, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// delete zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestAddLocation(t *testing.T) {
	initialize(t)
	zone1Name := "zone1.com."
	_, err := addZone(users[0].Id, zone1Name)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// add location
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// duplicate
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+"invalid.none."+"/locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// bad request
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations", `name: "www", enabled: true`)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
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
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "www1",
		},
		{
			"id": "www2",
		},
		{
			"id": "www3",
		},
	}))

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// another zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone2Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "www4",
		},
		{
			"id": "www5",
		},
		{
			"id": "www6",
		},
	}))

	// zone with no location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone3Name+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`{"code":200,"message":"successful","data":[{"id":"@"}]}`)))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none."+"/locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// limit results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations?start=1&count=1", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "www1",
		},
	}))

	// with q
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations?q=2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": "www2",
		},
	}))

	// empty results
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations?q=asdasd", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`{"code":200,"message":"successful","data":[]}`)))
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

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+"invalid", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invali.none"+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
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
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var l zone.GetLocationResponse
	err = json.Unmarshal(respBody, &l)
	Expect(err).To(BeNil())
	Expect(l.Enabled).To(BeFalse())

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodPut, "/zones/"+zone1Name+"/locations/"+location1, `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPut, "/zones/"+"invalid.none."+"/locations/"+location1, `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodPut, "/zones/"+zone1Name+"/locations/"+"invalid", `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
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
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// delete location
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing location
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+"invalid", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+"invalid.none."+"/locations/"+location1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
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
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// add record set
	resp = execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// duplicate
	resp = execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets", body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodPost, "/zones/"+"invalid.none."+"/locations/"+location1+"/rrsets", body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// bad request
	body = `ttl: 300, records: {"ip": "1.2.3.4"}`
	resp = execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
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
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	Expect(err).To(BeNil())
	r2 := "aaaa"
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r2, `{"ttl": 300, "records": [{"ip": "::1"}]}`)
	Expect(err).To(BeNil())
	zone2Name := "zone2.com."
	_, err = addZone(users[0].Id, zone2Name)
	Expect(err).To(BeNil())
	location3 := "www3"
	_, err = addLocation(users[0].Id, zone2Name, location3)
	Expect(err).To(BeNil())
	r3 := "aname"
	_, err = addRecordSet(users[0].Id, zone2Name, location3, r3, `{"location": "aname.example.com."}`)
	Expect(err).To(BeNil())
	r4 := "cname"
	_, err = addRecordSet(users[0].Id, zone2Name, location3, r4, `{"ttl": 300, "host": "x.example.com."}`)
	Expect(err).To(BeNil())

	// get record sets
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var response handlers.Response
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": r1,
		},
		{
			"id": r2,
		},
	}))

	// another zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone2Name+"/locations/"+location3+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &response)
	Expect(err).To(BeNil())
	Expect(response.Data).To(ContainElements([]map[string]interface{}{
		{
			"id": r3,
		},
		{
			"id": r4,
		},
	}))

	// location with no record sets
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location2+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`{"code":200,"message":"successful","data":[]}`)))

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none"+"/locations/"+location1+"/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
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
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	Expect(err).To(BeNil())

	// get record set
	resp := execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(string(body)).To(Equal(
		`{"code":200,"message":"successful","data":{"value":{"ttl":300,"filter":{},"health_check":{},"records":[{"ip":"1.2.3.4"}]},"enabled":true}}`,
	))

	// unauthorized
	resp = execRequest(users[1].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// non-existing record set
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+"tlsa", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing location
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing zone
	resp = execRequest(users[0].Id, http.MethodGet, "/zones/"+"invalid.none"+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
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
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	Expect(err).To(BeNil())
	path := "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets/" + r1
	body := `{"enabled": true, "value": {"ttl": 400, "records": [{"ip": "1.2.3.5"}]}}`

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// update record set
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp = execRequest(users[0].Id, http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(string(respBody)).To(Equal(
		`{"code":200,"message":"successful","data":{"value":{"ttl":400,"filter":{},"health_check":{},"records":[{"ip":"1.2.3.5"}]},"enabled":true}}`,
	))

	// non-existing zone
	path = "/zones/" + "invalid.none" + "/locations/" + location1 + "/rrsets/" + r1
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing location
	path = "/zones/" + zone1Name + "/locations/" + "invalid" + "/rrsets/" + r1
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing record set
	path = "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets/" + "tlsa"
	body = `{"enabled": true, "value": {"ttl": 400, "records": [{"ip": "::1"}]}}`
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// invalid record set
	path = "/zones/" + zone1Name + "/locations/" + location1 + "/rrsets/" + "xxx"
	body = `{"enabled": true, "value": {"ttl": 400, "records": [{"ip": "::1"}]}}`
	resp = execRequest(users[0].Id, http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
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
	_, err = addRecordSet(users[0].Id, zone1Name, location1, r1, `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	Expect(err).To(BeNil())

	// unauthorized
	resp := execRequest(users[1].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))

	// delete record set
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing record set
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+location1+"/rrsets/"+"tlsa", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing zone
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+"invalid.none"+"/locations/"+location1+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing location
	resp = execRequest(users[0].Id, http.MethodDelete, "/zones/"+zone1Name+"/locations/"+"invalid"+"/rrsets/"+r1, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestSignupAndVerify(t *testing.T) {
	initialize(t)
	err := redis.Del("email_verification")
	Expect(err).To(BeNil())

	// add new user
	body := `{"email": "user1@example.com", "password": "password"}`
	path := "/auth/signup"
	resp := execRequest(users[0].Id, http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// check new user status is pending
	user, err := db.GetUser("user1@example.com")
	Expect(err).To(BeNil())
	Expect(user.Email).To(Equal("user1@example.com"))
	Expect(user.Status).To(Equal(database.UserStatusPending))

	// get verification code
	item, err := redis.XRead("email_verification", "0")
	Expect(err).To(BeNil())
	Expect(len(item)).To(Equal(1))
	code := item[0].Value

	// verify user
	path = "/auth/verify?code=" + code
	resp = execRequest(users[0].Id, http.MethodPost, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// check user status is active
	user, err = db.GetUser("user1@example.com")
	Expect(err).To(BeNil())
	Expect(user.Email).To(Equal("user1@example.com"))
	Expect(user.Status).To(Equal(database.UserStatusActive))
}

func TestMain(m *testing.M) {
	var err error
	db, err = database.Connect(connectionStr)
	if err != nil {
		panic(err)
	}
	redis = hiredis.NewRedis(&redisConfig)
	s := NewServer(&serverConfig, db, redis)
	go func() {
		_ = s.ListenAndServer()
	}()
	err = db.Clear(true)
	if err != nil {
		panic(err)
	}
	tokens = make(map[database.ObjectId]string)
	for i := range users {
		users[i].Id, err = db.AddUser(database.NewUser{Email: users[i].Email, Password: users[i].Email, Status: database.UserStatusActive})
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
	body := strings.NewReader(fmt.Sprintf(`{"email":"%s", "password": "%s"}`, user, password))
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	loginResp := make(map[string]interface{})
	err = json.Unmarshal(respBody, &loginResp)
	if err != nil {
		return "", err
	}
	if loginResp["code"].(float64) != 200 {
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
	soa := types.SOA_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 3600},
		Ns:           "ns1.example.com.",
		MBox:         "mail.example.com.",
		Data:         nil,
		Refresh:      3600,
		Retry:        3600,
		Expire:       3600,
		MinTtl:       3600,
		Serial:       3600,
	}
	ns := types.NS_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 3600},
		Data:         []types.NS_RR{
			{Host: "ns1.example.com."},
			{Host: "ns2.example.com."},
		},
	}
	return db.AddZone(userId, database.NewZone{Name: zone}, soa, ns)
}

func addLocation(userId database.ObjectId, zoneName string, location string) (database.ObjectId, error) {
	return db.AddLocation(userId, zoneName, database.NewLocation{Name: location})
}

func addRecordSet(userId database.ObjectId, zoneName string, location string, recordType string, recordset string) (database.ObjectId, error) {
	return db.AddRecordSet(userId, zoneName, location, database.NewRecordSet{Enabled: true, Type: recordType, Value: recordset})
}

func execRequest(userId database.ObjectId, method string, path string, body string) *http.Response {
	url := generateURL(path)
	reqBody := strings.NewReader(body)
	req, err := http.NewRequest(method, url, reqBody)
	Expect(err).To(BeNil())
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+tokens[userId])
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	return resp
}
