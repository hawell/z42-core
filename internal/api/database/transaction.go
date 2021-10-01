package database

import (
	"database/sql"
)

type txFn func(tx *sql.Tx) error

func (db *DataBase) withTransaction(fn txFn) (err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			// a panic occurred, rollback and panic
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			// something went wrong, rollback
			_ = tx.Rollback()
		} else {
			// all good, commit
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}
