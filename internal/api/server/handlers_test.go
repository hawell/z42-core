package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hawell/z42/internal/api/database"
	"github.com/hawell/z42/internal/api/handlers/zone"
	"github.com/hawell/z42/pkg/hiredis"
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
	token         string
	client        http.Client
)

func TestAddZone(t *testing.T) {
	initialize(t)
	body := `{"name": "example.com.", "enabled": true, "dnssec":true, "cname_flattening": false}`
	path := "/zones"

	// add zone
	resp := execRequest(http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// duplicate
	resp = execRequest(http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// bad request
	body = `"name"="example.com.", "enabled"=true, "dnssec"=true, "cname_flattening"=false`
	resp = execRequest(http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
}

func TestGetZones(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addZone("user1", "zone2.com.")
	addZone("user1", "zone3.com.")

	// get zones
	resp := execRequest(http.MethodGet, "/zones", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var items zone.ListResponse
	err = json.Unmarshal(body, &items)
	Expect(err).To(BeNil())
	Expect(items).To(ContainElements("zone1.com.","zone2.com.","zone3.com."))

	// limit results
	resp = execRequest(http.MethodGet, "/zones?start=1&count=1", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["zone2.com."]`)))

	// with q
	resp = execRequest(http.MethodGet, "/zones?q=2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["zone2.com."]`)))

	// empty results
	resp = execRequest(http.MethodGet, "/zones?q=asdas", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))

	// user with no zone
	_, err = db.DeleteZone("zone1.com.")
	Expect(err).To(BeNil())
	_, err = db.DeleteZone("zone2.com.")
	Expect(err).To(BeNil())
	_, err = db.DeleteZone("zone3.com.")
	Expect(err).To(BeNil())
	resp = execRequest(http.MethodGet, "/zones", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))
}

func TestGetZone(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")

	// get zone
	resp := execRequest(http.MethodGet, "/zones/zone1.com.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var z zone.GetZoneResponse
	err = json.Unmarshal(body, &z)
	Expect(err).To(BeNil())
	Expect(z.Name).To(Equal("zone1.com."))

	// non-existing zone
	resp = execRequest(http.MethodGet, "/zones/zone2.com.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestUpdateZone(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")

	// update zone
	resp := execRequest(http.MethodPut, "/zones/zone1.com.", `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp = execRequest(http.MethodGet, "/zones/zone1.com.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var z zone.GetZoneResponse
	err = json.Unmarshal(respBody, &z)
	Expect(err).To(BeNil())
	Expect(z.Name).To(Equal("zone1.com."))
	Expect(z.Enabled).To(BeTrue())
	Expect(z.Dnssec).To(BeTrue())

	// non-existing zone
	resp = execRequest(http.MethodPut, "/zones/zone2.com.", `{"enabled": true, "dnssec":true, "cname_flattening": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestDeleteZone(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")

	// delete zone
	resp := execRequest(http.MethodDelete, "/zones/zone1.com.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing zone
	resp = execRequest(http.MethodDelete, "/zones/zone1.com.", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestAddLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")

	// add zone
	resp := execRequest(http.MethodPost, "/zones/zone1.com./locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// duplicate
	resp = execRequest(http.MethodPost, "/zones/zone1.com./locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// non-existing zone
	resp = execRequest(http.MethodPost, "/zones/zone2.com./locations", `{"name": "www", "enabled": true}`)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// bad request
	resp = execRequest(http.MethodPost, "/zones/zone1.com./locations", `name: "www", enabled: true`)
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
	resp := execRequest(http.MethodGet, "/zones/zone1.com./locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www1","www2","www3"]`)))

	// another zone
	resp = execRequest(http.MethodGet, "/zones/zone2.com./locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var items zone.ListResponse
	err = json.Unmarshal(body, &items)
	Expect(err).To(BeNil())
	Expect(items).To(ContainElements("www4","www5","www6"))

	// zone with no location
	resp = execRequest(http.MethodGet, "/zones/zone3.com./locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))

	// non-existing zone
	resp = execRequest(http.MethodGet, "/zones/zone4.com./locations", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// limit results
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations?start=1&count=1", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www2"]`)))

	// with q
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations?q=2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`["www2"]`)))

	// empty results
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations?q=asdasd", "")
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
	resp := execRequest(http.MethodGet, "/zones/zone1.com./locations/www", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var l zone.GetLocationResponse
	err = json.Unmarshal(body, &l)
	Expect(err).To(BeNil())

	// non-existing location
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// non-existing zone
	resp = execRequest(http.MethodGet, "/zones/zone2.com./locations/www", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestUpdateLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")

	// update location
	resp := execRequest(http.MethodPut, "/zones/zone1.com./locations/www", `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var l zone.GetLocationResponse
	err = json.Unmarshal(respBody, &l)
	Expect(err).To(BeNil())
	Expect(l.Enabled).To(BeFalse())

	// non-existing zone
	resp = execRequest(http.MethodPut, "/zones/zone2.com./locations/www", `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing location
	resp = execRequest(http.MethodPut, "/zones/zone1.com./locations/www2", `{"enabled": false}`)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestDeleteLocation(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")

	// delete location
	resp := execRequest(http.MethodDelete, "/zones/zone1.com./locations/www", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing location
	resp = execRequest(http.MethodDelete, "/zones/zone1.com./locations/www2", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing zone
	resp = execRequest(http.MethodDelete, "/zones/zone2.com./locations/www", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestAddRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	path := "/zones/zone1.com./locations/www/rrsets"
	body := `{"type": "a", "enabled": true, "value": "{\"ttl\": 300, \"records\": [{\"ip\": \"1.2.3.4\"}]}"}`

	// add record set
	resp := execRequest(http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// duplicate
	resp = execRequest(http.MethodPost, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))

	// non-existing location
	resp = execRequest(http.MethodPost, "/zones/zone1.com./locations/www2/rrsets", body)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing zone
	resp = execRequest(http.MethodPost, "/zones/zone2.com./locations/www/rrsets", body)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// bad request
	body = `ttl: 300, records: {"ip": "1.2.3.4"}`
	resp = execRequest(http.MethodPost, path, body)
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
	resp := execRequest(http.MethodGet, "/zones/zone1.com./locations/www/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var items zone.ListResponse
	err = json.Unmarshal(body, &items)
	Expect(err).To(BeNil())
	Expect(items).To(ContainElements("a","aaaa"))

	// another zone
	resp = execRequest(http.MethodGet, "/zones/zone2.com./locations/www/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	err = json.Unmarshal(body, &items)
	Expect(err).To(BeNil())
	Expect(items).To(ContainElements("aname","cname"))

	// location with no record sets
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www2/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err = ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	Expect(body).To(Equal([]byte(`[]`)))

	// non-existing location
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www3/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing zone
	resp = execRequest(http.MethodGet, "/zones/zone4.com./locations/www/rrsets", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestGetRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)

	// get record set
	resp := execRequest(http.MethodGet, "/zones/zone1.com./locations/www/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var r zone.GetRecordSetResponse
	err = json.Unmarshal(body, &r)
	Expect(err).To(BeNil())
	Expect(r.Value).To(Equal(`{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`))

	// non-existing record set
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www/rrsets/aaaa", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// invalid record type
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www/rrsets/adsd", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing location
	resp = execRequest(http.MethodGet, "/zones/zone1.com./locations/www2/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing zone
	resp = execRequest(http.MethodGet, "/zones/zone2.com./locations/www/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestUpdateRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)
	path := "/zones/zone1.com./locations/www/rrsets/a"
	body := `{"enabled": true, "value": "{\"ttl\": 400, \"records\": [{\"ip\": \"1.2.3.5\"}]}"}`

	// update record set
	resp := execRequest(http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
	resp = execRequest(http.MethodGet, path, "")
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	respBody, err := ioutil.ReadAll(resp.Body)
	Expect(err).To(BeNil())
	var r zone.GetRecordSetResponse
	err = json.Unmarshal(respBody, &r)
	Expect(err).To(BeNil())
	Expect(r.Value).To(Equal(`{"ttl": 400, "records": [{"ip": "1.2.3.5"}]}`))

	// non-existing zone
	path = "/zones/zone2.com./locations/www/rrsets/a"
	resp = execRequest(http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing location
	path = "/zones/zone1.com./locations/www2/rrsets/a"
	resp = execRequest(http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// non-existing record set
	path = "/zones/zone1.com./locations/www/rrsets/aaaa"
	body = `{"enabled": true, "value": "{\"ttl\": 400, \"records\": [{\"ip\": \"::1\"}]}"}`
	resp = execRequest(http.MethodPut, path, body)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestDeleteRecordSet(t *testing.T) {
	initialize(t)
	addZone("user1", "zone1.com.")
	addLocation("zone1.com.", "www")
	addRecordSet("zone1.com.", "www", "a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`)

	// delete record set
	resp := execRequest(http.MethodDelete, "/zones/zone1.com./locations/www/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// delete non-existing record set
	resp = execRequest(http.MethodDelete, "/zones/zone1.com./locations/www/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// delete non-existing zone
	resp = execRequest(http.MethodDelete, "/zones/zone2.com./locations/www/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// delete non-existing location
	resp = execRequest(http.MethodDelete, "/zones/zone1.com./locations/www2/rrsets/a", "")
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
}

func TestSignupAndVerify(t *testing.T) {
	initialize(t)
	err := redis.Del("email_verification")
	Expect(err).To(BeNil())

	// add new user
	body := `{"email": "user1@example.com", "password": "password"}`
	path := "/auth/signup"
	resp := execRequest(http.MethodPost, path, body)
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
	resp = execRequest(http.MethodPost, path, "")
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
	err = db.Clear()
	if err != nil {
		panic(err)
	}
	addUser("user1")
	token, err = login("user1", "user1")
	if err != nil {
		panic(err)
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
	err := db.Clear()
	if err != nil {
		panic(err)
	}
	addUser("user1")
	addUser("user2")
	addUser("user3")
}

func addUser(name string) {
	_, err := db.AddUser(database.User{Email: name, Password: name, Status: database.UserStatusActive})
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

func execRequest(method string, path string, body string) *http.Response {
	url := generateURL(path)
	reqBody := strings.NewReader(body)
	req, err := http.NewRequest(method, url, reqBody)
	Expect(err).To(BeNil())
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	Expect(err).To(BeNil())
	return resp
}
