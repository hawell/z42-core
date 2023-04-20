package database

import (
	"z42-core/internal/types"
	"github.com/miekg/dns"
	. "github.com/onsi/gomega"
	"net"
	"testing"
)

var (
	connectionStr = "root:root@tcp(127.0.0.1:3306)/z42"
	db            *DataBase
	soa           = types.SOA_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 3600},
		Ns:           "ns1.example.com.",
		MBox:         "admin.example.com.",
		Refresh:      44,
		Retry:        55,
		Expire:       66,
		MinTtl:       100,
		Serial:       123456,
	}
	ns = types.NS_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 3600},
		Data: []types.NS_RR{
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
	_, _, err = db.AddUser(NewUser{Email: "dbUser1", Password: "12345678", Status: UserStatusActive})
	Expect(err).To(BeNil())

	// get
	u, err := db.GetUser("dbUser1")
	Expect(err).To(BeNil())
	Expect(u.Email).To(Equal("dbUser1"))
	Expect(u.Status).To(Equal(UserStatusActive))

	// get non-existing user
	u, err = db.GetUser("dbUser2")
	Expect(err).To(Equal(ErrNotFound))

	// duplicate
	_, _, err = db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// delete
	err = db.DeleteUser("dbUser1")
	Expect(err).To(BeNil())
	_, err = db.GetUser("dbUser1")
	Expect(err).NotTo(BeNil())

	// delete non-existing user
	err = db.DeleteUser("dbUser1")
	Expect(err).To(Equal(ErrNotFound))
}

func TestAddZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	// add zone for user
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// zone with no user
	_, err = db.AddZone(NewObjectId(), NewZone{Name: "example0.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(Equal(ErrInvalid))

	// duplicate add
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// add zone for another user
	user2Id, _, err := db.AddUser(NewUser{Email: "dbUser2", Password: "dbUser2", Status: UserStatusActive})
	Expect(err).To(BeNil())
	_, err = db.AddZone(user2Id, NewZone{Name: "example4.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// cannot add already added zone for another user
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(Equal(ErrDuplicateEntry))
}

func TestGetZones(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	User1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	User2Id, _, err := db.AddUser(NewUser{Email: "dbUser2", Password: "dbUser2", Status: UserStatusActive})
	Expect(err).To(BeNil())
	User3Id, _, err := db.AddUser(NewUser{Email: "dbUser3", Password: "dbUser3", Status: UserStatusActive})
	Expect(err).To(BeNil())

	_, err = db.AddZone(User1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddZone(User1Id, NewZone{Name: "example2.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddZone(User1Id, NewZone{Name: "example3.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddZone(User2Id, NewZone{Name: "example4.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddZone(User2Id, NewZone{Name: "example5.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddZone(User2Id, NewZone{Name: "example6.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// a user
	zones, err := db.GetZones(User1Id, 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(zones).To(Equal(List{
		Total: 3,
		Items: []ListItem{
			{
				Id:      "example1.com.",
				Enabled: true,
			},
			{
				Id:      "example2.com.",
				Enabled: true,
			},
			{
				Id:      "example3.com.",
				Enabled: true,
			}},
	}))

	// another user
	zones, err = db.GetZones(User2Id, 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(zones).To(Equal(List{
		Total: 3,
		Items: []ListItem{
			{
				Id:      "example4.com.",
				Enabled: true,
			},
			{
				Id:      "example5.com.",
				Enabled: true,
			},
			{
				Id:      "example6.com.",
				Enabled: true,
			}},
	}))

	// user with no zones
	zones, err = db.GetZones(User3Id, 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(zones).To(Equal(List{Items: []ListItem{}}))

	// non-existing user
	zones, err = db.GetZones(NewObjectId(), 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(zones).To(Equal(List{Items: []ListItem{}}))

	// limit results
	zones, err = db.GetZones(User2Id, 1, 1, "", true)
	Expect(err).To(BeNil())
	Expect(zones.Total).To(Equal(3))
	Expect(len(zones.Items)).To(Equal(1))
	Expect(zones.Items[0]).To(Equal(ListItem{Id: "example5.com.", Enabled: true}))

	// with q
	zones, err = db.GetZones(User1Id, 0, 100, "2", true)
	Expect(err).To(BeNil())
	Expect(zones.Total).To(Equal(1))
	Expect(zones.Items[0]).To(Equal(ListItem{Id: "example2.com.", Enabled: true}))

	// empty results
	zones, err = db.GetZones(User1Id, 0, 100, "no-result", true)
	Expect(err).To(BeNil())
	Expect(zones).To(Equal(List{Items: []ListItem{}}))
}

func TestGetZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "example1.com."
	zone1Id, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: true, CNameFlattening: true, Enabled: true, SOA: soa, NS: ns})
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
		SOA:             soa,
	}))

	// non-existing zone
	_, err = db.GetZone(user1Id, "example2.com.")
	Expect(err).To(Equal(ErrNotFound))
}

func TestUpdateZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "example1.com."
	zone1Id, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: true, CNameFlattening: true, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// update zone
	err = db.UpdateZone(user1Id,
		ZoneUpdate{
			Name:            zoneName,
			Dnssec:          false,
			CNameFlattening: false,
			Enabled:         false,
			SOA:             soa,
		},
	)
	Expect(err).To(BeNil())
	z, err := db.GetZone(user1Id, zoneName)
	Expect(err).To(BeNil())
	Expect(z).To(Equal(Zone{
		Id:              zone1Id,
		Name:            zoneName,
		Enabled:         false,
		Dnssec:          false,
		CNameFlattening: false,
		SOA:             soa,
	}))

	// non-existing zone
	err = db.UpdateZone(user1Id, ZoneUpdate{Name: "example2.com.", Dnssec: false, CNameFlattening: false, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
}

func TestDeleteZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "zone1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: true, CNameFlattening: true, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// delete
	err = db.DeleteZone(user1Id, ZoneDelete{Name: zoneName})
	Expect(err).To(BeNil())

	// non-existing zone
	err = db.DeleteZone(user1Id, ZoneDelete{Name: "zone2.com."})
	Expect(err).To(Equal(ErrNotFound))
}

func TestAddLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zoneName := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// add location to zone
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zoneName, Location: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zoneName, Location: "a", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zoneName, Location: "b", Enabled: true})
	Expect(err).To(BeNil())

	// add location to invalid zone
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: "zone2.com.", Location: "www", Enabled: true})
	Expect(err).To(Equal(ErrNotFound))

	// add duplicate location
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zoneName, Location: "www2", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zoneName, Location: "www2", Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))
}

func TestGetLocations(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: "a", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: "b", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: "c", Enabled: true})
	Expect(err).To(BeNil())
	zone2Name := "example2.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone2Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone2Name, Location: "d", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone2Name, Location: "e", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone2Name, Location: "f", Enabled: true})
	Expect(err).To(BeNil())
	zone3Name := "example3.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone3Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	// a zone
	locations, err := db.GetLocations(user1Id, zone1Name, 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(locations.Total).To(Equal(4))
	Expect(locations.Items).To(ContainElements([]ListItem{
		{
			Id:      "@",
			Enabled: true,
		},
		{
			Id:      "a",
			Enabled: true,
		},
		{
			Id:      "b",
			Enabled: true,
		},
		{
			Id:      "c",
			Enabled: true,
		},
	}))

	// another zone
	locations, err = db.GetLocations(user1Id, zone2Name, 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(locations.Total).To(Equal(4))
	Expect(locations.Items).To(ContainElements([]ListItem{
		{
			Id:      "@",
			Enabled: true,
		},
		{
			Id:      "d",
			Enabled: true,
		},
		{
			Id:      "e",
			Enabled: true,
		},
		{
			Id:      "f",
			Enabled: true,
		},
	}))

	// zone with no locations
	locations, err = db.GetLocations(user1Id, zone3Name, 0, 100, "", true)
	Expect(err).To(BeNil())
	Expect(locations.Total).To(Equal(1))

	// non-existing zone
	locations, err = db.GetLocations(user1Id, "zone4.com.", 0, 100, "", true)
	Expect(err).To(Equal(ErrNotFound))

	// limit results
	locations, err = db.GetLocations(user1Id, zone1Name, 1, 1, "", true)
	Expect(err).To(BeNil())
	Expect(locations.Total).To(Equal(4))
	Expect(len(locations.Items)).To(Equal(1))
	Expect(locations.Items).To(ContainElement(ListItem{
		Id:      "a",
		Enabled: true,
	}))

	// with q
	locations, err = db.GetLocations(user1Id, zone1Name, 0, 100, "b", true)
	Expect(err).To(BeNil())
	Expect(locations.Total).To(Equal(1))
	Expect(locations.Items).To(ContainElement(ListItem{
		Id:      "b",
		Enabled: true,
	}))

	// empty result
	locations, err = db.GetLocations(user1Id, zone1Name, 0, 100, "no-result", true)
	Expect(err).To(BeNil())
	Expect(locations.Total).To(Equal(0))
}

func TestGetLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "a"
	l1Id, err := db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
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
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "a"
	l1Id, err := db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())

	// update
	err = db.UpdateLocation(user1Id, LocationUpdate{ZoneName: zone1Name, Location: l1, Enabled: false})
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
	err = db.UpdateLocation(user1Id, LocationUpdate{ZoneName: zone1Name, Location: "bad", Enabled: true})
	Expect(err).To(Equal(ErrNotFound))
}

func TestDeleteLocation(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "a"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())

	// delete
	err = db.DeleteLocation(user1Id, LocationDelete{ZoneName: zone1Name, Location: l1})
	Expect(err).To(BeNil())
	_, err = db.GetLocation(user1Id, zone1Name, l1)
	Expect(err).To(Equal(ErrNotFound))

	// non-existing zone
	err = db.DeleteLocation(user1Id, LocationDelete{ZoneName: "zone2.com.", Location: l1})
	Expect(err).To(Equal(ErrNotFound))

	// non-existing location
	err = db.DeleteLocation(user1Id, LocationDelete{ZoneName: zone1Name, Location: "b"})
	Expect(err).To(Equal(ErrNotFound))
}

func TestAddRecordSet(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())

	// add
	r1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: "a", Value: r1, Enabled: true})
	Expect(err).To(BeNil())

	// duplicate
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: "a", Value: r1, Enabled: true})
	Expect(err).To(Equal(ErrDuplicateEntry))

	// recordset with invalid type
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: "invalid-type", Value: r1, Enabled: true})
	Expect(err).NotTo(BeNil())

	// recordset with invalid location
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: "www2", Type: "a", Value: r1, Enabled: true})
	Expect(err).To(Equal(ErrNotFound))

	// recordset with invalid zone
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: "zone2.com.", Location: l1, Type: "a", Value: r1, Enabled: true})
	Expect(err).To(Equal(ErrNotFound))

	// add recordset to location
	rrKeys := []ListItem{
		{Id: "a", Enabled: true},
		{Id: "aaaa", Enabled: true},
		{Id: "aname", Enabled: true},
		{Id: "caa", Enabled: true},
		{Id: "cname", Enabled: true},
		{Id: "ds", Enabled: true},
		{Id: "mx", Enabled: true},
		{Id: "ns", Enabled: true},
		{Id: "ptr", Enabled: true},
		{Id: "srv", Enabled: true},
		{Id: "tlsa", Enabled: true},
		{Id: "txt", Enabled: true},
	}
	rrValues := []types.RRSet{
		&types.IP_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.IP_RR{{Ip: net.ParseIP("1.2.3.4")}}},
		&types.IP_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.IP_RR{{Ip: net.ParseIP("::1")}}},
		&types.ANAME_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Location: "aname.example.com."},
		&types.CAA_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.CAA_RR{{Tag: "issue", Flag: 0, Value: "godaddy.com;"}}},
		&types.CNAME_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Host: "x.example.com."},
		&types.DS_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.DS_RR{{Digest: "B6DCD485719ADCA18E5F3D48A2331627FDD3636B", KeyTag: 57855, Algorithm: 5, DigestType: 1}}},
		&types.MX_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.MX_RR{{Host: "mx1.example.com.", Preference: 10}, {Host: "mx2.example.com.", Preference: 10}}},
		&types.NS_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.NS_RR{{Host: "ns1.example.com."}, {Host: "ns2.example.com."}}},
		&types.PTR_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Domain: "localhost"},
		&types.SRV_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.SRV_RR{{Port: 555, Target: "sip.example.com.", Weight: 100, Priority: 10}}},
		&types.TLSA_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.TLSA_RR{{Usage: 0, Selector: 0, Certificate: "d2abde240d7cd3ee6b4b28c54df034b97983a1d16e8a410e4561cb106618e971", MatchingType: 1}}},
		&types.TXT_RRSet{GenericRRSet: types.GenericRRSet{TtlValue: 300}, Data: []types.TXT_RR{{Text: "foo"}, {Text: "bar"}}},
	}
	l2 := "a"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: "a", Enabled: true})
	Expect(err).To(BeNil())
	for i := range rrKeys {
		_, err := db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l2, Type: rrKeys[i].Id, Value: rrValues[i], Enabled: true})
		Expect(err).To(BeNil())
	}
	sets, err := db.GetRecordSets(user1Id, zone1Name, l2)
	Expect(err).To(BeNil())
	Expect(sets.Items).To(ContainElements(rrKeys))
}

func TestGetRecordSets(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example1.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: "a", Value: r1, Enabled: true})
	Expect(err).To(BeNil())
	r2 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("::1"),
			},
		},
	}
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: "aaaa", Value: r2, Enabled: true})
	Expect(err).To(BeNil())

	// get
	r, err := db.GetRecordSets(user1Id, zone1Name, l1)
	Expect(err).To(BeNil())
	Expect(r.Total).To(Equal(2))
	Expect(r.Items).To(ContainElements([]ListItem{
		{
			Id:      "a",
			Enabled: true,
		},
		{
			Id:      "aaaa",
			Enabled: true,
		},
	}))

	// empty location
	l2 := "www2"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l2, Enabled: true})
	Expect(err).To(BeNil())
	r, err = db.GetRecordSets(user1Id, zone1Name, l2)
	Expect(err).To(BeNil())
	Expect(r.Total).To(Equal(0))

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
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	r1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	r1Id, err := db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: r1Type, Value: r1, Enabled: true})
	Expect(err).To(BeNil())

	// get
	r, err := db.GetRecordSet(user1Id, zone1Name, l1, r1Type)
	Expect(err).To(BeNil())
	Expect(r).To(Equal(RecordSet{
		Id:      r1Id,
		Type:    r1Type,
		Value:   r1,
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
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	r1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	r1Id, err := db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: r1Type, Value: r1, Enabled: true})
	Expect(err).To(BeNil())

	// update
	r1.TtlValue = 400
	err = db.UpdateRecordSet(user1Id, RecordSetUpdate{ZoneName: zone1Name, Location: l1, Type: r1Type, Value: r1, Enabled: false})
	Expect(err).To(BeNil())
	r, err := db.GetRecordSet(user1Id, zone1Name, l1, r1Type)
	Expect(err).To(BeNil())
	Expect(r).To(Equal(RecordSet{
		Id:      r1Id,
		Type:    r1Type,
		Value:   r1,
		Enabled: false,
	}))

	// non-existing zone
	err = db.UpdateRecordSet(user1Id, RecordSetUpdate{ZoneName: "zone2.com.", Location: l1, Type: r1Type, Value: r1, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))

	// non-existing location
	err = db.UpdateRecordSet(user1Id, RecordSetUpdate{ZoneName: zone1Name, Location: "xxx", Type: r1Type, Value: r1, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))

	// non-existing record
	err = db.UpdateRecordSet(user1Id, RecordSetUpdate{ZoneName: zone1Name, Location: l1, Type: "aaaa", Value: r1, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))

	// invalid type
	err = db.UpdateRecordSet(user1Id, RecordSetUpdate{ZoneName: zone1Name, Location: l1, Type: "invalid-type", Value: r1, Enabled: false})
	Expect(err).To(Equal(ErrNotFound))
}

func TestDeleteRecordSet(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com."
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	r1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: r1Type, Value: r1, Enabled: true})
	Expect(err).To(BeNil())

	// delete
	err = db.DeleteRecordSet(user1Id, RecordSetDelete{ZoneName: zone1Name, Location: l1, Type: r1Type})
	Expect(err).To(BeNil())

	// non-existing zone
	err = db.DeleteRecordSet(user1Id, RecordSetDelete{ZoneName: "zone2.com.", Location: l1, Type: r1Type})
	Expect(err).To(Equal(ErrNotFound))

	// non-existing location
	err = db.DeleteRecordSet(user1Id, RecordSetDelete{ZoneName: zone1Name, Location: "xxx", Type: r1Type})
	Expect(err).To(Equal(ErrNotFound))

	// non-existing record
	err = db.DeleteRecordSet(user1Id, RecordSetDelete{ZoneName: zone1Name, Location: l1, Type: "aaaa"})
	Expect(err).To(Equal(ErrNotFound))
}

func TestCascadeDelete(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())

	user1Id, _, err := db.AddUser(NewUser{Email: "admin", Password: "admin", Status: UserStatusActive})
	Expect(err).To(BeNil())
	zone1Name := "example.com"
	_, err = db.AddZone(user1Id, NewZone{Name: zone1Name, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	l1 := "www"
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zone1Name, Location: l1, Enabled: true})
	Expect(err).To(BeNil())
	r1Type := "a"
	r1 := &types.IP_RRSet{
		GenericRRSet: types.GenericRRSet{TtlValue: 300},
		Data: []types.IP_RR{
			{
				Ip: net.ParseIP("1.2.3.4"),
			},
		},
	}
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zone1Name, Location: l1, Type: r1Type, Value: r1, Enabled: true})
	Expect(err).To(BeNil())

	err = db.DeleteZone(user1Id, ZoneDelete{Name: zone1Name})
	Expect(err).To(BeNil())

	locations, err := db.GetLocations(user1Id, zone1Name, 0, 100, "", true)
	Expect(err).To(Equal(ErrNotFound))
	Expect(locations.Total).To(Equal(0))
	recordSets, err := db.GetRecordSets(user1Id, zone1Name, l1)
	Expect(err).To(Equal(ErrNotFound))
	Expect(recordSets.Total).To(Equal(0))
}

func TestAutoInsertedItemsAfterAddZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	// add zone for user
	zoneName := "example.com."
	zoneId, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())

	z, err := db.GetZone(user1Id, zoneName)
	Expect(err).To(BeNil())
	Expect(z).To(Equal(Zone{
		Id:              zoneId,
		Name:            zoneName,
		Enabled:         true,
		Dnssec:          false,
		CNameFlattening: false,
		SOA:             soa,
	}))

	_, err = db.GetLocation(user1Id, zoneName, "@")
	Expect(err).To(BeNil())

	nsRecord, err := db.GetRecordSet(user1Id, zoneName, "@", "ns")
	Expect(err).To(BeNil())
	Expect(nsRecord.Value).To(Equal(&ns))
}

func TestEvent(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())

	zoneName := "example.com."
	zoneId, err := db.AddZone(user1Id, NewZone{Name: zoneName, Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	_, err = db.AddLocation(user1Id, NewLocation{ZoneName: zoneName, Location: "www", Enabled: true})
	Expect(err).To(BeNil())
	_, err = db.AddRecordSet(user1Id, NewRecordSet{ZoneName: zoneName, Location: "www", Enabled: true, Type: "a",
		Value: &types.IP_RRSet{
			GenericRRSet: types.GenericRRSet{TtlValue: 300},
			Data:         []types.IP_RR{{Ip: net.ParseIP("1.2.3.4")}},
		}})
	Expect(err).To(BeNil())
	events, err := db.GetEvents(0, 0, 10)
	Expect(events).To(Equal([]Event{
		{
			Revision: 1,
			ZoneId:   string(zoneId),
			Type:     AddZone,
			Value:    `{"ns": {"ttl": 3600, "records": [{"host": "ns1.example.com."}, {"host": "ns2.example.com."}]}, "soa": {"ns": "ns1.example.com.", "ttl": 3600, "mbox": "admin.example.com.", "retry": 55, "expire": 66, "minttl": 100, "serial": 123456, "refresh": 44}, "keys": {"DS": "", "KSKPublic": "", "ZSKPublic": "", "KSKPrivate": "", "ZSKPrivate": ""}, "name": "example.com.", "dnssec": false, "enabled": true, "cname_flattening": false}`,
		},
		{
			Revision: 2,
			ZoneId:   string(zoneId),
			Type:     AddLocation,
			Value:    `{"enabled": true, "location": "www", "zone_name": "example.com."}`,
		},
		{
			Revision: 3,
			ZoneId:   string(zoneId),
			Type:     AddRecord,
			Value:    `{"type": "a", "value": {"ttl": 300, "filter": {}, "records": [{"ip": "1.2.3.4"}], "health_check": {}}, "enabled": true, "location": "www", "zone_name": "example.com."}`,
		},
	}))
}

func TestImportZone(t *testing.T) {
	RegisterTestingT(t)
	err := db.Clear(true)
	Expect(err).To(BeNil())
	user1Id, _, err := db.AddUser(NewUser{Email: "dbUser1", Password: "dbUser1", Status: UserStatusActive})
	Expect(err).To(BeNil())
	_, err = db.AddZone(user1Id, NewZone{Name: "example1.com.", Dnssec: false, CNameFlattening: false, Enabled: true, SOA: soa, NS: ns})
	Expect(err).To(BeNil())
	zoneImport := ZoneImport{
		Name: "example1.com.",
		Entries: map[string]map[string]types.RRSet{
			"@": {
				types.TypeToString(dns.TypeNS): &types.NS_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 300},
					Data:         []types.NS_RR{{Host: "ns11.example1.com."}, {Host: "ns22.example11.com."}},
				},
				types.TypeToString(dns.TypeA): &types.IP_RRSet{
					GenericRRSet:      types.GenericRRSet{TtlValue: 300},
					FilterConfig:      types.IpFilterConfig{},
					HealthCheckConfig: types.IpHealthCheckConfig{},
					Data:              []types.IP_RR{{Ip: net.ParseIP("1.2.3.4")}, {Ip: net.ParseIP("2.3.4.5")}},
				},
			},
			"www": {
				types.TypeToString(dns.TypeA): &types.IP_RRSet{
					GenericRRSet:      types.GenericRRSet{TtlValue: 300},
					FilterConfig:      types.IpFilterConfig{},
					HealthCheckConfig: types.IpHealthCheckConfig{},
					Data:              []types.IP_RR{{Ip: net.ParseIP("1.2.3.4")}, {Ip: net.ParseIP("2.3.4.5")}},
				},
				types.TypeToString(dns.TypeTXT): &types.TXT_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 300},
					Data:         []types.TXT_RR{{Text: "1234"}, {Text: "hello"}, {Text: "foo bar"}},
				},
			},
			"a": {
				types.TypeToString(dns.TypeCNAME): &types.CNAME_RRSet{
					GenericRRSet: types.GenericRRSet{TtlValue: 300},
					Host:         "example1.com.",
				},
			},
		},
	}

	err = db.ImportZone(user1Id, zoneImport)
	Expect(err).To(BeNil())
	for label, location := range zoneImport.Entries {
		_, err = db.GetLocation(user1Id, "example1.com.", label)
		Expect(err).To(BeNil())

		for rtype, rrset := range location {
			rr, err := db.GetRecordSet(user1Id, "example1.com.", label, rtype)
			Expect(err).To(BeNil())
			Expect(rr.Value).To(Equal(rrset))
		}
	}
}

func TestMain(m *testing.M) {
	db, _ = Connect(connectionStr)
	m.Run()
	_ = db.Close()
}
