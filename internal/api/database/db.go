package database

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/hawell/z42/internal/types"
	jsoniter "github.com/json-iterator/go"
	"log"
	"math/rand"
	"time"
)

var src = rand.NewSource(time.Now().UnixNano())

const (
	letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randomString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

var (
	ErrDuplicateEntry = errors.New("duplicate entry")
	ErrNotFound       = errors.New("not found")
	ErrInvalid        = errors.New("invalid operation")
	ErrUnauthorized   = errors.New("authorization failed")
)

type DataBase struct {
	db *sql.DB
}

func Connect(connectionString string) (*DataBase, error) {
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, parseError(err)
	}
	return &DataBase{db}, nil
}

func (db *DataBase) Close() error {
	return db.db.Close()
}

func (db *DataBase) Clear(removeUsers bool) error {
	var err error
	err = db.withTransaction(func(t transaction) error {
		if removeUsers {
			if _, err := t.Exec("DELETE FROM User"); err != nil {
				return err
			}
		} else {
			if _, err := t.Exec("DELETE FROM Zone"); err != nil {
				return err
			}
			if _, err := t.Exec("DELETE FROM ACL"); err != nil {
				return err
			}
			if _, err := t.Exec("DELETE FROM Verification"); err != nil {
				return err
			}
		}
		if _, err := t.Exec("DELETE FROM Events"); err != nil {
			return err
		}
		if _, err := t.Exec("ALTER TABLE Events AUTO_INCREMENT = 1"); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) AddUser(u NewUser) (ObjectId, error) {
	hash, err := HashPassword(u.Password)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	id := NewObjectId()
	_, err = db.db.Exec("INSERT INTO User(Id, Email, Password, Status) VALUES (?, ?, ?, ?)", id, u.Email, hash, u.Status)
	if err != nil {
		log.Println(err)
		return EmptyObjectId, parseError(err)
	}
	return id, nil
}

func (db *DataBase) AddVerification(name string, verificationType string) (string, error) {
	u, err := db.GetUser(name)
	if err != nil {
		if err == ErrNotFound {
			return "", ErrInvalid
		}
		return "", parseError(err)
	}
	code := randomString(50)
	_, err = db.db.Exec("INSERT INTO Verification(Code, Type, User_Id) VALUES (?, ?, ?)", code, verificationType, u.Id)
	if err != nil {
		return "", parseError(err)
	}
	return code, nil
}

func (db *DataBase) Verify(code string) error {
	res := db.db.QueryRow("select U.Id, V.Type from Verification V left join User U on U.Id = V.User_Id WHERE Code = ?", code)
	var (
		userId           ObjectId
		verificationType string
	)
	if err := res.Scan(&userId, &verificationType); err != nil {
		return parseError(err)
	}
	switch verificationType {
	case VerificationTypeSignup:
		if _, err := db.db.Exec("UPDATE User SET Status = ? WHERE Id = ?", UserStatusActive, userId); err != nil {
			return parseError(err)
		}
		if _, err := db.db.Exec("DELETE FROM Verification WHERE Code = ?", code); err != nil {
			return parseError(err)
		}
	default:
		return errors.New("unknown verification type")
	}

	return nil
}

func (db *DataBase) GetUser(name string) (User, error) {
	res := db.db.QueryRow("SELECT Id, Email, Password, Status FROM User WHERE Email = ?", name)
	var u User
	err := res.Scan(&u.Id, &u.Email, &u.Password, &u.Status)
	return u, parseError(err)
}

func (db *DataBase) DeleteUser(name string) (int64, error) {
	res, err := db.db.Exec("DELETE FROM User WHERE Email = ?", name)
	if err != nil {
		return 0, parseError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, parseError(err)
	}
	if rows == 0 {
		return 0, ErrNotFound
	}
	return rows, parseError(err)
}

func (db *DataBase) getZoneOwner(zoneName string) (ObjectId, error) {
	res := db.db.QueryRow("SELECT User_Id FROM Zone WHERE Name = ?", zoneName)
	var userId ObjectId
	err := res.Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmptyObjectId, nil
		}
		return EmptyObjectId, parseError(err)
	}
	return userId, nil
}

func (db *DataBase) GetEvents(revision int, start int, count int) ([]Event, error) {
	rows, err := db.db.Query("SELECT Revision, ZoneId, Type, Value FROM Events WHERE Revision > ? ORDER BY Revision LIMIT ?, ?", revision, start, count)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var res []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(&event.Revision, &event.ZoneId, &event.Type, &event.Value)
		if err != nil {
			return nil, err
		}
		res = append(res, event)
	}
	return res, nil
}

func (db *DataBase) AddZone(userId ObjectId, z NewZone) (ObjectId, error) {
	owner, err := db.getZoneOwner(z.Name)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if owner == EmptyObjectId {
		var zoneId ObjectId
		err := db.withTransaction(func(t transaction) error {
			var err error
			zoneId, err = db.addZone(userId, z)
			if err != nil {
				return err
			}
			if err = db.setPrivileges(userId, zoneId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: true}); err != nil {
				return err
			}
			if err = db.setSOA(zoneId, z.SOA); err != nil {
				return err
			}
			rootLocation := NewLocation{
				ZoneName: z.Name,
				Location: "@",
				Enabled:  true,
			}
			locationId, err := db.addLocation(zoneId, rootLocation)
			if err != nil {
				return err
			}
			if err = db.setPrivileges(userId, locationId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: false}); err != nil {
				return err
			}
			nsRecord := NewRecordSet{Type: "ns", Value: &z.NS, Enabled: true}
			nsId, err := db.addRecordSet(locationId, nsRecord)
			if err != nil {
				return err
			}
			if err := db.setPrivileges(userId, nsId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: false}); err != nil {
				return err
			}
			if _, err = db.addEvent(zoneId, AddZone, z); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return EmptyObjectId, parseError(err)
		}
		return zoneId, nil
	}
	if owner == userId {
		return EmptyObjectId, ErrDuplicateEntry
	}
	return EmptyObjectId, ErrInvalid
}

func (db *DataBase) GetZones(userId ObjectId, start int, count int, q string, ascendingOrder bool) (List, error) {
	like := "%" + q + "%"
	query := "SELECT Name, Enabled FROM Zone WHERE User_Id = ? AND Name LIKE ? ORDER BY Name LIMIT ?, ?"
	if !ascendingOrder {
		query = "SELECT Name, Enabled FROM Zone WHERE User_Id = ? AND Name LIKE ? ORDER BY Name DESC LIMIT ?, ?"
	}
	rows, err := db.db.Query(query, userId, like, start, count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmptyList(), nil
		}
		return List{}, parseError(err)
	}
	defer func() { _ = rows.Close() }()
	res := List{}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return List{}, parseError(err)
		}
		res.Items = append(res.Items, item)
	}
	row := db.db.QueryRow("SELECT count(*) FROM Zone WHERE User_Id = ? AND Name LIKE ?", userId, like)
	if err := row.Scan(&res.Total); err != nil {
		return List{}, parseError(err)
	}
	return res, nil
}

func (db *DataBase) GetZone(userId ObjectId, zoneName string) (Zone, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return Zone{}, parseError(err)
	}
	if err := db.canGetZone(userId, zoneId); err != nil {
		return Zone{}, parseError(err)
	}
	res := db.db.QueryRow("SELECT Id, Name, CNameFlattening, Dnssec, Enabled, TTL, NS, MBox, Refresh, Retry, Expire, MinTTL, Serial FROM Zone LEFT JOIN SOA ON Zone.Id = SOA.Zone_Id WHERE Zone.Id = ?", zoneId)
	var (
		z Zone
	)
	err = res.Scan(&z.Id, &z.Name, &z.CNameFlattening, &z.Dnssec, &z.Enabled, &z.SOA.TtlValue, &z.SOA.Ns, &z.SOA.MBox, &z.SOA.Refresh, &z.SOA.Retry, &z.SOA.Expire, &z.SOA.MinTtl, &z.SOA.Serial)
	if err != nil {
		return Zone{}, parseError(err)
	}
	return z, nil
}

func (db *DataBase) UpdateZone(userId ObjectId, z ZoneUpdate) error {
	zoneId, err := db.getZoneId(z.Name)
	if err != nil {
		return parseError(err)
	}
	err = db.withTransaction(func(t transaction) error {
		if err := db.canUpdateZone(userId, zoneId); err != nil {
			return err
		}
		if err := db.updateZone(zoneId, z); err != nil {
			return err
		}
		if err := db.setSOA(zoneId, z.SOA); err != nil {
			return err
		}
		if _, err := db.addEvent(zoneId, UpdateZone, z); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return parseError(err)
	}
	return nil
}

func (db *DataBase) DeleteZone(userId ObjectId, z ZoneDelete) error {
	zoneId, err := db.getZoneId(z.Name)
	if err != nil {
		return parseError(err)
	}
	err = db.withTransaction(func(t transaction) error {
		if err := db.canDeleteZone(userId, zoneId); err != nil {
			return err
		}
		if err := db.deleteZone(zoneId); err != nil {
			return err
		}
		if err := db.deletePrivileges(zoneId); err != nil {
			return err
		}
		if _, err := db.addEvent(zoneId, DeleteZone, z.Name); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return parseError(err)
	}
	return nil
}

func (db *DataBase) AddLocation(userId ObjectId, l NewLocation) (ObjectId, error) {
	zoneId, err := db.getZoneId(l.ZoneName)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if err := db.canAddLocation(userId, zoneId); err != nil {
		return EmptyObjectId, parseError(err)
	}
	var locationId ObjectId
	err = db.withTransaction(func(t transaction) error {
		var err error
		locationId, err = db.addLocation(zoneId, l)
		if err != nil {
			return err
		}
		if err := db.setPrivileges(userId, locationId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: true}); err != nil {
			return err
		}
		if _, err := db.addEvent(zoneId, AddLocation, l); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	return locationId, nil
}

func (db *DataBase) GetLocations(userId ObjectId, zoneName string, start int, count int, q string, ascendingOrder bool) (List, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return List{}, parseError(err)
	}
	if err := db.canGetLocations(userId, zoneId); err != nil {
		return List{}, parseError(err)
	}
	like := "%" + q + "%"
	query := "SELECT Name, Enabled FROM Location WHERE Zone_Id = ? AND Name LIKE ? ORDER BY Name LIMIT ?, ?"
	if !ascendingOrder {
		query = "SELECT Name, Enabled FROM Location WHERE Zone_Id = ? AND Name LIKE ? ORDER BY Name DESC LIMIT ?, ?"
	}
	rows, err := db.db.Query(query, zoneId, like, start, count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmptyList(), nil
		}
		return List{}, parseError(err)
	}
	defer func() { _ = rows.Close() }()
	res := List{}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return List{}, parseError(err)
		}
		res.Items = append(res.Items, item)
	}
	row := db.db.QueryRow("SELECT count(*) FROM Location WHERE Zone_Id = ? AND Name LIKE ?", zoneId, like)
	if err := row.Scan(&res.Total); err != nil {
		return List{}, parseError(err)
	}
	return res, nil
}

func (db *DataBase) GetLocation(userId ObjectId, zoneName string, location string) (Location, error) {
	zoneId, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return Location{}, parseError(err)
	}
	if err := db.canGetLocation(userId, zoneId, locationId); err != nil {
		return Location{}, parseError(err)
	}
	res := db.db.QueryRow("SELECT Id, Name, Enabled FROM Location WHERE Id = ?", locationId)
	var l Location
	err = res.Scan(&l.Id, &l.Name, &l.Enabled)
	return l, parseError(err)
}

func (db *DataBase) UpdateLocation(userId ObjectId, l LocationUpdate) error {
	zoneId, locationId, err := db.getLocationId(l.ZoneName, l.Location)
	if err != nil {
		return parseError(err)
	}
	err = db.withTransaction(func(t transaction) error {
		if err := db.canUpdateLocation(userId, zoneId, locationId); err != nil {
			return err
		}
		if err := db.updateLocation(locationId, l); err != nil {
			return err
		}
		if _, err := db.addEvent(zoneId, UpdateLocation, l); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return parseError(err)
	}
	return nil
}

func (db *DataBase) DeleteLocation(userId ObjectId, l LocationDelete) error {
	zoneId, locationId, err := db.getLocationId(l.ZoneName, l.Location)
	if err != nil {
		return parseError(err)
	}
	err = db.withTransaction(func(t transaction) error {
		if err := db.canDeleteLocation(userId, zoneId, locationId); err != nil {
			return err
		}
		if err := db.deleteLocation(locationId); err != nil {
			return err
		}
		if _, err := db.addEvent(zoneId, DeleteLocation, l); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return parseError(err)
	}
	return nil
}

func (db *DataBase) AddRecordSet(userId ObjectId, r NewRecordSet) (ObjectId, error) {
	zoneId, locationId, err := db.getLocationId(r.ZoneName, r.Location)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if err := db.canAddRecordSet(userId, zoneId, locationId); err != nil {
		return EmptyObjectId, parseError(err)
	}
	var recordId ObjectId
	err = db.withTransaction(func(t transaction) error {
		var err error
		if recordId, err = db.addRecordSet(locationId, r); err != nil {
			return err
		}
		if err = db.setPrivileges(userId, recordId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: true}); err != nil {
			return err
		}
		if _, err = db.addEvent(zoneId, AddRecord, r); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	return recordId, nil
}

func (db *DataBase) GetRecordSets(userId ObjectId, zoneName string, location string) (List, error) {
	zoneId, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return List{}, parseError(err)
	}
	if err := db.canGetRecordSets(userId, zoneId, locationId); err != nil {
		return List{}, parseError(err)
	}
	rows, err := db.db.Query("SELECT Type, Enabled FROM RecordSet WHERE Location_Id = ? ORDER BY Type", locationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmptyList(), nil
		}
		return List{}, parseError(err)
	}
	defer func() { _ = rows.Close() }()
	res := List{}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return List{}, parseError(err)
		}
		res.Items = append(res.Items, item)
	}
	row := db.db.QueryRow("SELECT count(*) FROM RecordSet WHERE Location_Id = ?", locationId)
	if err := row.Scan(&res.Total); err != nil {
		return List{}, parseError(err)
	}
	return res, nil
}

func (db *DataBase) GetRecordSet(userId ObjectId, zoneName string, location string, recordType string) (RecordSet, error) {
	zoneId, locationId, recordId, err := db.getRecordId(zoneName, location, recordType)
	if err != nil {
		return RecordSet{}, parseError(err)
	}
	if err := db.canGetRecordSet(userId, zoneId, locationId, recordId); err != nil {
		return RecordSet{}, parseError(err)
	}
	row := db.db.QueryRow("SELECT Id, Type, Value, Enabled FROM RecordSet WHERE Id = ?", recordId)
	var (
		r     RecordSet
		value string
	)
	err = row.Scan(&r.Id, &r.Type, &value, &r.Enabled)
	rrset := types.TypeToRRSet[recordType]()
	err = jsoniter.Unmarshal([]byte(value), &rrset)
	if err != nil {
		return RecordSet{}, err
	}
	r.Value = rrset
	return r, parseError(err)
}

func (db *DataBase) UpdateRecordSet(userId ObjectId, r RecordSetUpdate) error {
	zoneId, locationId, recordId, err := db.getRecordId(r.ZoneName, r.Location, r.Type)
	if err != nil {
		return parseError(err)
	}
	err = db.withTransaction(func(t transaction) error {
		if err := db.canUpdateRecordSet(userId, zoneId, locationId, recordId); err != nil {
			return err
		}
		if err := db.updateRecordSet(recordId, r); err != nil {
			return err
		}
		if _, err := db.addEvent(zoneId, UpdateRecord, r); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return parseError(err)
	}
	return nil
}

func (db *DataBase) DeleteRecordSet(userId ObjectId, r RecordSetDelete) error {
	zoneId, locationId, recordId, err := db.getRecordId(r.ZoneName, r.Location, r.Type)
	if err != nil {
		return parseError(err)
	}
	err = db.withTransaction(func(t transaction) error {
		if err := db.canDeleteRecordSet(userId, zoneId, locationId, recordId); err != nil {
			return parseError(err)
		}
		if err := db.deleteRecordSet(recordId); err != nil {
			return parseError(err)
		}
		if _, err := db.addEvent(zoneId, DeleteRecord, r); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return parseError(err)
	}
	return nil
}

func (db *DataBase) getZoneId(zoneName string) (ObjectId, error) {
	var zoneId ObjectId
	res := db.db.QueryRow("SELECT Id FROM Zone WHERE Name = ?", zoneName)
	err := res.Scan(&zoneId)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	return zoneId, nil
}

func (db *DataBase) getLocationId(zoneName string, location string) (ObjectId, ObjectId, error) {
	res := db.db.QueryRow("SELECT Zone.Id, L.Id FROM Zone LEFT JOIN Location L on Zone.Id = L.Zone_Id WHERE Zone.Name = ? AND L.Name = ?", zoneName, location)
	var (
		zoneId     ObjectId
		locationId ObjectId
	)
	err := res.Scan(&zoneId, &locationId)
	if err != nil {
		return EmptyObjectId, EmptyObjectId, parseError(err)
	}
	return zoneId, locationId, nil
}

func (db *DataBase) getRecordId(zoneName string, location string, recordType string) (ObjectId, ObjectId, ObjectId, error) {
	res := db.db.QueryRow("SELECT Zone.Id, L.Id, RS.Id FROM Zone LEFT JOIN Location L on Zone.Id = L.Zone_Id LEFT JOIN RecordSet RS on L.Id = RS.Location_Id WHERE Zone.Name = ? AND L.Name = ? AND RS.Type = ?", zoneName, location, recordType)
	var (
		zoneId     ObjectId
		locationId ObjectId
		recordId   ObjectId
	)
	err := res.Scan(&zoneId, &locationId, &recordId)
	if err != nil {
		return EmptyObjectId, EmptyObjectId, EmptyObjectId, parseError(err)
	}
	return zoneId, locationId, recordId, nil
}

func (db *DataBase) setSOA(zoneId ObjectId, soa types.SOA_RRSet) error {
	_, err := db.db.Exec("REPLACE INTO SOA VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", soa.TtlValue, soa.Ns, soa.MBox, soa.Refresh, soa.Retry, soa.Expire, soa.MinTtl, soa.Serial, zoneId)
	return err
}

func (db *DataBase) addZone(userId ObjectId, z NewZone) (ObjectId, error) {
	zoneId := NewObjectId()
	if _, err := db.db.Exec("INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (?, ?, ?, ?, ?, ?)", zoneId, z.Name, z.CNameFlattening, z.Dnssec, z.Enabled, userId); err != nil {
		return EmptyObjectId, err
	}
	return zoneId, nil
}

func (db *DataBase) updateZone(zoneId ObjectId, z ZoneUpdate) error {
	_, err := db.db.Exec("UPDATE Zone SET Name = ?, Dnssec = ?, CNameFlattening = ?, Enabled = ? WHERE Id = ?", z.Name, z.Dnssec, z.CNameFlattening, z.Enabled, zoneId)
	return err
}

func (db *DataBase) deleteZone(zoneId ObjectId) error {
	_, err := db.db.Exec("DELETE FROM Zone WHERE Id = ?", zoneId)
	return err
}

func (db *DataBase) addLocation(zoneId ObjectId, l NewLocation) (ObjectId, error) {
	locationId := NewObjectId()
	if _, err := db.db.Exec("INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (?, ?, ?, ?)", locationId, l.Location, l.Enabled, zoneId); err != nil {
		return EmptyObjectId, err
	}
	return locationId, nil
}

func (db *DataBase) updateLocation(locationId ObjectId, l LocationUpdate) error {
	_, err := db.db.Exec("UPDATE Location SET Name = ?, Enabled = ? WHERE Id = ?", l.Location, l.Enabled, locationId)
	return err
}

func (db *DataBase) deleteLocation(locationId ObjectId) error {
	_, err := db.db.Exec("DELETE FROM Location WHERE Id = ?", locationId)
	return err
}

func (db *DataBase) addRecordSet(locationId ObjectId, r NewRecordSet) (ObjectId, error) {
	value, err := jsoniter.Marshal(r.Value)
	if err != nil {
		return EmptyObjectId, err
	}
	recordId := NewObjectId()
	if _, err := db.db.Exec("INSERT INTO RecordSet(Id, Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?, ?)", recordId, locationId, r.Type, value, r.Enabled); err != nil {
		return EmptyObjectId, err
	}
	return recordId, nil
}

func (db *DataBase) updateRecordSet(recordId ObjectId, r RecordSetUpdate) error {
	value, err := jsoniter.Marshal(r.Value)
	if err != nil {
		return err
	}
	_, err = db.db.Exec("UPDATE RecordSet SET Value = ?, Enabled = ?  WHERE Id = ?", value, r.Enabled, recordId)
	return err
}

func (db *DataBase) deleteRecordSet(recordId ObjectId) error {
	_, err := db.db.Exec("DELETE FROM RecordSet WHERE Id = ?", recordId)
	return err
}

func (db *DataBase) addEvent(zoneId ObjectId, eventType EventType, value interface{}) (int64, error) {
	jsonValue, err := jsoniter.Marshal(value)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("INSERT INTO Events(ZoneId, Type, Value) VALUES (?, ?, ?)", zoneId, eventType, jsonValue)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func parseError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062:
			return ErrDuplicateEntry
		case 1452:
			return ErrInvalid
		default:
			return err
		}
	}
	return err
}
