package database

import (
	"database/sql"
	"github.com/hawell/z42/internal/types"
	jsoniter "github.com/json-iterator/go"
)

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
	_, err := t.Exec("DELETE FROM Resource WHERE Id = ?", zoneId)
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
	_, err := t.Exec("DELETE FROM Resource WHERE Id = ?", locationId)
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
