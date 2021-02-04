package server

import (
	"encoding/json"
	"github.com/hawell/z42/internal/api/database"
	. "github.com/onsi/gomega"
	"io"
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
	connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"
	db *database.DataBase
)

func TestAddZone(t *testing.T) {
	initialize(t)
	url := generateURL("/zones", "user1")
	body := strings.NewReader(`{"name": "example.com.", "enabled": true, "dnssec":true, "cname_flattening": false}`)
	resp, err := http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// duplicate
	_, _ = body.Seek(0, io.SeekStart)
	resp, err = http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// bad request
	body = strings.NewReader(`"name"="example.com.", "enabled"=true, "dnssec"=true, "cname_flattening"=false`)
	resp, err = http.Post(url, "text/plain", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestGetZones(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addZone("user1", "zone2.com.")
	addZone("user1", "zone3.com.")
	addZone("user2", "zone4.com.")
	addZone("user2", "zone5.com.")
	addZone("user2", "zone6.com.")

	// get zones
	url := generateURL("/zones", "user1")
	resp, err := http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["zone1.com.","zone2.com.","zone3.com."]`)))

	// another user
	url = generateURL("/zones", "user2")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["zone4.com.","zone5.com.","zone6.com."]`)))

	// user with no zone
	url = generateURL("/zones", "user3")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))

	// non-existing user
	url = generateURL("/zones", "user4")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// limit results
	url = generateURL("/zones", "user2")
	url = url + "&start=1&count=1"
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["zone5.com."]`)))

	// with q
	url = generateURL("/zones", "user1")
	url = url + "&q=2"
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["zone2.com."]`)))

	// empty results
	url = generateURL("/zones", "user1")
	url = url + "&q=asasdas"
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))
}

func TestGetZone(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")

	// get zone
	url := generateURL("/zones/zone1.com.", "user1")
	resp, err := http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var z database.Zone
	err = json.Unmarshal(body, &z)
	Expect(err).To(BeNil())
	Expect(z.Name).To(Equal("zone1.com."))

	// non-existing zone
	url = generateURL("/zones/zone2.com.", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestUpdateZone(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	client := &http.Client{}

	// update zone
	url := generateURL("/zones/zone1.com.", "user1")
	body := strings.NewReader(`{"name": "zone1.com.", "enabled": true, "dnssec":true, "cname_flattening": false}`)
	req, err := http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var z database.Zone
	err = json.Unmarshal(respBody, &z)
	Expect(err).To(BeNil())
	Expect(z.Name).To(Equal("zone1.com."))
	Expect(z.Enabled).To(BeTrue())
	Expect(z.Dnssec).To(BeTrue())

	// non-existing zone
	url = generateURL("/zones/zone2.com.", "user1")
	body = strings.NewReader(`{"name": "zone2.com.", "enabled": true, "dnssec":true, "cname_flattening": false}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// zone name mismatch
	url = generateURL("/zones/zone1.com.", "user1")
	body = strings.NewReader(`{"name": "zone2.com.", "enabled": true, "dnssec":true, "cname_flattening": false}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestDeleteZone(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addZone("user2", "zone2.com.")
	client := &http.Client{}

	// delete zone
	url := generateURL("/zones/zone1.com.", "user1")
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	Expect(err).To(BeNil())
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing zone
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestAddLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	url := generateURL("/zones/zone1.com./locations", "user1")
	body := strings.NewReader(`{"name": "www", "enabled": true}`)
	resp, err := http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// duplicate
	_, _ = body.Seek(0, io.SeekStart)
	resp, err = http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// non-existing zone
	url = generateURL("/zones/zone2.com./locations", "user1")
	_, _ = body.Seek(0, io.SeekStart)
	resp, err = http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// bad request
	body = strings.NewReader(`"name"="www", "enabled"=true`)
	resp, err = http.Post(url, "text/plain", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestGetLocations(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addZone("user1", "zone2.com.")
	addZone("user1", "zone3.com.")
	addLocation("zone1.com.", "www1")
	addLocation("zone1.com.", "www2")
	addLocation("zone1.com.", "www3")
	addLocation("zone2.com.", "www4")
	addLocation("zone2.com.", "www5")
	addLocation("zone2.com.", "www6")

	// get locations
	url := generateURL("/zones/zone1.com./locations", "user1")
	resp, err := http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www1","www2","www3"]`)))

	// another zone
	url = generateURL("/zones/zone2.com./locations", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www4","www5","www6"]`)))

	// zone with no location
	url = generateURL("/zones/zone3.com./locations", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))

	// non-existing zone
	url = generateURL("/zones/zone4.com./locations", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// limit results
	url = generateURL("/zones/zone1.com./locations", "user1")
	url = url + "&start=1&count=1"
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www2"]`)))

	// with q
	url = generateURL("/zones/zone1.com./locations", "user1")
	url = url + "&q=2"
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www2"]`)))

	// empty results
	url = generateURL("/zones/zone1.com./locations", "user1")
	url = url + "&q=asasdas"
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))
}

func TestGetLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")

	// get location
	url := generateURL("/zones/zone1.com./locations/www", "user1")
	resp, err := http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var l database.Location
	err = json.Unmarshal(body, &l)
	Expect(err).To(BeNil())
	Expect(l.Name).To(Equal("www"))

	// non-existing location
	url = generateURL("/zones/zone1.com./locations/www2", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing zone
	url = generateURL("/zones/zone2.com./locations/www", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestUpdateLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	client := &http.Client{}

	// update location
	url := generateURL("/zones/zone1.com./locations/www", "user1")
	body := strings.NewReader(`{"name": "www", "enabled": false}`)
	req, err := http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var l database.Location
	err = json.Unmarshal(respBody, &l)
	Expect(err).To(BeNil())
	Expect(l.Name).To(Equal("www"))
	Expect(l.Enabled).To(BeFalse())

	// non-existing zone
	url = generateURL("/zones/zone2.com./locations/www", "user1")
	body = strings.NewReader(`{"name": "www", "enabled": true}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing location
	url = generateURL("/zones/zone1.com./locations/www2", "user1")
	body = strings.NewReader(`{"name": "www2", "enabled": true}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// location name mismatch
	url = generateURL("/zones/zone1.com./locations/www", "user1")
	body = strings.NewReader(`{"name": "www2", "enabled": true}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestDeleteLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	client := &http.Client{}

	// delete location
	url := generateURL("/zones/zone1.com./locations/www", "user1")
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	Expect(err).To(BeNil())
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing location
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing zone
	url = generateURL("/zones/zone2.com./locations/www", "user1")
	req, err = http.NewRequest(http.MethodDelete, url, nil)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestAddRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")

	// add record set
	url := generateURL("/zones/zone1.com./locations/www/rrsets", "user1")
	body := strings.NewReader(`{"type": "a", "enabled": true, "value": "{\"ttl\": 300, \"records\": [{\"ip\": \"1.2.3.4\"}]}"}`)
	resp, err := http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// duplicate
	_, _ = body.Seek(0, io.SeekStart)
	resp, err = http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// non-existing location
	url = generateURL("/zones/zone1.com./locations/www2/rrsets", "user1")
	_, _ = body.Seek(0, io.SeekStart)
	resp, err = http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing zone
	url = generateURL("/zones/zone2.com./locations/www/rrsets", "user1")
	_, _ = body.Seek(0, io.SeekStart)
	resp, err = http.Post(url, "application/json", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// bad request
	url = generateURL("/zones/zone1.com./locations/www/rrsets", "user1")
	body = strings.NewReader(`"ttl": 300, "records": [{"ip": "1.2.3.4"}]`)
	resp, err = http.Post(url, "text/plain", body)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestGetRecordSets(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addLocation("zone1.com.", "www2")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	addRecordSet("zone1.com.", "www", "aaaa", `{"ttl": 300, "records": [{"ip": "::1"}]}`)
	addZone("user1", "zone2.com.")
	addLocation("zone2.com.", "www")
	addRecordSet("zone2.com.", "www", "aname", `{"location": "aname.example.com."}`)
	addRecordSet("zone2.com.", "www", "cname", `{"ttl": 300, "host": "x.example.com."}`)
	addZone("user1", "zone3.com.")

	// get record sets
	url := generateURL("/zones/zone1.com./locations/www/rrsets", "user1")
	resp, err := http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["a","aaaa"]`)))

	// another zone
	url = generateURL("/zones/zone2.com./locations/www/rrsets", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["aname","cname"]`)))

	// location with no record sets
	url = generateURL("/zones/zone1.com./locations/www2/rrsets", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))

	// non-existing location
	url = generateURL("/zones/zone1.com./locations/www3/rrsets", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing zone
	url = generateURL("/zones/zone4.com./locations/www3/rrsets", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestGetRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)

	// get record set
	url := generateURL("/zones/zone1.com./locations/www/rrsets/a", "user1")
	resp, err := http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var r database.RecordSet
	err = json.Unmarshal(body, &r)
	Expect(err).To(BeNil())
	Expect(r.Type).To(Equal("a"))
	Expect(r.Value).To(Equal(`{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`))

	// non-existing record set
	url = generateURL("/zones/zone1.com./locations/www/rrsets/aaaa", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// invalid record type
	url = generateURL("/zones/zone1.com./locations/www/rrsets/adsds", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	// non-existing location
	url = generateURL("/zones/zone1.com./locations/www2/rrsets/a", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing zone
	url = generateURL("/zones/zone2.com./locations/www/rrsets/a", "user1")
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestUpdateRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	client := &http.Client{}

	// update record set
	url := generateURL("/zones/zone1.com./locations/www/rrsets/a", "user1")
	body := strings.NewReader(`{"type": "a", "enabled": true, "value": "{\"ttl\": 400, \"records\": [{\"ip\": \"1.2.3.5\"}]}"}`)
	req, err := http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp, err = http.Get(url)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var r database.RecordSet
	err = json.Unmarshal(respBody, &r)
	Expect(err).To(BeNil())
	Expect(r.Type).To(Equal("a"))
	Expect(r.Value).To(Equal(`{"ttl": 400, "records": [{"ip": "1.2.3.5"}]}`))

	// non-existing zone
	url = generateURL("/zones/zone2.com./locations/www/rrsets/a", "user1")
	body = strings.NewReader(`{"type": "a", "enabled": true, "value": "{\"ttl\": 300, \"records\": [{\"ip\": \"1.2.3.5\"}]}"}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing location
	url = generateURL("/zones/zone1.com./locations/www2/rrsets/a", "user1")
	body = strings.NewReader(`{"type": "a", "enabled": true, "value": "{\"ttl\": 300, \"records\": [{\"ip\": \"1.2.3.5\"}]}"}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing record set
	url = generateURL("/zones/zone1.com./locations/www/rrsets/aaaa", "user1")
	body = strings.NewReader(`{"type": "aaaa", "enabled": true, "value": "{\"ttl\": 300, \"records\": [{\"ip\": \"::1\"}]}"}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// record set type name mismatch
	url = generateURL("/zones/zone1.com./locations/www/rrsets/a", "user1")
	body = strings.NewReader(`{"type": "aaaa", "enabled": true, "value": "{\"ttl\": 300, \"records\": [{\"ip\": \"::1\"}]}"}`)
	req, err = http.NewRequest(http.MethodPut, url, body)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestDeleteRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	client := &http.Client{}

	// delete record set
	url := generateURL("/zones/zone1.com./locations/www/rrsets/a", "user1")
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	Expect(err).To(BeNil())
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing record set
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing zone
	url = generateURL("/zones/zone2.com./locations/www/rrsets/a", "user1")
	req, err = http.NewRequest(http.MethodDelete, url, nil)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// delete non-existing location
	url = generateURL("/zones/zone1.com./locations/www2/rrsets/a", "user1")
	req, err = http.NewRequest(http.MethodDelete, url, nil)
	Expect(err).To(BeNil())
	resp, err = client.Do(req)
	Expect(err).To(BeNil())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestMain(m *testing.M) {
	db, _ = database.Connect(connectionStr)
	s := NewServer(&serverConfig, db)
	go func() {
		_ = s.ListenAndServer()
	}()
	m.Run()
	_ = s.Shutdown()
	_ = db.Close()
}

func initialize(t *testing.T) {
	RegisterTestingT(t)
	_ = db.Clear()
	addUser("user1")
	addUser("user2")
	addUser("user3")
}

func addUser(name string) {
	_, err := db.AddUser(database.User{Name: name})
	if err != nil {
		panic(err)
	}
}

func addZone(user string, zone string) {
	_, err := db.AddZone(user, database.Zone{Name: zone})
	if err != nil {
		panic(err)
	}
}

func addLocation(zone string, location string) {
	_, err := db.AddLocation(zone, database.Location{Name: location})
	if err != nil {
		panic(err)
	}
}

func addRecordSet(zone string, location string, rtype string, recordset string) {
	_, err := db.AddRecordSet(zone, location, database.RecordSet{Type: rtype, Value: recordset})
	if err != nil {
		panic(err)
	}
}

func generateURL(path string, user string) string {
	return "http://" + serverConfig.BindAddress + path + "?user=" + user
}