package database

import (
	"github.com/hawell/z42/internal/types"
	jsoniter "github.com/json-iterator/go"
	. "github.com/onsi/gomega"
	"testing"
)

var (
	connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"
	db            *DataBase
	soa = types.SOA_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 3600},
		Ns:           "n1.example.com.",
		MBox:         "admin.example.com.",
		Refresh:      3600,
		Retry:        3600,
		Expire:       3660,
		MinTtl:       3600,
		Serial:       123456,
	}
	ns = types.NS_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 3600},
		Data:         []types.NS_RR{
			{Host: "ns1.example.com."},
			{Host: "ns2.example.com."},
		},
	}
)

func TestConnect(t *testing.T) {
	RegisterTestingT(t)
	db, err := Connect(connectionStr)
	Expect(err).To(BeNil())
	err = db.Close()
	Expect(err).To(BeNil())
}

func TestUser(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())

	// add
	_, err = db.AddUser(NewUser{Email: "user1", Password: "12345678", Status: UserStatusActive})
	Expect(err).To(BeNil())

	// get
	u, err := db.GetUser("user1")
	Expect(err).To(BeNil())
	Expect(u.Email).To(Equal("user1"))
	Expect(u.Status).To(Equal(UserStatusActive))

	// get non-existing user
	u, err = db.GetUser("user2")
	Expect(err).To(Equal(ErrNotFound))

	// duplicate
	_, err = db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
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

func TestAddZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	// add zone for user
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// zone with no user
	_, err = db.AddZone(NewObjectId(), NewZone{Name: "example0.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(Equal(ErrInvalid))

	// duplicate add
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(Equal(ErrDuplicateEntry))

	// add zone for another user
	user2Id, err := db.AddUser(NewUser{Email: "user2", Password: "user2", Status: UserStatusActive})
	Expect(err).To(BeNil())
	_, err = db.AddZone(user2Id, NewZone{Name: "example4.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// cannot add already added zone for another user
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(Equal(ErrDuplicateEntry))
}

func TestGetZones(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	User1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	User2Id, err := db.AddUser(NewUser{Email: "user2", Password: "user2", Status: UserStatusActive})
	Expect(err).To(BeNil())
	User3Id, err := db.AddUser(NewUser{Email: "user3", Password: "user3", Status: UserStatusActive})
	Expect(err).To(BeNil())

	_, err = db.AddZone(User1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddZone(User1Id, NewZone{Name: "example2.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddZone(User1Id, NewZone{Name: "example3.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddZone(User2Id, NewZone{Name: "example4.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddZone(User2Id, NewZone{Name: "example5.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddZone(User2Id, NewZone{Name: "example6.com.", Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// a user
	zones, err := db.GetZones(User1Id, 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(3))
	Expect(zones).To(ContainElements(List{
		{
			Id: "example1.com.",
		},
		{
			Id: "example2.com.",
		},
		{
			Id: "example3.com.",
		},
	}))

	// another user
	zones, err = db.GetZones(User2Id, 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(3))
	Expect(zones).To(ContainElements(List{
		{
			Id: "example4.com.",
		},
		{
			Id: "example5.com.",
		},
		{
			Id: "example6.com.",
		},
	}))

	// user with no zones
	zones, err = db.GetZones(User3Id, 0, 100, "")
	Expect(err).To(BeNil())
	Expect(zones).To(BeEmpty())

	// non-existing user
	zones, err = db.GetZones(NewObjectId(), 0, 100, "")
	Expect(err).To(BeNil())
	Expect(zones).To(BeEmpty())

	// limit results
	zones, err = db.GetZones(User2Id, 1, 1, "")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(1))
	Expect(zones[0]).To(Equal(ListItem{"example5.com."}))

	// with q
	zones, err = db.GetZones(User1Id, 0, 100, "2")
	Expect(err).To(BeNil())
	Expect(len(zones)).To(Equal(1))
	Expect(zones[0]).To(Equal(ListItem{"example2.com."}))

	// empty results
	zones, err = db.GetZones(User1Id, 0, 100, "no-result")
	Expect(err).To(BeNil())
	Expect(zones).To(BeEmpty())
}

func TestGetZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "example1.com."
	zone1Id, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: true, CNameFlattening: true, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// get zone
	z, err := db.GetZone(user1Id, zoneName)
	Expect(err).To(BeNil())
	Expect(z).To(Equal(Zone{
		Id:              zone1Id,
		Name:            zoneName,
		Enabled:         true,
		Dnssec:          true,
		CNameFlattening: true,
	}))

	// non-existing zone
	_, err = db.GetZone(user1Id, "example2.com.")
	Expect(err).To(Equal(ErrNotFound))
}

func TestUpdateZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "example1.com."
	zone1Id, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: true, CNameFlattening: true, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// update zone
	res, err := db.UpdateZone(user1Id, zoneName, ZoneUpdate{Dnssec: false, CNameFlattening: false, Enabled: false})
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))
	z, err := db.GetZone(user1Id, zoneName)
	Expect(err).To(BeNil())
	Expect(z).To(Equal(Zone{
		Id:              zone1Id,
		Name:            zoneName,
		Enabled:         false,
		Dnssec:          false,
		CNameFlattening: false,
	}))

	// non-existing zone
	res, err = db.UpdateZone(user1Id, "example2.com.", ZoneUpdate{Dnssec: false, CNameFlattening: false, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestDeleteZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "zone1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: true, CNameFlattening: true, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// delete
	res, err := db.DeleteZone(user1Id, zoneName)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))

	// non-existing zone
	res, err = db.DeleteZone(user1Id, "zone2.com.")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestAddLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zoneName := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// add location to zone
	_, err = db.AddLocation(user1Id, zoneName, NewLocation{Name: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zoneName, NewLocation{Name: "a", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zoneName, NewLocation{Name: "b", Enabled: true})
	Expect(err).To(BeNil())

	// add location to invalid zone
	_, err = db.AddLocation(user1Id, "zone2.com.", NewLocation{Name: "www", Enabled: true})
	Expect(err).To(Equal(ErrNotFound))

	// add duplicate location
	_, err = db.AddLocation(user1Id, zoneName, NewLocation{Name: "www2", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zoneName, NewLocation{Name: "www2", Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))
}

func TestGetLocations(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: "a", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: "b", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: "c", Enabled: true})
	Expect(err).To(BeNil())
	zone2Name := "example2.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone2Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zone2Name, NewLocation{Name: "d", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zone2Name, NewLocation{Name: "e", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, zone2Name, NewLocation{Name: "f", Enabled: true})
	Expect(err).To(BeNil())
	zone3Name := "example3.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone3Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	// a zone
	locations, err := db.GetLocations(user1Id, zone1Name, 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(4))
	Expect(locations).To(ContainElements(List{
		{
			Id: "@",
		},
		{
			Id: "a",
		},
		{
			Id: "b",
		},
		{
			Id: "c",
		},
	}))

	// another zone
	locations, err = db.GetLocations(user1Id, zone2Name, 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(4))
	Expect(locations).To(ContainElements(List{
		{
			Id: "@",
		},
		{
			Id: "d",
		},
		{
			Id: "e",
		},
		{
			Id: "f",
		},
	}))

	// zone with no locations
	locations, err = db.GetLocations(user1Id, zone3Name, 0, 100, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(1))

	// non-existing zone
	locations, err = db.GetLocations(user1Id, "zone4.com.", 0, 100, "")
	Expect(err).To(Equal(ErrNotFound))

	// limit results
	locations, err = db.GetLocations(user1Id, zone1Name, 1, 1, "")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(1))
	Expect(locations).To(ContainElement(ListItem{
		Id: "a",
	}))

	// with q
	locations, err = db.GetLocations(user1Id, zone1Name, 0, 100, "b")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(1))
	Expect(locations).To(ContainElement(ListItem{
		Id: "b",
	}))

	// empty result
	locations, err = db.GetLocations(user1Id, zone1Name, 0, 100, "no-result")
	Expect(err).To(BeNil())
	Expect(len(locations)).To(Equal(0))
}

func TestGetLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "a"
	l1Id, err := db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())

	// get location
	l, err := db.GetLocation(user1Id, zone1Name, l1)
	Expect(err).To(BeNil())
	Expect(l).To(Equal(Location{
		Id:      l1Id,
		Name:    "a",
		Enabled: true,
	}))

	// non-existing location
	l, err = db.GetLocation(user1Id, zone1Name, "bad")
	Expect(err).To(Equal(ErrNotFound))
}

func TestUpdateLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "a"
	l1Id, err := db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())

	// update
	_, err = db.UpdateLocation(user1Id, zone1Name, l1, LocationUpdate{Enabled: false})
	Expect(err).To(BeNil())
	l, err := db.GetLocation(user1Id, zone1Name, l1)
	Expect(err).To(BeNil())
	Expect(l).To(Equal(Location{
		Id:      l1Id,
		Name:    l1,
		Enabled: false,
	}))
	Expect(l.Enabled).To(BeFalse())

	// non-existing location
	res, err := db.UpdateLocation(user1Id, zone1Name, "bad", LocationUpdate{Enabled: true})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestDeleteLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "a"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())

	// delete
	_, err = db.DeleteLocation(user1Id, zone1Name, l1)
	Expect(err).To(BeNil())
	_, err = db.GetLocation(user1Id, zone1Name, l1)
	Expect(err).To(Equal(ErrNotFound))

	// non-existing zone
	res, err := db.DeleteLocation(user1Id, "zone2.com.", l1)
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing location
	res, err = db.DeleteLocation(user1Id, zone1Name, "b")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestAddRecordSet(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())

	// add
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// duplicate
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// recordset with invalid type
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: "invalid-type", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).NotTo(BeNil())

	// recordset with invalid location
	_, err = db.AddRecordSet(user1Id, zone1Name, "www2", NewRecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrNotFound))

	// recordset with invalid zone
	_, err = db.AddRecordSet(user1Id, "zone2.com.", l1, NewRecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(Equal(ErrNotFound))

	// add recordset to location
	rrKeys := []ListItem{
		{Id: "a"},
		{Id: "aaaa"},
		{Id: "aname"},
		{Id: "caa"},
		{Id: "cname"},
		{Id: "ds"},
		{Id: "mx"},
		{Id: "ns"},
		{Id: "ptr"},
		{Id: "srv"},
		{Id: "tlsa"},
		{Id: "txt"},
	}
	rrValues := []string{
		`{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`,
		`{"ttl": 300, "records": [{"ip": "::1"}]}`,
		`{"location": "aname.example.com."}`,
		`{"ttl": 300, "records": [{"tag": "issue", "flag": 0, "value": "godaddy.com;"}]}`,
		`{"ttl": 300, "host": "x.example.com."}`,
		`{"ttl": 300, "records": [{"digest": "B6DCD485719ADCA18E5F3D48A2331627FDD3636B", "key_tag": 57855, "algorithm": 5, "digest_type": 1}]}`,
		`{"ttl": 300, "records": [{"host": "mx1.example.com.", "preference": 10}, {"host": "mx2.example.com.", "preference": 10}]}`,
		`{"ttl": 300, "records": [{"host": "ns1.example.com."}, {"host": "ns2.example.com."}]}`,
		`{"ttl": 300, "domain": "localhost"}`,
		`{"ttl": 300, "records": [{"port": 555, "target": "sip.example.com.", "weight": 100, "priority": 10}]}`,
		`{"ttl": 300, "records": [{"usage": 0, "selector": 0, "certificate": "d2abde240d7cd3ee6b4b28c54df034b97983a1d16e8a410e4561cb106618e971", "matching_type": 1}]}`,
		`{"ttl": 300, "records": [{"text": "foo"}, {"text": "bar"}]}`,
	}
	l2 := "a"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: "a", Enabled: true})
	Expect(err).To(BeNil())
	for i := range rrKeys {
		_, err := db.AddRecordSet(user1Id, zone1Name, l2, NewRecordSet{Type: rrKeys[i].Id, Value: rrValues[i], Enabled: true})
		Expect(err).To(BeNil())
	}
	sets, err := db.GetRecordSets(user1Id, zone1Name, l2)
	Expect(err).To(BeNil())
	Expect(sets).To(ContainElements(rrKeys))
}

func TestGetRecordSets(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: "a", Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: "aaaa", Value: `{"ttl": 300, "records":[{"ip":"::1"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// get
	r, err := db.GetRecordSets(user1Id, zone1Name, l1)
	Expect(err).To(BeNil())
	Expect(len(r)).To(Equal(2))
	Expect(r).To(ContainElements(List{
		{
			Id: "a",
		},
		{
			Id: "aaaa",
		},
	}))

	// empty location
	l2 := "www2"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l2, Enabled: true})
	Expect(err).To(BeNil())
	r, err = db.GetRecordSets(user1Id, zone1Name, l2)
	Expect(err).To(BeNil())
	Expect(len(r)).To(Equal(0))

	// invalid zone
	r, err = db.GetRecordSets(user1Id, "zone2.com.", l1)
	Expect(err).To(Equal(ErrNotFound))

	// invalid location
	r, err = db.GetRecordSets(user1Id, zone1Name, "bad")
	Expect(err).To(Equal(ErrNotFound))
}

func TestGetRecordSet(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	r1Id, err := db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: r1Type, Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// get
	r, err := db.GetRecordSet(user1Id, zone1Name, l1, r1Type)
	Expect(err).To(BeNil())
	Expect(r).To(Equal(RecordSet{
		Id:      r1Id,
		Type:    r1Type,
		Value:   `{"ttl": 300, "records": [{"ip": "1.2.3.4"}]}`,
		Enabled: true,
	}))

	// non-existing zone
	_, err = db.GetRecordSet(user1Id, "zone2.com.", l1, r1Type)
	Expect(err).To(Equal(ErrNotFound))

	// non-existing location
	_, err = db.GetRecordSet(user1Id, zone1Name, "xxx", r1Type)
	Expect(err).To(Equal(ErrNotFound))

	// non-existing record
	_, err = db.GetRecordSet(user1Id, zone1Name, l1, "aaaa")
	Expect(err).To(Equal(ErrNotFound))
}

func TestUpdateRecordSet(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	r1Id, err := db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: r1Type, Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// update
	_, err = db.UpdateRecordSet(user1Id, zone1Name, l1, r1Type, RecordSetUpdate{Value: `{"ttl": 400, "records":[{"ip":"2.2.3.4"}]}`, Enabled: false})
	Expect(err).To(BeNil())
	r, err := db.GetRecordSet(user1Id, zone1Name, l1, r1Type)
	Expect(err).To(BeNil())
	Expect(r).To(Equal(RecordSet{
		Id:      r1Id,
		Type:    r1Type,
		Value:   `{"ttl": 400, "records": [{"ip": "2.2.3.4"}]}`,
		Enabled: false,
	}))

	// non-existing zone
	res, err := db.UpdateRecordSet(user1Id, "zone2.com.", l1, r1Type, RecordSetUpdate{Value: `{"ttl": 400, "records":[{"ip":"::1"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing location
	res, err = db.UpdateRecordSet(user1Id, zone1Name, "xxx", r1Type, RecordSetUpdate{Value: `{"ttl": 400, "records":[{"ip":"::1"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing record
	res, err = db.UpdateRecordSet(user1Id, zone1Name, l1, "aaaa", RecordSetUpdate{Value: `{"ttl": 400, "records":[{"ip":"::1"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// invalid type
	_, err = db.UpdateRecordSet(user1Id, zone1Name, l1, "invalid-type", RecordSetUpdate{Value: `{"ttl": 400, "records":[{"ip":"2.2.3.4"}]}`, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
}

func TestDeleteRecordSet(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: r1Type, Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	// delete
	_, err = db.DeleteRecordSet(user1Id, zone1Name, l1, r1Type)
	Expect(err).To(BeNil())

	// non-existing zone
	res, err := db.DeleteRecordSet(user1Id, "zone2.com.", l1, r1Type)
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing location
	res, err = db.DeleteRecordSet(user1Id, zone1Name, "xxx", r1Type)
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))

	// non-existing record
	res, err = db.DeleteRecordSet(user1Id, zone1Name, l1, "aaaa")
	Expect(err).To(Equal(ErrNotFound))
	Expect(res).To(Equal(int64(0)))
}

func TestCascadeDelete(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())

	user1Id, err := db.AddUser(NewUser{Email: "admin", Password: "admin", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com"
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, zone1Name, NewLocation{Name: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	_, err = db.AddRecordSet(user1Id, zone1Name, l1, NewRecordSet{Type: r1Type, Value: `{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}`, Enabled: true})
	Expect(err).To(BeNil())

	res, err := db.DeleteZone(user1Id, zone1Name)
	Expect(err).To(BeNil())
	Expect(res).To(Equal(int64(1)))

	locations, err := db.GetLocations(user1Id, zone1Name, 0, 100, "")
	Expect(err).To(Equal(ErrNotFound))
	Expect(len(locations)).To(Equal(0))
	recordSets, err := db.GetRecordSets(user1Id, zone1Name, l1)
	Expect(err).To(Equal(ErrNotFound))
	Expect(len(recordSets)).To(Equal(0))
}

func TestAutoInsertedItemsAfterAddZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, err := db.AddUser(NewUser{Email: "user1", Password: "user1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	// add zone for user
	zoneName := "example.com."
	zoneId, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: false, CNameFlattening: false, Enabled: true}, soa, ns)
	Expect(err).To(BeNil())

	z, err := db.GetZone(user1Id, zoneName)
	Expect(err).To(BeNil())
	Expect(z).To(Equal(Zone{
		Id:              zoneId,
		Name:            zoneName,
		Enabled:         true,
		Dnssec:          false,
		CNameFlattening: false,
	}))

	_, err = db.GetLocation(user1Id, zoneName, "@")
	Expect(err).To(BeNil())

	soaRecord, err := db.GetRecordSet(user1Id, zoneName, "@", "soa")
	Expect(err).To(BeNil())
	var storedSOA types.SOA_RRSet
	err = jsoniter.Unmarshal([]byte(soaRecord.Value), &storedSOA)
	Expect(err).To(BeNil())
	Expect(storedSOA).To(Equal(soa))

	nsRecord, err := db.GetRecordSet(user1Id, zoneName, "@", "ns")
	Expect(err).To(BeNil())
	var storedNS types.NS_RRSet
	err = jsoniter.Unmarshal([]byte(nsRecord.Value), &storedNS)
	Expect(err).To(BeNil())
	Expect(storedNS).To(Equal(ns))
}

func TestMain(m *testing.M) {
	db, _ = Connect(connectionStr)
	m.Run()
	_ = db.Close()
}
