package db

import (
	"database/sql"
	. "github.com/onsi/gomega"
	"testing"
)

var connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"

func TestConnect(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	db.Close()
}

func TestUser(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	// add
	_, err = db.AddUser("user1")
	Expect(err).To(BeNil())

	// duplicate
	_, err = db.AddUser("user1")
	Expect(err).NotTo(BeNil())

	// delete
	res, err := db.DeleteUser("user1")
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))
	_, err = db.GetUser("user1")
	Expect(err).NotTo(BeNil())

	// delete non-existing user
	res, err = db.DeleteUser("user1")
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(0)))
}

func TestZone(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	// zone with no user
	_, err = db.AddZone("user0", "example0.com.", true)
	Expect(err).NotTo(BeNil())

	// add zone for user
	_, err = db.AddUser("user1")
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", "example1.com.", true)
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", "example2.com.", true)
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", "example3.com.", true)
	Expect(err).To(BeNil())
	zones, err := db.GetZones("user1", 0, 100)
	Expect(len(zones)).To(Equal(3))
	Expect(zones[0].Enabled).To(BeTrue())
	Expect(zones[0].Name).To(Equal("example1.com."))
	Expect(zones[1].Enabled).To(BeTrue())
	Expect(zones[1].Name).To(Equal("example2.com."))
	Expect(zones[2].Enabled).To(BeTrue())
	Expect(zones[2].Name).To(Equal("example3.com."))

	// duplicate add
	_, err = db.AddZone("user1", "example1.com.", true)
	Expect(err).NotTo(BeNil())

	// add zone for another user
	_, err = db.AddUser("user2")
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", "example4.com.", true)
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", "example5.com.", true)
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", "example6.com.", true)
	Expect(err).To(BeNil())
	zones, err = db.GetZones("user2", 0, 100)
	Expect(len(zones)).To(Equal(3))
	Expect(zones[0].Enabled).To(BeTrue())
	Expect(zones[0].Name).To(Equal("example4.com."))
	Expect(zones[1].Enabled).To(BeTrue())
	Expect(zones[1].Name).To(Equal("example5.com."))
	Expect(zones[2].Enabled).To(BeTrue())
	Expect(zones[2].Name).To(Equal("example6.com."))

	// cannot add already added zone for another user
	_, err = db.AddZone("user2", "example1.com.", true)
	Expect(err).NotTo(BeNil())
}

func TestLocation(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	_, err = db.AddUser("user1")
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", "example.com.", true)

	// location with invalid zone
	_, err = db.AddLocation("example2.com.", "www", true)
	Expect(err).NotTo(BeNil())

	// add location to zone
	_, err = db.AddLocation("example.com.", "www", true)
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", "a", true)
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", "b", true)
	Expect(err).To(BeNil())

	// add duplicate location
	_, err = db.AddLocation("example.com.", "www2", true)
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", "www2", true)
	Expect(err).NotTo(BeNil())

	locations, err := db.GetLocations("example.com.", 0, 100)
	Expect(len(locations)).To(Equal(4))
	Expect(locations[0].Name).To(Equal("a"))
	Expect(locations[0].Enabled).To(BeTrue())
	Expect(locations[1].Name).To(Equal("b"))
	Expect(locations[1].Enabled).To(BeTrue())
	Expect(locations[2].Name).To(Equal("www"))
	Expect(locations[2].Enabled).To(BeTrue())
	Expect(locations[3].Name).To(Equal("www2"))
	Expect(locations[3].Enabled).To(BeTrue())
}

func TestRecordSet(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	_, err = db.AddUser("user1")
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", "example.com.", true)
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", "www", true)

	// recordset with invalid location
	_, err = db.AddRecordSet("example.com.", "www2", "a", `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, true)
	Expect(err).NotTo(BeNil())

	// recordset with invalid zone
	_, err = db.AddRecordSet("example1.com.", "www", "a", `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, true)
	Expect(err).NotTo(BeNil())

	// recordset with invalid type
	_, err = db.AddRecordSet("example1.com.", "www", "abcd", `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, true)
	Expect(err).NotTo(BeNil())

	// add recordset to location
	rrs := [][]string {
		{"a", `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`},
		{"aaaa", `{"ttl": 300, "records": [{"ip": "::1"}]}`},
		{"aname", `{"location": "aname.example.com."}`},
		{"caa", `{"ttl": 300, "records": [{"tag": "issue", "flag": 0, "value": "godaddy.com;"}]}`},
		{"cname", `{"ttl": 300, "host": "x.example.com."}`},
		{"ds", `{"ttl": 300, "records": [{"digest": "B6DCD485719ADCA18E5F3D48A2331627FDD3636B", "key_tag": 57855, "algorithm": 5, "digest_type": 1}]}`},
		{"mx", `{"ttl": 300, "records": [{"host": "mx1.example.com.", "preference": 10}, {"host": "mx2.example.com.", "preference": 10}]}`},
		{"ns", `{"ttl": 300, "records": [{"host": "ns1.example.com."}, {"host": "ns2.example.com."}]}`},
		{"ptr", `{"ttl": 300, "domain": "localhost"}`},
		{"srv", `{"ttl": 300, "records": [{"port": 555, "target": "sip.example.com.", "weight": 100, "priority": 10}]}`},
		{"tlsa", `{"ttl": 300, "records": [{"usage": 0, "selector": 0, "certificate": "d2abde240d7cd3ee6b4b28c54df034b97983a1d16e8a410e4561cb106618e971", "matching_type": 1}]}`},
		{"txt", `{"ttl": 300, "records": [{"text": "foo"}, {"text": "bar"}]}`},
	}
	for _, rr := range rrs {
		_, err = db.AddRecordSet("example.com.", "www", rr[0], rr[1], true)
		Expect(err).To(BeNil())
	}
	sets, err := db.GetRecordSets("example.com.", "www")
	Expect(err).To(BeNil())
	Expect(len(sets)).To(Equal(12))
	for i, set := range sets {
		Expect(set.Enabled).To(BeTrue())
		Expect(set.Type).To(Equal(rrs[i][0]))
		Expect(set.Value).To(Equal(rrs[i][1]))
	}
}

func TestCascadeDelete(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	_, err = db.AddUser("admin")
	Expect(err).To(BeNil())
	_, err = db.AddZone("admin", "example.com.", true)
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", "www", true)
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", "a", `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, true)
	Expect(err).To(BeNil())

	res, err := db.DeleteUser("admin")
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))

	zones, err := db.GetZones("admin", 0, 100)
	Expect(err).To(Equal(sql.ErrNoRows))
	Expect(len(zones)).To(Equal(0))
	locations, err := db.GetLocations("example.com.", 0, 100)
	Expect(err).To(Equal(sql.ErrNoRows))
	Expect(len(locations)).To(Equal(0))
	recordSets, err := db.GetRecordSets("example.com.", "www")
	Expect(err).To(Equal(sql.ErrNoRows))
	Expect(len(recordSets)).To(Equal(0))
}
