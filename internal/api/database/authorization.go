package database

import (
    "database/sql"
)

func (db *DataBase) isZoneOwner(user string, zone string) error {
    var (
        userID  ObjectId
        ownerID ObjectId
    )
    {
        res := db.db.QueryRow("SELECT User.Id FROM User WHERE User.Email = ?", user)
        err := res.Scan(&userID)
        if err != nil {
            return err
        }
    }
    {
        res := db.db.QueryRow("SELECT User_Id FROM Zone WHERE Name = ?", zone)
        err := res.Scan(&ownerID)
        if err != nil {
            if err == sql.ErrNoRows {
                return ErrNotFound
            }
            return err
        }
    }
    if userID != ownerID {
        return ErrUnauthorized
    }
    return nil
}

func (db *DataBase) canGetZone(user string, zone string) error {
    return db.isZoneOwner(user, zone)
}

func (db *DataBase) canUpdateZone(user string, zone string) error {
    return db.isZoneOwner(user, zone)
}

func (db *DataBase) canDeleteZone(user string, zone string) error {
    return db.isZoneOwner(user, zone)
}

func (db *DataBase) canAddLocation(user string, zone string, location string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canGetLocations(user string, zone string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canGetLocation(user string, zone string, location string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canUpdateLocation(user string, zone string, location string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canDeleteLocation(user string, zone string, location string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canAddRecordSet(user string, zone string, location string, rtype string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canGetRecordSets(user string, zone string, location string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canGetRecordSet(user string, zone string, location string, rtype string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canUpdateRecordSet(user string, zone string, location string, rtype string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}

func (db *DataBase) canDeleteRecordSet(user string, zone string, location string, rtype string) error {
    err := db.isZoneOwner(user, zone)
    if err == ErrNotFound {
        return ErrInvalid
    }
    return err
}
