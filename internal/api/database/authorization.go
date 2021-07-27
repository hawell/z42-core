package database

import (
	"database/sql"
)

type ACL struct {
	Read   bool
	List   bool
	Edit   bool
	Insert bool
	Delete bool
}

func (db *DataBase) getPrivileges(userId ObjectId, resourceId ObjectId) (ACL, error) {
	res := db.db.QueryRow("SELECT CanRead, CanList, CanEdit, CanInsert, CanDelete FROM ACL WHERE User_Id = ? AND Id = ?", userId, resourceId)
	var acl ACL
	err := res.Scan(&acl.Read, &acl.List, &acl.Edit, &acl.Insert, &acl.Delete)
	if err != nil {
		if err == sql.ErrNoRows {
			return ACL{}, ErrUnauthorized
		}
		return ACL{}, err
	}
	return acl, nil
}

func (db *DataBase) setPrivileges(userId ObjectId, resourceId ObjectId, acl ACL) error {
	_, err := db.db.Exec("REPLACE INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (?, ?, ?, ?, ?, ?, ?)", resourceId, acl.Read, acl.List, acl.Edit, acl.Insert, acl.Delete, userId)
	return err
}

func (db *DataBase) canRead(userId ObjectId, resourceId ObjectId) error {
	acl, err := db.getPrivileges(userId, resourceId)
	if err != nil {
		return err
	}
	if !acl.Read {
		return ErrUnauthorized
	}
	return nil
}

func (db *DataBase) canList(userId ObjectId, resourceId ObjectId) error {
	acl, err := db.getPrivileges(userId, resourceId)
	if err != nil {
		return err
	}
	if !acl.List {
		return ErrUnauthorized
	}
	return nil
}

func (db *DataBase) canEdit(userId ObjectId, resourceId ObjectId) error {
	acl, err := db.getPrivileges(userId, resourceId)
	if err != nil {
		return err
	}
	if !acl.Edit {
		return ErrUnauthorized
	}
	return nil
}

func (db *DataBase) canInsert(userId ObjectId, resourceId ObjectId) error {
	acl, err := db.getPrivileges(userId, resourceId)
	if err != nil {
		return err
	}
	if !acl.Insert {
		return ErrUnauthorized
	}
	return nil
}

func (db *DataBase) canDelete(userId ObjectId, resourceId ObjectId) error {
	acl, err := db.getPrivileges(userId, resourceId)
	if err != nil {
		return err
	}
	if !acl.Delete {
		return ErrUnauthorized
	}
	return nil
}

func (db *DataBase) canGetZone(userId ObjectId, zoneId ObjectId) error {
	return db.canRead(userId, zoneId)
}

func (db *DataBase) canUpdateZone(userId ObjectId, zoneId ObjectId) error {
	return db.canEdit(userId, zoneId)
}

func (db *DataBase) canDeleteZone(userId ObjectId, zoneId ObjectId) error {
	return db.canDelete(userId, zoneId)
}

func (db *DataBase) canAddLocation(userId ObjectId, zoneId ObjectId) error {
	return db.canInsert(userId, zoneId)
}

func (db *DataBase) canGetLocations(userId ObjectId, zoneId ObjectId) error {
	return db.canList(userId, zoneId)
}

func (db *DataBase) canGetLocation(userId ObjectId, zoneId ObjectId, locationId ObjectId) error {
	if err := db.canRead(userId, zoneId); err != nil {
		return err
	}
	return db.canRead(userId, locationId)
}

func (db *DataBase) canUpdateLocation(userId ObjectId, zoneId ObjectId, locationId ObjectId) error {
	if err := db.canEdit(userId, zoneId); err != nil {
		return err
	}
	return db.canEdit(userId, locationId)
}

func (db *DataBase) canDeleteLocation(userId ObjectId, zoneId ObjectId, locationId ObjectId) error {
	if err := db.canEdit(userId, zoneId); err != nil {
		return err
	}
	return db.canDelete(userId, locationId)
}

func (db *DataBase) canAddRecordSet(userId ObjectId, zoneId ObjectId, locationId ObjectId) error {
	if err := db.canEdit(userId, zoneId); err != nil {
		return err
	}
	return db.canInsert(userId, locationId)
}

func (db *DataBase) canGetRecordSets(userId ObjectId, zoneId ObjectId, locationId ObjectId) error {
	if err := db.canRead(userId, zoneId); err != nil {
		return err
	}
	return db.canList(userId, locationId)
}

func (db *DataBase) canGetRecordSet(userId ObjectId, zoneId ObjectId, locationId ObjectId, recordId ObjectId) error {
	if err := db.canRead(userId, zoneId); err != nil {
		return err
	}
	if err := db.canRead(userId, locationId); err != nil {
		return err
	}
	return db.canRead(userId, recordId)
}

func (db *DataBase) canUpdateRecordSet(userId ObjectId, zoneId ObjectId, locationId ObjectId, recordId ObjectId) error {
	if err := db.canEdit(userId, zoneId); err != nil {
		return err
	}
	if err := db.canEdit(userId, locationId); err != nil {
		return err
	}
	return db.canEdit(userId, recordId)
}

func (db *DataBase) canDeleteRecordSet(userId ObjectId, zoneId ObjectId, locationId ObjectId, recordId ObjectId) error {
	if err := db.canEdit(userId, zoneId); err != nil {
		return err
	}
	if err := db.canEdit(userId, locationId); err != nil {
		return err
	}
	return db.canDelete(userId, recordId)
}
