package database

import (
	. "github.com/onsi/gomega"
	"testing"
)

var connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"

func TestConnect(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Close()
	Expect(err).To(BeNil())
}

func TestUser(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	// add
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())

	// get
	u, err := db.GetUser("user1")
	Expect(err).To(BeNil())
	Expect(u.Name).To(Equal("user1"))

	// get non-existing user
	u, err = db.GetUser("user2")
	Expect(err).To(Equal(ErrNotFound))

	// duplicate
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// delete
	res, err := db.DeleteUser("user1")
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))
	_, err = db.GetUser("user1")
	Expect(err).NotTo(BeNil())

	// delete non-existing user
	res, err = db.DeleteUser("user1")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func  TestAddZone(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())

	// add zone for user
	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())

	// zone with no user
	_, err = db.AddZone("user0", Zone{Name: "example0.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(Equal(ErrInvalid))

	// duplicate add
	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// add zone for another user
	_, err = db.AddUser(User{Name: "user2"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", Zone{Name: "example4.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())

	// cannot add already added zone for another user
	_, err = db.AddZone("user2", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(Equal(ErrInvalid))
}

func TestGetZones(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user2"})
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user3"})
	Expect(err).To(BeNil())

	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example2.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example3.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", Zone{Name: "example4.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", Zone{Name: "example5.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user2", Zone{Name: "example6.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())

	// a user
	zones, err := db.GetZones("user1", 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(3))
	Expect(zones[0]).To(Equal("example1.com."))
	Expect(zones[1]).To(Equal("example2.com."))
	Expect(zones[2]).To(Equal("example3.com."))

	// another user
	zones, err = db.GetZones("user2", 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(3))
	Expect(zones[0]).To(Equal("example4.com."))
	Expect(zones[1]).To(Equal("example5.com."))
	Expect(zones[2]).To(Equal("example6.com."))

	// user with no zones
	zones, err = db.GetZones("user3", 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(0))

	// non-existing user
	zones, err = db.GetZones("user4", 0, 100, "")
	Expect(err).To(Equal(ErrInvalid))

	// limit results
	zones, err = db.GetZones("user2", 1, 1, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(1))
	Expect(zones[0]).To(Equal("example5.com."))

	// with q
	zones, err = db.GetZones("user1", 0, 100, "2")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(1))
	Expect(zones[0]).To(Equal("example2.com."))

	// empty results
	zones, err = db.GetZones("user1", 0, 100, "fkfkfkf")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(0))
}

func TestGetZone(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())

	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: true, CNameFlattening: true, Enabled: true})
	Expect(err).To(BeNil())

	// get zone
	z, err := db.GetZone("example1.com.")
	Expect(err).To(BeNil())
	Expect(z.Name).To(Equal("example1.com."))
	Expect(z.CNameFlattening).To(BeTrue())
	Expect(z.Dnssec).To(BeTrue())
	Expect(z.Enabled).To(BeTrue())

	// non-existing zone
	_, err = db.GetZone("example2.com.")
	Expect(err).To(Equal(ErrNotFound))
}

func TestUpdateZone(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())

	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: true, CNameFlattening: true, Enabled: true})
	Expect(err).To(BeNil())

	// update zone
	res, err := db.UpdateZone(Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: false})
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))
	z, err := db.GetZone("example1.com.")
	Expect(err).To(BeNil())
	Expect(z.Name).To(Equal("example1.com."))
	Expect(z.CNameFlattening).To(BeFalse())
	Expect(z.Dnssec).To(BeFalse())
	Expect(z.Enabled).To(BeFalse())

	// non-existing zone
	res, err = db.UpdateZone(Zone{Name: "example2.com.", Dnssec: false, CNameFlattening: false, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestDeleteZone(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())

	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: true, CNameFlattening: true, Enabled: true})
	Expect(err).To(BeNil())

	// delete
	res, err := db.DeleteZone("example1.com.")
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))

	// non-existing zone
	res, err = db.DeleteZone("example1.com.")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestAddLocation(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())

	// add location to zone
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "b", Enabled: true})
	Expect(err).To(BeNil())

	// add location to invalid zone
	_, err = db.AddLocation("example2.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(Equal(ErrInvalid))

	// add duplicate location
	_, err = db.AddLocation("example.com.", Location{Name: "www2", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www2", Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))
}

func TestGetLocations(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example1.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example1.com.", Location{Name: "b", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example1.com.", Location{Name: "c", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example2.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example2.com.", Location{Name: "d", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example2.com.", Location{Name: "e", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example2.com.", Location{Name: "f", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example3.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())

	// a zone
	locations, err := db.GetLocations("example1.com.", 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(3))
	Expect(locations[0]).To(Equal("a"))
	Expect(locations[1]).To(Equal("b"))
	Expect(locations[2]).To(Equal("c"))

	// another zone
	locations, err = db.GetLocations("example2.com.", 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(3))
	Expect(locations[0]).To(Equal("d"))
	Expect(locations[1]).To(Equal("e"))
	Expect(locations[2]).To(Equal("f"))

	// zone with no locations
	locations, err = db.GetLocations("example3.com.", 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(0))

	// non-existing zone
	locations, err = db.GetLocations("example4.com.", 0, 100, "")
	Expect(err).To(Equal(ErrInvalid))

	// limit results
	locations, err = db.GetLocations("example1.com.", 1, 1, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(1))
	Expect(locations[0]).To(Equal("b"))

	// with q
	locations, err = db.GetLocations("example1.com.", 0, 100, "b")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(1))
	Expect(locations[0]).To(Equal("b"))

	// empty result
	locations, err = db.GetLocations("example1.com.", 0, 100, "bdsdsds")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(0))
}

func TestGetLocation(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example1.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(BeNil())

	// get location
	l, err := db.GetLocation("example1.com.", "a")
	Expect(err).To(BeNil())
	Expect(l.Enabled).To(BeTrue())
	Expect(l.Name).To(Equal("a"))

	// non-existing location
	l, err = db.GetLocation("example1.com.", "b")
	Expect(err).To(Equal(ErrNotFound))

	// non-existing zone
	l, err = db.GetLocation("example2.com.", "a")
	Expect(err).To(Equal(ErrInvalid))
}

func TestUpdateLocation(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example1.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(BeNil())

	// update
	_, err = db.UpdateLocation("example1.com.", Location{Name: "a", Enabled: false})
	Expect(err).To(BeNil())
	l, err := db.GetLocation("example1.com.", "a")
	Expect(err).To(BeNil())
	Expect(l.Name).To(Equal("a"))
	Expect(l.Enabled).To(BeFalse())

	// non-existing location
	res, err := db.UpdateLocation("example1.com.", Location{Name: "b", Enabled: true})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing zone
	_, err = db.UpdateLocation("example2.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(Equal(ErrInvalid))
}

func TestDeleteLocation(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example1.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(BeNil())

	// delete
	_, err = db.DeleteLocation("example1.com.", "a")
	Expect(err).To(BeNil())
	_, err = db.GetLocation("example1.com.", "a")
	Expect(err).To(Equal(ErrNotFound))

	// non-existing location
	res, err := db.DeleteLocation("example1.com.", "b")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing zone
	_, err = db.DeleteLocation("example2.com.", "a")
	Expect(err).To(Equal(ErrInvalid))
}

func TestAddRecordSet(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{ Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())

	// add
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// duplicate
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// recordset with invalid type
	_, err = db.AddRecordSet("example1.com.", "www", RecordSet{Type: "abcd", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrInvalid))

	// recordset with invalid location
	_, err = db.AddRecordSet("example.com.", "www2", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrInvalid))

	// recordset with invalid zone
	_, err = db.AddRecordSet("example1.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrInvalid))

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
	_, err = db.AddLocation("example.com.", Location{Name: "a", Enabled: true})
	Expect(err).To(BeNil())
	for _, rr := range rrs {
		_, err = db.AddRecordSet("example.com.", "a", RecordSet{Type: rr[0], Value: rr[1], Enabled: true})
		Expect(err).To(BeNil())
	}
	sets, err := db.GetRecordSets("example.com.", "a")
	Expect(err).To(BeNil())
	Expect(len(sets)).To(Equal(12))
	for i, set := range sets {
		Expect(set).To(Equal(rrs[i][0]))
	}
}

func TestGetRecordSets(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{ Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "aaaa", Value: `{"ttl": 300, "records":[{"ip":"::1"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// get
	r, err := db.GetRecordSets("example.com.", "www")
	Expect(err).To(BeNil())
	Expect(len(r)).To(Equal(2))
	Expect(r[0]).To(Equal("a"))
	Expect(r[1]).To(Equal("aaaa"))

	// empty location
	_, err = db.AddLocation("example.com.", Location{Name: "www2", Enabled: true})
	Expect(err).To(BeNil())
	r, err = db.GetRecordSets("example.com.", "www2")
	Expect(err).To(BeNil())
	Expect(len(r)).To(Equal(0))

	// invalid location
	r, err = db.GetRecordSets("example.com.", "www3")
	Expect(err).To(Equal(ErrInvalid))
}

func TestGetRecordSet(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{ Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// get
	r, err := db.GetRecordSet("example.com.", "www", "a")
	Expect(err).To(BeNil())
	Expect(r.Type).To(Equal("a"))
	Expect(r.Enabled).To(BeTrue())
	Expect(r.Value).To(Equal(`{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`))

	// non-existing type
	_, err = db.GetRecordSet("example.com.", "www", "aaaa")
	Expect(err).To(Equal(ErrNotFound))

	// invalid type
	_, err = db.GetRecordSet("example.com.", "www", "abcd")
	Expect(err).To(Equal(ErrInvalid))

	// invalid location
	_, err = db.GetRecordSet("example.com.", "www2", "a")
	Expect(err).To(Equal(ErrInvalid))

	// invalid zone
	_, err = db.GetRecordSet("example2.com.", "www", "a")
	Expect(err).To(Equal(ErrInvalid))
}

func TestUpdateRecordSet(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{ Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// update
	_, err = db.UpdateRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 400, "records":[{"ip":"2.2.3.4"}]}`, Enabled: false})
	Expect(err).To(BeNil())
	r, err := db.GetRecordSet("example.com.", "www", "a")
	Expect(err).To(BeNil())
	Expect(r.Type).To(Equal("a"))
	Expect(r.Enabled).To(BeFalse())
	Expect(r.Value).To(Equal(`{"ttl": 400, "records": [{"ip": "2.2.3.4"}]}`))

	// non-existing type
	res, err := db.UpdateRecordSet("example.com.", "www", RecordSet{Type: "aaaa", Value: `{"ttl": 400, "records":[{"ip":"::1"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// invalid type
	_, err = db.UpdateRecordSet("example.com.", "www", RecordSet{Type: "abcd", Value: `{"ttl": 400, "records":[{"ip":"2.2.3.4"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrInvalid))

	// invalid location
	_, err = db.UpdateRecordSet("example.com.", "www2", RecordSet{Type: "a", Value: `{"ttl": 400, "records":[{"ip":"2.2.3.4"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrInvalid))

	// invalid zone
	_, err = db.UpdateRecordSet("example2.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 400, "records":[{"ip":"2.2.3.4"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrInvalid))
}

func TestDeleteRecordSet(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())
	_, err = db.AddUser(User{Name: "user1"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("user1", Zone{ Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// delete
	_, err = db.DeleteRecordSet("example.com.", "www", "a")
	Expect(err).To(BeNil())

	// non-existing
	res, err := db.DeleteRecordSet("example.com.", "www", "a")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// invalid location
	_, err = db.DeleteRecordSet("example.com.", "www2", "a")
	Expect(err).To(Equal(ErrInvalid))

	// invalid zone
	_, err = db.DeleteRecordSet("example2.com.", "www", "a")
	Expect(err).To(Equal(ErrInvalid))
}

func TestCascadeDelete(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Clear()
	Expect(err).To(BeNil())

	_, err = db.AddUser(User{ Name: "admin"})
	Expect(err).To(BeNil())
	_, err = db.AddZone("admin", Zone{Name: "example.com.", Dnssec: false, CNameFlattening: false, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation("example.com.", Location{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet("example.com.", "www", RecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	res, err := db.DeleteUser("admin")
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))

	zones, err := db.GetZones("admin", 0, 100, "")
	Expect(err).To(Equal(ErrInvalid))
	Expect(len(zones)).To(Equal(0))
	locations, err := db.GetLocations("example.com.", 0, 100, "")
	Expect(err).To(Equal(ErrInvalid))
	Expect(len(locations)).To(Equal(0))
	recordSets, err := db.GetRecordSets("example.com.", "www")
	Expect(err).To(Equal(ErrInvalid))
	Expect(len(recordSets)).To(Equal(0))
}
