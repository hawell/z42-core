package database

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
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
)

type DataBase struct {
	db *sql.DB
}

func Connect(connectionString string) (*DataBase, error) {
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		return nil, err
	}
	return &DataBase{db}, nil
}

func (db *DataBase) Close() error {
	return db.db.Close()
}

func (db *DataBase) Clear() error {
	_, err := db.db.Exec("DELETE FROM User")
	return err
}

func (db *DataBase) AddUser(u User) (ObjectId, error) {
	hash, err := HashPassword(u.Password)
	if err != nil {
		return emptyObjectId, err
	}
	id := NewObjectId()
	_, err = db.db.Exec("INSERT INTO User(Id, Email, Password, Status) VALUES (?, ?, ?, ?)", id, u.Email, hash, u.Status)
	if err != nil {
		return emptyObjectId, parseError(err)
	}
	return id, nil
}

func (db *DataBase) AddVerification(name string, verificationType string) (string, error) {
	u, err := db.GetUser(name)
	if err != nil {
		if err == ErrNotFound {
			return "", ErrInvalid
		}
		return "", err
	}
	code := randomString(50)
	_, err = db.db.Exec("INSERT INTO Verification(Code, Type, User_Id) VALUES (?, ?, ?)", code, verificationType, u.Id)
	if err != nil {
		return "", err
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
		return 0, err
	}
	if rows == 0 {
		return 0, ErrNotFound
	}
	return rows, err
}

func (db *DataBase) getZoneOwner(zone string) (ObjectId, error) {
	res := db.db.QueryRow("SELECT User_Id FROM Zone WHERE Name = ?", zone)
	var userId ObjectId
	err := res.Scan(&userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return emptyObjectId, nil
		}
		return emptyObjectId, err
	}
	return userId, nil
}

func (db *DataBase) AddZone(user string, z Zone) (ObjectId, error) {
	u, err := db.GetUser(user)
	if err != nil {
		if err == ErrNotFound {
			return emptyObjectId, ErrInvalid
		}
		return emptyObjectId, err
	}
	owner, err := db.getZoneOwner(z.Name)
	if err != nil {
		return emptyObjectId, err
	}
	if owner == emptyObjectId {
		id := NewObjectId()
		_, err := db.db.Exec("INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (?, ?, ?, ?, ?, ?)", id, z.Name, z.CNameFlattening, z.Dnssec, z.Enabled, u.Id)
		if err != nil {
			return emptyObjectId, parseError(err)
		}
		return id, nil
	}
	if owner == u.Id {
		return emptyObjectId, ErrDuplicateEntry
	}
	return emptyObjectId, ErrInvalid
}

func (db *DataBase) GetZones(user string, start int, count int, q string) ([]string, error) {
	u, err := db.GetUser(user)
	if err != nil {
		if err == ErrNotFound {
			return nil, ErrInvalid
		}
		return nil, err
	}
	like := "%" + q + "%"
	rows, err := db.db.Query("SELECT Name FROM Zone WHERE User_Id = ? AND Name LIKE ? ORDER BY Name LIMIT ?, ?", u.Id, like, start, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []string{}
	for rows.Next() {
		var zone string
		err := rows.Scan(&zone)
		if err != nil {
			return nil, err
		}
		res = append(res, zone)
	}
	return res, nil
}

func (db *DataBase) GetZone(zone string) (Zone, error) {
	res := db.db.QueryRow("SELECT Id, Name, CNameFlattening, Dnssec, Enabled FROM Zone WHERE Name = ?", zone)
	var z Zone
	err := res.Scan(&z.Id, &z.Name, &z.CNameFlattening, &z.Dnssec, &z.Enabled)
	return z, parseError(err)
}

func (db *DataBase) UpdateZone(z Zone) (int64, error) {
	_, err := db.GetZone(z.Name)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("UPDATE Zone SET Dnssec = ?, CNameFlattening = ?, Enabled = ? WHERE Name = ?", z.Dnssec, z.CNameFlattening, z.Enabled, z.Name)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteZone(zone string) (int64, error) {
	res, err := db.db.Exec("DELETE FROM Zone WHERE Name = ?", zone)
	if err != nil {
		return 0, parseError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		return 0, ErrNotFound
	}
	return rows, err
}

func (db *DataBase) AddLocation(zone string, l Location) (ObjectId, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		if err == ErrNotFound {
			return emptyObjectId, ErrInvalid
		}
		return emptyObjectId, err
	}
	id := NewObjectId()
	_, err = db.db.Exec("INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (?, ?, ?, ?)", id, l.Name, l.Enabled, z.Id)
	if err != nil {
		return emptyObjectId, parseError(err)
	}
	return id, nil
}

func (db *DataBase) GetLocations(zone string, start int, count int, q string) ([]string, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		if err == ErrNotFound {
			return nil, ErrInvalid
		}
		return nil, err
	}
	like := "%" + q + "%"
	rows, err := db.db.Query("SELECT Name FROM Location WHERE Zone_Id = ? AND Name LIKE ? ORDER BY Name LIMIT ?, ?", z.Id, like, start, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []string{}
	for rows.Next() {
		var location string
		err := rows.Scan(&location)
		if err != nil {
			return nil, err
		}
		res = append(res, location)
	}
	return res, nil
}

func (db *DataBase) GetLocation(zone string, location string) (Location, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		if err == ErrNotFound {
			return Location{}, ErrInvalid
		}
		return Location{}, err
	}
	res := db.db.QueryRow("SELECT Id, Name, Enabled FROM Location WHERE Zone_Id = ? AND Name = ?", z.Id, location)
	var l Location
	err = res.Scan(&l.Id, &l.Name, &l.Enabled)
	return l, parseError(err)
}

func (db *DataBase) locationExists(zone string, location string) (bool, error) {
	res := db.db.QueryRow("select count(*) from Zone left join Location L on Zone.Id = L.Zone_Id where Zone.Name = ? and L.Name = ?", zone, location)
	var count int64
	err := res.Scan(&count)
	return count > 0, err
}

func (db *DataBase) UpdateLocation(zone string, l Location) (int64, error) {
	storedLocation, err := db.GetLocation(zone, l.Name)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("UPDATE Location SET Enabled = ? WHERE Id = ?", l.Enabled, storedLocation.Id)
	if err != nil {
		return 0, parseError(err)
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteLocation(zone string, location string) (int64, error) {
	storedLocation, err := db.GetLocation(zone, location)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("DELETE FROM Location WHERE Id = ?", storedLocation.Id)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		return 0, ErrNotFound
	}
	return rows, err
}

func (db *DataBase) AddRecordSet(zone string, location string, r RecordSet) (ObjectId, error) {
	if !rtypeValid(r.Type) {
		return emptyObjectId, ErrInvalid
	}
	l, err := db.GetLocation(zone, location)
	if err != nil {
		if err == ErrNotFound {
			return emptyObjectId, ErrInvalid
		}
		return emptyObjectId, err
	}
	id := NewObjectId()
	_, err = db.db.Exec("INSERT INTO RecordSet(Id, Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?, ?)", id, l.Id, r.Type, r.Value, r.Enabled)
	if err != nil {
		return emptyObjectId, parseError(err)
	}
	return id, nil
}

func (db *DataBase) GetRecordSets(zone string, location string) ([]string, error) {
	l, err := db.GetLocation(zone, location)
	if err != nil {
		if err == ErrNotFound {
			return nil, ErrInvalid
		}
		return nil, err
	}
	rows, err := db.db.Query("SELECT Type FROM RecordSet WHERE Location_Id = ?", l.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []string{}
	for rows.Next() {
		var rset string
		err := rows.Scan(&rset)
		if err != nil {
			return nil, err
		}
		res = append(res, rset)
	}
	return res, nil
}

func (db *DataBase) GetRecordSet(zone string, location string, rtype string) (RecordSet, error) {
	if !rtypeValid(rtype) {
		return RecordSet{}, ErrInvalid
	}
	l, err := db.GetLocation(zone, location)
	if err != nil {
		if err == ErrNotFound {
			return RecordSet{}, ErrInvalid
		}
		return RecordSet{}, err
	}
	row := db.db.QueryRow("SELECT Id, Type, Value, Enabled FROM RecordSet WHERE Location_Id = ? AND Type = ?", l.Id, rtype)
	var r RecordSet
	err = row.Scan(&r.Id, &r.Type, &r.Value, &r.Enabled)
	return r, parseError(err)
}

func (db *DataBase) UpdateRecordSet(zone string, location string, r RecordSet) (int64, error) {
	if !rtypeValid(r.Type) {
		return 0, ErrInvalid
	}
	storedRecordSet, err := db.GetRecordSet(zone, location, r.Type)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("UPDATE RecordSet SET Value = ?, Enabled = ?  WHERE Id = ?", r.Value, r.Enabled, storedRecordSet.Id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteRecordSet(zone string, location string, rtype string) (int64, error) {
	if !rtypeValid(rtype) {
		return 0, ErrInvalid
	}
	storedRecordSet, err := db.GetRecordSet(zone, location, rtype)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("DELETE FROM RecordSet WHERE Id = ?", storedRecordSet.Id)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		return 0, ErrNotFound
	}
	return rows, err
}

func parseError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if mysqlErr.Number == 1062 {
			return ErrDuplicateEntry
		}
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func rtypeValid(rtype string) bool {
	if rtype == "" {
		return false
	}
	for _, t := range SupportedTypes {
		if rtype == t {
			return true
		}
	}
	return false
}
