package database

import (
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"math/rand"
	"time"
)

func randomString(n int) string {
	const (
		letterBytes   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	src := rand.NewSource(time.Now().UnixNano())
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
	err = db.withTransaction(func(t *sql.Tx) error {
		if removeUsers {
			if err := deleteUsers(t); err != nil {
				return err
			}
		}
		if err := deleteResources(t); err != nil {
			return err
		}
		if err := deleteVerifications(t); err != nil {
			return err
		}
		if err := deleteEvents(t); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) AddUser(u NewUser) (ObjectId, string, error) {
	hash, err := HashPassword(u.Password)
	if err != nil {
		return EmptyObjectId, "", err
	}
	u.Password = hash
	userId := NewObjectId()
	code := randomString(50)
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := addUser(t, userId, u); err != nil {
			return err
		}
		if u.Status == UserStatusPending {
			err := setVerification(t, userId, Verification{Code: code, Type: VerificationTypeSignup})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return userId, code, parseError(err)
}

func (db *DataBase) Verify(code string) error {
	err := db.applyVerifiedAction(code, VerificationTypeSignup, func(t *sql.Tx, userId ObjectId) error {
		return setUserStatus(t, userId, UserStatusActive)
	})
	return parseError(err)
}

func (db *DataBase) SetRecoveryCode(userId ObjectId) (string, error) {
	code := randomString(50)
	err := db.withTransaction(func(t *sql.Tx) error {
		err := setVerification(t, userId, Verification{Type: VerificationTypeRecover, Code: code})
		return err
	})
	if err != nil {
		return "", parseError(err)
	}
	return code, nil
}

func (db *DataBase) ResetPassword(code string, newPassword string) error {
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	err = db.applyVerifiedAction(code, VerificationTypeRecover, func(t *sql.Tx, userId ObjectId) error {
		return setUserPassword(t, userId, hash)
	})
	return parseError(err)
}

func (db *DataBase) AddAPIKey(userId ObjectId, newAPIKey APIKeyItem) (string, error) {
	owner, err := db.getZoneOwner(newAPIKey.ZoneName)
	if err != nil {
		return "", parseError(err)
	}
	if owner != userId {
		return "", ErrInvalid
	}
	zoneId, err := db.getZoneId(newAPIKey.ZoneName)
	if err != nil {
		return "", parseError(err)
	}
	key := randomString(50)
	hash, err := HashPassword(key)
	if err != nil {
		return "", err
	}
	err = db.addAPIKey(userId, zoneId, newAPIKey, hash)
	return key, parseError(err)
}

func (db *DataBase) GetAPIKeys(userId ObjectId) ([]APIKeyItem, error) {
	res, err := db.getAPIKeys(userId)
	return res, parseError(err)
}

func (db *DataBase) GetAPIKey(userId ObjectId, name string) (APIKeyItem, error) {
	item, err := db.getAPIKey(userId, name)
	return item, parseError(err)
}

func (db *DataBase) UpdateAPIKey(userId ObjectId, model APIKeyUpdate) error {
	return parseError(db.updateAPIKey(userId, model))
}

func (db *DataBase) DeleteAPIKey(userId ObjectId, name string) error {
	return parseError(db.deleteAPIKey(userId, name))
}

func (db *DataBase) GetUser(name string) (User, error) {
	u, err := db.getUser(name)
	return u, parseError(err)
}

func (db *DataBase) DeleteUser(name string) error {
	return parseError(db.deleteUser(name))
}

func (db *DataBase) GetEvents(revision int, start int, count int) ([]Event, error) {
	res, err := db.getEvents(revision, start, count)
	return res, parseError(err)
}

func (db *DataBase) ImportZone(userId ObjectId, z ZoneImport) error {
	owner, err := db.getZoneOwner(z.Name)
	if err != nil {
		return parseError(err)
	}
	zoneId, err := db.getZoneId(z.Name)
	if err != nil {
		return parseError(err)
	}
	if owner != userId {
		return ErrInvalid
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := deleteZoneData(t, zoneId); err != nil {
			return err
		}
		for label, location := range z.Entries {
			locationId, err := addResource(t, EmptyObjectId)
			if err != nil {
				return err
			}
			if err = addLocation(t, zoneId, locationId, NewLocation{
				ZoneName: z.Name,
				Location: label,
				Enabled:  true,
			}); err != nil {
				return err
			}
			for rtype, rset := range location {
				recordId, err := addResource(t, EmptyObjectId)
				if err != nil {
					return err
				}
				if err = addRecordSet(t, locationId, recordId, NewRecordSet{
					ZoneName: z.Name,
					Location: "",
					Type:     rtype,
					Value:    rset,
					Enabled:  true,
				}); err != nil {
					return err
				}
			}
		}
		if err = updateSerial(t, zoneId); err != nil {
			return err
		}
		if _, err = addEvent(t, zoneId, ImportZone, z); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) AddZone(userId ObjectId, z NewZone) (ObjectId, error) {
	owner, err := db.getZoneOwner(z.Name)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if owner == EmptyObjectId {
		var zoneId ObjectId
		err := db.withTransaction(func(t *sql.Tx) error {
			var err error
			zoneId, err = addResource(t, EmptyObjectId)
			if err != nil {
				return err
			}
			err = addZone(t, userId, zoneId, z)
			if err != nil {
				return err
			}
			if err = setSOA(t, zoneId, z.SOA); err != nil {
				return err
			}
			rootLocation := NewLocation{
				ZoneName: z.Name,
				Location: "@",
				Enabled:  true,
			}
			locationId, err := addResource(t, EmptyObjectId)
			if err != nil {
				return err
			}
			err = addLocation(t, zoneId, locationId, rootLocation)
			if err != nil {
				return err
			}
			nsRecord := NewRecordSet{Type: "ns", Value: &z.NS, Enabled: true}
			nsId, err := addResource(t, EmptyObjectId)
			if err != nil {
				return err
			}
			err = addRecordSet(t, locationId, nsId, nsRecord)
			if err != nil {
				return err
			}
			if err = setZoneKeys(t, zoneId, z.Keys); err != nil {
				return err
			}
			if _, err = addEvent(t, zoneId, AddZone, z); err != nil {
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
	res, err := db.getZones(userId, start, count, q, ascendingOrder)
	return res, parseError(err)
}

func (db *DataBase) GetZone(userId ObjectId, zoneName string) (Zone, error) {
	zoneId, err := db.getZoneId(zoneName)
	if err != nil {
		return Zone{}, parseError(err)
	}
	if !db.isAuthorized(userId, zoneName) {
		return Zone{}, ErrUnauthorized
	}
	z, err := db.getZone(zoneId)
	return z, parseError(err)
}

func (db *DataBase) UpdateZone(userId ObjectId, z ZoneUpdate) error {
	zoneId, err := db.getZoneId(z.Name)
	if err != nil {
		return parseError(err)
	}
	if !db.isAuthorized(userId, z.Name) {
		return ErrUnauthorized
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := updateZone(t, zoneId, z); err != nil {
			return err
		}
		if err := setSOA(t, zoneId, z.SOA); err != nil {
			return err
		}
		if _, err := addEvent(t, zoneId, UpdateZone, z); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) DeleteZone(userId ObjectId, z ZoneDelete) error {
	zoneId, err := db.getZoneId(z.Name)
	if err != nil {
		return parseError(err)
	}
	if !db.isAuthorized(userId, z.Name) {
		return ErrUnauthorized
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := deleteZone(t, zoneId); err != nil {
			return err
		}
		if _, err := addEvent(t, zoneId, DeleteZone, z.Name); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) AddLocation(userId ObjectId, l NewLocation) (ObjectId, error) {
	zoneId, err := db.getZoneId(l.ZoneName)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if !db.isAuthorized(userId, l.ZoneName) {
		return EmptyObjectId, parseError(err)
	}
	var locationId ObjectId
	err = db.withTransaction(func(t *sql.Tx) error {
		var err error
		locationId, err = addResource(t, EmptyObjectId)
		if err != nil {
			return err
		}
		err = addLocation(t, zoneId, locationId, l)
		if err != nil {
			return err
		}
		if _, err := addEvent(t, zoneId, AddLocation, l); err != nil {
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
	if !db.isAuthorized(userId, zoneName) {
		return List{}, parseError(err)
	}
	res, err := db.getLocations(zoneId, start, count, q, ascendingOrder)
	return res, parseError(err)
}

func (db *DataBase) GetLocation(userId ObjectId, zoneName string, location string) (Location, error) {
	_, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return Location{}, parseError(err)
	}
	if !db.isAuthorized(userId, zoneName) {
		return Location{}, parseError(err)
	}
	l, err := db.getLocation(locationId)
	return l, parseError(err)
}

func (db *DataBase) UpdateLocation(userId ObjectId, l LocationUpdate) error {
	zoneId, locationId, err := db.getLocationId(l.ZoneName, l.Location)
	if err != nil {
		return parseError(err)
	}
	if !db.isAuthorized(userId, l.ZoneName) {
		return parseError(err)
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := updateLocation(t, locationId, l); err != nil {
			return err
		}
		if _, err := addEvent(t, zoneId, UpdateLocation, l); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) DeleteLocation(userId ObjectId, l LocationDelete) error {
	zoneId, locationId, err := db.getLocationId(l.ZoneName, l.Location)
	if err != nil {
		return parseError(err)
	}
	if !db.isAuthorized(userId, l.ZoneName) {
		return parseError(err)
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := deleteLocation(t, locationId); err != nil {
			return err
		}
		if _, err := addEvent(t, zoneId, DeleteLocation, l); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) AddRecordSet(userId ObjectId, r NewRecordSet) (ObjectId, error) {
	zoneId, locationId, err := db.getLocationId(r.ZoneName, r.Location)
	if err != nil {
		return EmptyObjectId, parseError(err)
	}
	if !db.isAuthorized(userId, r.ZoneName) {
		return EmptyObjectId, parseError(err)
	}
	var recordId ObjectId
	err = db.withTransaction(func(t *sql.Tx) error {
		var err error
		if recordId, err = addResource(t, EmptyObjectId); err != nil {
			return err
		}
		if err = addRecordSet(t, locationId, recordId, r); err != nil {
			return err
		}
		if _, err = addEvent(t, zoneId, AddRecord, r); err != nil {
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
	_, locationId, err := db.getLocationId(zoneName, location)
	if err != nil {
		return List{}, parseError(err)
	}
	if !db.isAuthorized(userId, zoneName) {
		return List{}, parseError(err)
	}
	res, err := db.getRecordSets(locationId)
	return res, parseError(err)
}

func (db *DataBase) GetRecordSet(userId ObjectId, zoneName string, location string, recordType string) (RecordSet, error) {
	_, _, recordId, err := db.getRecordId(zoneName, location, recordType)
	if err != nil {
		return RecordSet{}, parseError(err)
	}
	if !db.isAuthorized(userId, zoneName) {
		return RecordSet{}, parseError(err)
	}
	r, err := db.getRecordSet(recordId, recordType)
	return r, parseError(err)
}

func (db *DataBase) UpdateRecordSet(userId ObjectId, r RecordSetUpdate) error {
	zoneId, _, recordId, err := db.getRecordId(r.ZoneName, r.Location, r.Type)
	if err != nil {
		return parseError(err)
	}
	if !db.isAuthorized(userId, r.ZoneName) {
		return parseError(err)
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := updateRecordSet(t, recordId, r); err != nil {
			return err
		}
		if _, err := addEvent(t, zoneId, UpdateRecord, r); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) DeleteRecordSet(userId ObjectId, r RecordSetDelete) error {
	zoneId, _, recordId, err := db.getRecordId(r.ZoneName, r.Location, r.Type)
	if err != nil {
		return parseError(err)
	}
	if !db.isAuthorized(userId, r.ZoneName) {
		return parseError(err)
	}
	err = db.withTransaction(func(t *sql.Tx) error {
		if err := deleteRecordSet(t, recordId); err != nil {
			return parseError(err)
		}
		if _, err := addEvent(t, zoneId, DeleteRecord, r); err != nil {
			return err
		}
		return nil
	})
	return parseError(err)
}

func (db *DataBase) GetVerification(userId ObjectId, verificationType VerificationType) (string, error) {
	code, err := db.getVerification(userId, verificationType)
	if err != nil {
		return "", parseError(err)
	}
	return code, nil
}
