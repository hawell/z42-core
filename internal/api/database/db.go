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
	if removeUsers {
		_, err := db.db.Exec("DELETE FROM User")
		return parseError(err)
	}
	err := db.withTransaction(func(t transaction) error {
		if _, err := db.db.Exec("DELETE FROM Zone"); err != nil {
			return parseError(err)
		}
		if _, err := db.db.Exec("DELETE FROM ACL"); err != nil {
			return parseError(err)
		}
		if _, err := db.db.Exec("DELETE FROM Verification"); err != nil {
			return parseError(err)
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

func (db *DataBase) AddZone(userId ObjectId, z NewZone, soa types.SOA_RRSet, ns types.NS_RRSet) (ObjectId, error) {
	owner, err := db.getZoneOwner(z.Name)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if owner == EmptyObjectId {
		zoneId := NewObjectId()
		err := db.withTransaction(func(t transaction) error {
			if _, err := db.db.Exec("INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (?, ?, ?, ?, ?, ?)", zoneId, z.Name, z.CNameFlattening, z.Dnssec, z.Enabled, userId); err != nil {
				return err
			}
			if err := db.setPrivileges(userId, zoneId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: true}); err != nil {
				return err
			}
			locationId := NewObjectId()
			if _, err := db.db.Exec("INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (?, ?, ?, ?)", locationId, "@", true, zoneId); err != nil {
				return err
			}
			if err := db.setPrivileges(userId, locationId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: false}); err != nil {
				return err
			}
			soaId := NewObjectId()
			soaValue, err := jsoniter.Marshal(soa)
			if err != nil {
				return err
			}
			if _, err = db.db.Exec("INSERT INTO RecordSet(Id, Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?, ?)", soaId, locationId, "soa", soaValue, true); err != nil {
				return err
			}
			if err := db.setPrivileges(userId, soaId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: false}); err != nil {
				return err
			}
			nsId := NewObjectId()
			nsValue, err := jsoniter.Marshal(ns)
			if err != nil {
				return err
			}
			if _, err = db.db.Exec("INSERT INTO RecordSet(Id, Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?, ?)", nsId, locationId, "ns", nsValue, true); err != nil {
				return err
			}
			if err := db.setPrivileges(userId, nsId, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: false}); err != nil {
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

func (db *DataBase) GetZones(userId ObjectId, start int, count int, q string) (List, error) {
	like := "%" + q + "%"
	rows, err := db.db.Query("SELECT Name, Enabled FROM Zone WHERE User_Id = ? AND Name LIKE ? ORDER BY Name LIMIT ?, ?", userId, like, start, count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return List{}, nil
		}
		return nil, parseError(err)
	}
	defer func() {_ = rows.Close()}()
	res := List{}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return nil, parseError(err)
		}
		res = append(res, item)
	}
	return res, nil
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

func (db *DataBase) GetZone(userId ObjectId, zoneName string) (Zone, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return Zone{}, parseError(err)
	}
	if err := db.canGetZone(userId, zoneId); err != nil {
		return Zone{}, parseError(err)
	}
	res := db.db.QueryRow("SELECT Id, Name, CNameFlattening, Dnssec, Enabled FROM Zone WHERE Id = ?", zoneId)
	var z Zone
	err = res.Scan(&z.Id, &z.Name, &z.CNameFlattening, &z.Dnssec, &z.Enabled)
	if err != nil {
		return Zone{}, parseError(err)
	}
	return z, nil
}

func (db *DataBase) UpdateZone(userId ObjectId, zoneName string, z ZoneUpdate) (int64, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return 0, parseError(err)
	}
	var rowsAffected int64
	err = db.withTransaction(func(t transaction) error {
		if err := db.canUpdateZone(userId, zoneId); err != nil {
			return err
		}
		res, err := db.db.Exec("UPDATE Zone SET Dnssec = ?, CNameFlattening = ?, Enabled = ? WHERE Id = ?", z.Dnssec, z.CNameFlattening, z.Enabled, zoneId)
		if err != nil {
			return err
		}
		rowsAffected, err = res.RowsAffected()
		if err != nil {
			return err
		}
		soa, err := db.GetRecordSet(userId, zoneName, "@", "soa")
		if err != nil {
			return err
		}
		if _, err = db.db.Exec("UPDATE RecordSet SET Value = ? WHERE Id = ?", z.SOA, soa.Id); err != nil {
			return err
		}
		if err := db.setPrivileges(userId, soa.Id, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: false}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, parseError(err)
	}
	return rowsAffected, nil
}

func (db *DataBase) DeleteZone(userId ObjectId, zoneName string) (int64, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return 0, parseError(err)
	}
	if err := db.canDeleteZone(userId, zoneId); err != nil {
		return 0, parseError(err)
	}
	res, err := db.db.Exec("DELETE FROM Zone WHERE Id = ?", zoneId)
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

func (db *DataBase) AddLocation(userId ObjectId, zoneName string, l NewLocation) (ObjectId, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if err := db.canAddLocation(userId, zoneId); err != nil {
		return EmptyObjectId, parseError(err)
	}
	id := NewObjectId()
	err = db.withTransaction(func(t transaction) error {
		if _, err := db.db.Exec("INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (?, ?, ?, ?)", id, l.Name, l.Enabled, zoneId); err != nil {
			return parseError(err)
		}
		if err := db.setPrivileges(userId, id, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: true}); err != nil {
			return parseError(err)
		}
		return nil
	})
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	return id, nil
}

func (db *DataBase) GetLocations(userId ObjectId, zoneName string, start int, count int, q string) (List, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return nil, parseError(err)
	}
	if err := db.canGetLocations(userId, zoneId); err != nil {
		return nil, parseError(err)
	}
	like := "%" + q + "%"
	rows, err := db.db.Query("SELECT Name, Enabled FROM Location WHERE Zone_Id = ? AND Name LIKE ? ORDER BY Name LIMIT ?, ?", zoneId, like, start, count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return List{}, nil
		}
		return nil, parseError(err)
	}
	defer func() {_ = rows.Close() }()
	res := List{}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return nil, parseError(err)
		}
		res = append(res, item)
	}
	return res, nil
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

func (db *DataBase) UpdateLocation(userId ObjectId, zoneName string, location string, l LocationUpdate) (int64, error) {
	zoneId, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return 0, parseError(err)
	}
	if err := db.canUpdateLocation(userId, zoneId, locationId); err != nil {
		return 0, parseError(err)
	}
	res, err := db.db.Exec("UPDATE Location SET Enabled = ? WHERE Id = ?", l.Enabled, locationId)
	if err != nil {
		return 0, parseError(err)
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteLocation(userId ObjectId, zoneName string, location string) (int64, error) {
	zoneId, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return 0, parseError(err)
	}
	if err := db.canDeleteLocation(userId, zoneId, locationId); err != nil {
		return 0, parseError(err)
	}
	res, err := db.db.Exec("DELETE FROM Location WHERE Id = ?", locationId)
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

func (db *DataBase) AddRecordSet(userId ObjectId, zoneName string, location string, r NewRecordSet) (ObjectId, error) {
	zoneId, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if err := db.canAddRecordSet(userId, zoneId, locationId); err != nil {
		return EmptyObjectId, parseError(err)
	}
	id := NewObjectId()
	err = db.withTransaction(func(t transaction) error {
		if _, err = db.db.Exec("INSERT INTO RecordSet(Id, Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?, ?)", id, locationId, r.Type, r.Value, r.Enabled); err != nil {
			return parseError(err)
		}
		if err := db.setPrivileges(userId, id, ACL{Read: true, List: true, Edit: true, Insert: true, Delete: true}); err != nil {
			return parseError(err)
		}
		return nil
	})
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	return id, nil
}

func (db *DataBase) GetRecordSets(userId ObjectId, zoneName string, location string) (List, error) {
	zoneId, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return nil, parseError(err)
	}
	if err := db.canGetRecordSets(userId, zoneId, locationId); err != nil {
		return nil, parseError(err)
	}
	rows, err := db.db.Query("SELECT Type, Enabled FROM RecordSet WHERE Location_Id = ?", locationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return List{}, nil
		}
		return nil, parseError(err)
	}
	defer func() {_ = rows.Close()}()
	res := List{}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return nil, parseError(err)
		}
		res = append(res, item)
	}
	return res, nil
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

func (db *DataBase) GetRecordSet(userId ObjectId, zoneName string, location string, recordType string) (RecordSet, error) {
	zoneId, locationId, recordId, err := db.getRecordId(zoneName, location, recordType)
	if err != nil {
		return RecordSet{}, parseError(err)
	}
	if err := db.canGetRecordSet(userId, zoneId, locationId, recordId); err != nil {
		return RecordSet{}, parseError(err)
	}
	row := db.db.QueryRow("SELECT Id, Type, Value, Enabled FROM RecordSet WHERE Id = ?", recordId)
	var r RecordSet
	err = row.Scan(&r.Id, &r.Type, &r.Value, &r.Enabled)
	return r, parseError(err)
}

func (db *DataBase) UpdateRecordSet(userId ObjectId, zoneName string, location string, recordType string, r RecordSetUpdate) (int64, error) {
	zoneId, locationId, recordId, err := db.getRecordId(zoneName, location, recordType)
	if err != nil {
		return 0, parseError(err)
	}
	if err := db.canUpdateRecordSet(userId, zoneId, locationId, recordId); err != nil {
		return 0, parseError(err)
	}
	res, err := db.db.Exec("UPDATE RecordSet SET Value = ?, Enabled = ?  WHERE Id = ?", r.Value, r.Enabled, recordId)
	if err != nil {
		return 0, parseError(err)
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteRecordSet(userId ObjectId, zoneName string, location string, recordType string) (int64, error) {
	zoneId, locationId, recordId, err := db.getRecordId(zoneName, location, recordType)
	if err != nil {
		return 0, parseError(err)
	}
	if err := db.canDeleteRecordSet(userId, zoneId, locationId, recordId); err != nil {
		return 0, parseError(err)
	}
	res, err := db.db.Exec("DELETE FROM RecordSet WHERE Id = ?", recordId)
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
