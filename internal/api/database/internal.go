package database

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	jsoniter "github.com/json-iterator/go"
	"z42-core/internal/types"
)

func addUser(t *sql.Tx, userId ObjectId, u NewUser) error {
	_, err := t.Exec("INSERT INTO User(Id, Email, Password, Status) VALUES (?, ?, ?, ?)", userId, u.Email, u.Password, u.Status)
	return err
}

func (db *DataBase) deleteUser(name string) error {
	res, err := db.db.Exec("DELETE FROM User WHERE Email = ?", name)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (db *DataBase) getUser(name string) (User, error) {
	res := db.db.QueryRow("SELECT Id, Email, Password, Status FROM User WHERE Email = ?", name)
	var u User
	err := res.Scan(&u.Id, &u.Email, &u.Password, &u.Status)
	return u, err
}

func deleteUsers(t *sql.Tx) error {
	_, err := t.Exec("DELETE FROM User")
	return err
}

func (db *DataBase) getVerification(userId ObjectId, verificationType VerificationType) (string, error) {
	res := db.db.QueryRow("SELECT Code FROM Verification WHERE User_Id = ? AND Type = ?", userId, verificationType)
	var code string
	err := res.Scan(&code)
	return code, err
}

func setVerification(t *sql.Tx, userId ObjectId, v Verification) error {
	_, err := t.Exec("REPLACE INTO Verification(Code, Type, User_Id) VALUES (?, ?, ?)", v.Code, v.Type, userId)
	return err
}

func deleteVerifications(t *sql.Tx) error {
	_, err := t.Exec("DELETE FROM Verification")
	return err
}

func setUserStatus(t *sql.Tx, userId ObjectId, status UserStatus) error {
	_, err := t.Exec("UPDATE User SET Status = ? WHERE Id = ?", status, userId)
	return err
}

func setUserPassword(t *sql.Tx, userId ObjectId, password string) error {
	_, err := t.Exec("UPDATE User SET Password = ? WHERE Id = ?", password, userId)
	return err
}

func setSOA(t *sql.Tx, zoneId ObjectId, soa types.SOA_RRSet) error {
	_, err := t.Exec("REPLACE INTO SOA VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", soa.TtlValue, soa.Ns, soa.MBox, soa.Refresh, soa.Retry, soa.Expire, soa.MinTtl, soa.Serial, zoneId)
	return err
}

func addResource(t *sql.Tx, resourceId ObjectId) (ObjectId, error) {
	if resourceId == EmptyObjectId {
		resourceId = NewObjectId()
	}
	if _, err := t.Exec("INSERT INTO Resource(Id) VALUES (?)", resourceId); err != nil {
		return EmptyObjectId, err
	}
	return resourceId, nil
}

func deleteResources(t *sql.Tx) error {
	_, err := t.Exec("DELETE FROM Resource")
	return err
}

func addZone(t *sql.Tx, userId ObjectId, resourceId ObjectId, z NewZone) error {
	if _, err := t.Exec("INSERT INTO Zone(Resource_Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (?, ?, ?, ?, ?, ?)", resourceId, z.Name, z.CNameFlattening, z.Dnssec, z.Enabled, userId); err != nil {
		return err
	}
	return nil
}

func updateZone(t *sql.Tx, zoneId ObjectId, z ZoneUpdate) error {
	_, err := t.Exec("UPDATE Zone SET Name = ?, Dnssec = ?, CNameFlattening = ?, Enabled = ? WHERE Resource_Id = ?", z.Name, z.Dnssec, z.CNameFlattening, z.Enabled, zoneId)
	return err
}

func deleteZone(t *sql.Tx, zoneId ObjectId) error {
	_, err := t.Exec(
		`
			WITH LocationIds AS (SELECT Resource_Id AS Id FROM Location WHERE Zone_Id = ?),
			RecordIds AS (SELECT Resource_Id AS Id FROM RecordSet WHERE Location_Id IN (SELECT * FROM LocationIds)),
			Ids AS (SELECT ? AS Id UNION SELECT Id FROM RecordIds UNION SELECT Id FROM LocationIds)
			DELETE FROM Resource WHERE Id IN (SELECT Id FROM Ids) 
		`, zoneId, zoneId)
	return err
}

func addLocation(t *sql.Tx, zoneId ObjectId, resourceId ObjectId, l NewLocation) error {
	if _, err := t.Exec("INSERT INTO Location(Resource_Id, Name, Enabled, Zone_Id) VALUES (?, ?, ?, ?)", resourceId, l.Location, l.Enabled, zoneId); err != nil {
		return err
	}
	return nil
}

func updateLocation(t *sql.Tx, locationId ObjectId, l LocationUpdate) error {
	_, err := t.Exec("UPDATE Location SET Name = ?, Enabled = ? WHERE Resource_Id = ?", l.Location, l.Enabled, locationId)
	return err
}

func deleteLocation(t *sql.Tx, locationId ObjectId) error {
	_, err := t.Exec("WITH IDS AS (SELECT ? as Id UNION SELECT Resource_Id FROM RecordSet WHERE Location_Id = ?) DELETE FROM Resource WHERE Id IN (SELECT * FROM IDS)", locationId, locationId)
	return err
}

func deleteZoneData(t *sql.Tx, zoneId ObjectId) error {
	_, err := t.Exec("DELETE FROM Resource WHERE Id IN (SELECT Resource_Id FROM Location WHERE Zone_Id = ?)", zoneId)
	return err
}

func updateSerial(t *sql.Tx, zoneId ObjectId) error {
	return nil
}

func addRecordSet(t *sql.Tx, locationId ObjectId, resourceId ObjectId, r NewRecordSet) error {
	value, err := jsoniter.Marshal(r.Value)
	if err != nil {
		return err
	}
	if _, err := t.Exec("INSERT INTO RecordSet(Resource_Id, Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?, ?)", resourceId, locationId, r.Type, value, r.Enabled); err != nil {
		return err
	}
	return nil
}

func updateRecordSet(t *sql.Tx, recordId ObjectId, r RecordSetUpdate) error {
	value, err := jsoniter.Marshal(r.Value)
	if err != nil {
		return err
	}
	_, err = t.Exec("UPDATE RecordSet SET Value = ?, Enabled = ?  WHERE Resource_Id = ?", value, r.Enabled, recordId)
	return err
}

func deleteRecordSet(t *sql.Tx, recordId ObjectId) error {
	_, err := t.Exec("DELETE FROM Resource WHERE Id = ?", recordId)
	return err
}

func setZoneKeys(t *sql.Tx, zoneId ObjectId, zoneKeys types.ZoneKeys) error {
	_, err := t.Exec("INSERT INTO `Keys`(KSK_Private, KSK_Public, ZSK_Private, ZSK_Public, DS, Zone_Id) VALUES (?, ?, ?, ?, ?, ?)", zoneKeys.KSKPrivate, zoneKeys.KSKPublic, zoneKeys.ZSKPrivate, zoneKeys.ZSKPublic, zoneKeys.DS, zoneId)
	return err
}

func addEvent(t *sql.Tx, zoneId ObjectId, eventType EventType, value interface{}) (int64, error) {
	jsonValue, err := jsoniter.Marshal(value)
	if err != nil {
		return 0, err
	}
	res, err := t.Exec("INSERT INTO Events(ZoneId, Type, Value) VALUES (?, ?, ?)", zoneId, eventType, jsonValue)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func deleteEvents(t *sql.Tx) error {
	if _, err := t.Exec("DELETE FROM Events"); err != nil {
		return err
	}
	_, err := t.Exec("ALTER TABLE Events AUTO_INCREMENT = 1")
	return err
}

func (db *DataBase) getZoneId(zoneName string) (ObjectId, error) {
	var zoneId ObjectId
	res := db.db.QueryRow("SELECT Resource_Id FROM Zone WHERE Name = ?", zoneName)
	err := res.Scan(&zoneId)
	if err != nil {
		return EmptyObjectId, err
	}
	return zoneId, nil
}

func (db *DataBase) getZones(userId ObjectId, start int, count int, q string, ascendingOrder bool) (List, error) {
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
		return List{}, err
	}
	defer func() { _ = rows.Close() }()
	res := List{Items: []ListItem{}}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return List{}, err
		}
		res.Items = append(res.Items, item)
	}
	row := db.db.QueryRow("SELECT count(*) FROM Zone WHERE User_Id = ? AND Name LIKE ?", userId, like)
	if err := row.Scan(&res.Total); err != nil {
		return List{}, err
	}
	return res, nil
}

func (db *DataBase) getZone(zoneId ObjectId) (Zone, error) {
	res := db.db.QueryRow("SELECT Resource_Id, Name, CNameFlattening, Dnssec, Enabled, TTL, NS, MBox, Refresh, Retry, Expire, MinTTL, Serial, DS FROM Zone LEFT JOIN SOA ON Zone.Resource_Id = SOA.Zone_Id  LEFT JOIN `Keys` K ON Zone.Resource_Id = K.Zone_Id WHERE Zone.Resource_Id = ?", zoneId)
	var (
		z Zone
	)
	err := res.Scan(&z.Id, &z.Name, &z.CNameFlattening, &z.Dnssec, &z.Enabled, &z.SOA.TtlValue, &z.SOA.Ns, &z.SOA.MBox, &z.SOA.Refresh, &z.SOA.Retry, &z.SOA.Expire, &z.SOA.MinTtl, &z.SOA.Serial, &z.DS)
	return z, err
}

func (db *DataBase) getLocations(zoneId ObjectId, start int, count int, q string, ascendingOrder bool) (List, error) {
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
		return List{}, err
	}
	defer func() { _ = rows.Close() }()
	res := List{Items: []ListItem{}}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return List{}, err
		}
		res.Items = append(res.Items, item)
	}
	row := db.db.QueryRow("SELECT count(*) FROM Location WHERE Zone_Id = ? AND Name LIKE ?", zoneId, like)
	if err := row.Scan(&res.Total); err != nil {
		return List{}, err
	}
	return res, nil
}

func (db *DataBase) getLocation(locationId ObjectId) (Location, error) {
	res := db.db.QueryRow("SELECT Resource_Id, Name, Enabled FROM Location WHERE Resource_Id = ?", locationId)
	var l Location
	err := res.Scan(&l.Id, &l.Name, &l.Enabled)
	return l, err
}

func (db *DataBase) getLocationId(zoneName string, location string) (ObjectId, ObjectId, error) {
	res := db.db.QueryRow("SELECT Zone.Resource_Id, L.Resource_Id FROM Zone LEFT JOIN Location L on Zone.Resource_Id = L.Zone_Id WHERE Zone.Name = ? AND L.Name = ?", zoneName, location)
	var (
		zoneId     ObjectId
		locationId ObjectId
	)
	err := res.Scan(&zoneId, &locationId)
	if err != nil {
		return EmptyObjectId, EmptyObjectId, err
	}
	return zoneId, locationId, nil
}

func (db *DataBase) getRecordId(zoneName string, location string, recordType string) (ObjectId, ObjectId, ObjectId, error) {
	res := db.db.QueryRow("SELECT Zone.Resource_Id, L.Resource_Id, RS.Resource_Id FROM Zone LEFT JOIN Location L on Zone.Resource_Id = L.Zone_Id LEFT JOIN RecordSet RS on L.Resource_Id = RS.Location_Id WHERE Zone.Name = ? AND L.Name = ? AND RS.Type = ?", zoneName, location, recordType)
	var (
		zoneId     ObjectId
		locationId ObjectId
		recordId   ObjectId
	)
	err := res.Scan(&zoneId, &locationId, &recordId)
	if err != nil {
		return EmptyObjectId, EmptyObjectId, EmptyObjectId, err
	}
	return zoneId, locationId, recordId, nil
}

func (db *DataBase) getRecordSets(locationId ObjectId) (List, error) {
	rows, err := db.db.Query("SELECT Type, Enabled FROM RecordSet WHERE Location_Id = ? ORDER BY Type", locationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmptyList(), nil
		}
		return List{}, err
	}
	defer func() { _ = rows.Close() }()
	res := List{Items: []ListItem{}}
	for rows.Next() {
		var item ListItem
		err := rows.Scan(&item.Id, &item.Enabled)
		if err != nil {
			return List{}, err
		}
		res.Items = append(res.Items, item)
	}
	row := db.db.QueryRow("SELECT count(*) FROM RecordSet WHERE Location_Id = ?", locationId)
	if err := row.Scan(&res.Total); err != nil {
		return List{}, err
	}
	return res, nil
}

func (db *DataBase) getRecordSet(recordId ObjectId, recordType string) (RecordSet, error) {
	row := db.db.QueryRow("SELECT Resource_Id, Type, Value, Enabled FROM RecordSet WHERE Resource_Id = ?", recordId)
	var (
		r     RecordSet
		value string
	)
	err := row.Scan(&r.Id, &r.Type, &value, &r.Enabled)
	rrset := types.TypeStrToRRSet(recordType)
	err = jsoniter.Unmarshal([]byte(value), &rrset)
	if err != nil {
		return RecordSet{}, err
	}
	r.Value = rrset
	return r, err
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

type actionFunction func(t *sql.Tx, userId ObjectId) error

func (db *DataBase) applyVerifiedAction(code string, verificationType VerificationType, action actionFunction) error {
	res := db.db.QueryRow("select U.Id, V.Type from Verification V left join User U on U.Id = V.User_Id WHERE Code = ?", code)
	var (
		userId     ObjectId
		storedType VerificationType
	)
	if err := res.Scan(&userId, &storedType); err != nil {
		return err
	}
	if storedType != verificationType {
		return errors.New("unknown verification type")
	}

	err := db.withTransaction(func(t *sql.Tx) error {
		if err := action(t, userId); err != nil {
			return err
		}
		if _, err := t.Exec("DELETE FROM Verification WHERE Code = ?", code); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (db *DataBase) getZoneOwner(zoneName string) (ObjectId, error) {
	res := db.db.QueryRow("SELECT User_Id FROM Zone WHERE Name = ?", zoneName)
	var userId ObjectId
	err := res.Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EmptyObjectId, nil
		}
		return EmptyObjectId, err
	}
	return userId, nil
}

func (db *DataBase) getEvents(revision int, start int, count int) ([]Event, error) {
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

func (db *DataBase) addAPIKey(userId ObjectId, zoneId ObjectId, apiKey APIKeyItem, hash string) error {
	_, err := db.db.Exec("INSERT INTO APIKeys(Name, Scope, Hash, Enabled, User_Id, Zone_Id) VALUES (?, ?, ?, ?, ?, ?)", apiKey.Name, apiKey.Scope, hash, apiKey.Enabled, userId, zoneId)
	return err
}

func (db *DataBase) getAPIKeys(userId ObjectId) ([]APIKeyItem, error) {
	rows, err := db.db.Query("SELECT T1.ZoneName, APIKeys.Name, APIKeys.Scope, APIKeys.Enabled FROM (SELECT User_Id, Z.Resource_Id AS Zone_Id, Z.Name AS ZoneName FROM Zone Z WHERE Z.User_Id = ?) T1 JOIN APIKeys ON T1.User_Id = APIKeys.User_Id AND T1.Zone_Id = APIKeys.Zone_Id ORDER BY APIKeys.Name", userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []APIKeyItem{}, nil
		}
		return []APIKeyItem{}, err
	}
	defer func() { _ = rows.Close() }()
	res := []APIKeyItem{}
	for rows.Next() {
		var item APIKeyItem
		err := rows.Scan(&item.ZoneName, &item.Name, &item.Scope, &item.Enabled)
		if err != nil {
			return []APIKeyItem{}, err
		}
		res = append(res, item)
	}
	return res, nil
}

func (db *DataBase) getAPIKey(userId ObjectId, name string) (APIKeyItem, error) {
	row := db.db.QueryRow("SELECT T1.ZoneName, APIKeys.Name, APIKeys.Scope, APIKeys.Enabled FROM (SELECT User_Id, Z.Resource_Id AS Zone_Id, Z.Name AS ZoneName FROM Zone Z WHERE Z.User_Id = ?) T1 JOIN APIKeys ON T1.User_Id = APIKeys.User_Id AND T1.Zone_Id = APIKeys.Zone_Id WHERE APIKeys.Name = ?", userId, name)
	var res APIKeyItem
	err := row.Scan(&res.ZoneName, &res.Name, &res.Scope, &res.Enabled)
	if err != nil {
		return APIKeyItem{}, err
	}
	return res, nil
}

func (db *DataBase) updateAPIKey(userId ObjectId, model APIKeyUpdate) error {
	_, err := db.db.Exec("UPDATE APIKeys SET Enabled = ?, Scope = ? WHERE User_Id = ? AND Name = ?", model.Enabled, model.Scope, userId, model.Name)
	return err
}

func (db *DataBase) deleteAPIKey(userId ObjectId, name string) error {
	_, err := db.db.Exec("DELETE FROM APIKeys WHERE  User_Id = ? AND Name = ?", userId, name)
	return err
}

func (db *DataBase) resourceExists(Id ObjectId) (bool, error) {
	row := db.db.QueryRow("SELECT COUNT(*) FROM Resource WHERE Id = ?", Id)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (db *DataBase) isAuthorized(userId ObjectId, zoneName string) bool {
	owner, err := db.getZoneOwner(zoneName)
	if err != nil || owner != userId {
		return false
	}
	return true
}
