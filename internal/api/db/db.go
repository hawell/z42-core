package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
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

func (db *DataBase) Close() {
	db.db.Close()
}

func (db *DataBase) Clear() error {
	_, err := db.db.Exec("DELETE FROM User")
	return err
}

func (db *DataBase) AddUser(name string) (int64, error) {
	res, err := db.db.Exec("INSERT INTO User(Name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DataBase) GetUser(name string) (User, error) {
	res := db.db.QueryRow("SELECT Id, Name FROM User WHERE Name = ?", name)
	var u User
	err := res.Scan(&u.Id, &u.Name)
	return u, err
}

func (db *DataBase) DeleteUser(name string) (int64, error) {
	res, err := db.db.Exec("DELETE FROM User WHERE Name = ?", name)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) AddZone(user string, zone string, enabled bool) (int64, error) {
	u, err := db.GetUser(user)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("INSERT INTO Zone(Name, Enabled, User_Id) VALUES (?, ?, ?)", zone, enabled, u.Id)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DataBase) GetZones(user string, start int, count int) ([]Zone, error) {
	u, err := db.GetUser(user)
	if err != nil {
		return nil, err
	}
	rows, err := db.db.Query("SELECT Id, Name, Enabled FROM Zone WHERE User_Id = ? ORDER BY Name LIMIT ?, ?", u.Id, start, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []Zone
	for rows.Next() {
		var z Zone
		err := rows.Scan(&z.Id, &z.Name, &z.Enabled)
		if err != nil {
			return nil, err
		}
		res = append(res, z)
	}
	return res, nil
}

func (db *DataBase) GetZone(zone string) (Zone, error) {
	res := db.db.QueryRow("SELECT Id, Name, Enabled FROM Zone WHERE Name = ?", zone)
	var z Zone
	err := res.Scan(&z.Id, &z.Name, &z.Enabled)
	return z, err
}

func (db *DataBase) UpdateZone(zone string, enabled bool) (int64, error) {
	res, err := db.db.Exec("UPDATE Zone SET Enabled = ? WHERE Name = ?", enabled, zone)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteZone(zone string) (int64, error) {
	res, err := db.db.Exec("DELETE FROM Zone WHERE Name = ?", zone)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) AddLocation(zone string, location string, enabled bool) (int64, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("INSERT INTO Location(Name, Enabled, Zone_Id) VALUES (?, ?, ?)", location, enabled, z.Id)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DataBase) GetLocations(zone string, start int, count int) ([]Location, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		return nil, err
	}
	rows, err := db.db.Query("SELECT Id, Name, Enabled FROM Location WHERE Zone_Id = ? ORDER BY Name LIMIT ?, ?", z.Id, start, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []Location
	for rows.Next() {
		var l Location
		err := rows.Scan(&l.Id, &l.Name, &l.Enabled)
		if err != nil {
			return nil, err
		}
		res = append(res, l)
	}
	return res, nil
}

func (db *DataBase) GetLocation(zone string, location string) (Location, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		return Location{}, err
	}
	res := db.db.QueryRow("SELECT Id, Name, Enabled FROM Location WHERE Zone_Id = ? AND Name = ?", z.Id, location)
	var l Location
	err = res.Scan(&l.Id, &l.Name, &l.Enabled)
	return l, err
}

func (db *DataBase) UpdateLocation(zone string, location string, enabled bool) (int64, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("UPDATE Location SET Enabled = ? WHERE Zone_Id = ? AND Name = ?", enabled, z.Id, location)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteLocation(zone string, location string) (int64, error) {
	z, err := db.GetZone(zone)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("DELETE FROM Location WHERE Zone_Id = ? AND Name = ?", z.Id, location)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) AddRecordSet(zone string, location string, rtype string, value string, enabled bool) (int64, error) {
	l, err := db.GetLocation(zone, location)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("INSERT INTO RecordSet(Location_Id, Type, Value, Enabled) VALUES (?, ?, ?, ?)", l.Id, rtype, value, enabled)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DataBase) GetRecordSets(zone string, location string) ([]RecordSet, error) {
	l, err := db.GetLocation(zone, location)
	if err != nil {
		return nil, err
	}
	rows, err := db.db.Query("SELECT Id, Type, Value, Enabled FROM RecordSet WHERE Location_Id = ?", l.Id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []RecordSet
	for rows.Next() {
		var rset RecordSet
		err := rows.Scan(&rset.Id, &rset.Type, &rset.Value, &rset.Enabled)
		if err != nil {
			return nil, err
		}
		res = append(res, rset)
	}
	return res, nil
}

func (db *DataBase) GetRecordSet(zone string, location string, rtype string) (RecordSet, error) {
	l, err := db.GetLocation(zone, location)
	if err != nil {
		return RecordSet{}, err
	}
	row := db.db.QueryRow("SELECT Id, Type, Value, Enabled FROM RecordSet WHERE Location_Id = ? AND Type = ?", l.Id, rtype)
	var r RecordSet
	err = row.Scan(&r.Id, &r.Type, &r.Value, &r.Enabled)
	return r, err
}

func (db *DataBase) UpdateRecordSet(zone string, location string, rtype string, value string, enabled bool) (int64, error) {
	l, err := db.GetLocation(zone , location)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("UPDATE RecordSet SET Value = ?, Enabled = ?  WHERE Location_Id = ? AND Type = ?", value, enabled, l.Id, rtype)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (db *DataBase) DeleteRecordSet(zone string, location string, rtype string) (int64, error){
	l, err := db.GetLocation(zone, location)
	if err != nil {
		return 0, err
	}
	res, err := db.db.Exec("DELETE FROM RecordSet WHERE Location_Id = ? AND Type = ?", l.Id, rtype)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
