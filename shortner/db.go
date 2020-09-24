package shortner

import "database/sql"

//Connect makes connection to database
func Connect() (*sql.DB, error) {
	driver := "mysql"
	user := config.Db.UserName
	pass := config.Db.Password
	dbName := config.Db.Name
	url := user + ":" + pass + "@/" + dbName
	db, err := sql.Open(driver, url)
	return db, err
}
