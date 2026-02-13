package postgres

import "database/sql"

func isNotFound(err error) bool {
	return err == sql.ErrNoRows
}
