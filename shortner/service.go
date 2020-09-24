package shortner

import (
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//Find retrives a ShortLink details for the provided link
func Find(link string) (ShortLink, error) {
	return findLink(link, false)
}

//FindByOriginalLink retrives a ShortLink by original link
func FindByOriginalLink(link string) (ShortLink, error) {
	return findLink(link, true)
}

//Persist saves the link data in database
func Persist(source ShortLink) (ShortLink, error) {
	db, err := Connect()
	if err != nil {
		return ShortLink{}, err
	}
	defer db.Close()
	statement, err := db.Prepare("insert into ShortLink(Shortened, original, expiry, created, hits) values(?,?,?,?,?)")

	if err != nil {
		return ShortLink{}, err
	}
	source.Created = time.Now()
	source.Expiry = 30
	source.Hits = 0
	statement.Exec(source.Shortened, source.Original, source.Expiry, source.Created, source.Hits)
	return source, nil
}

//UpdateCount updates hit count in database
func UpdateCount(link string) error {
	db, err := Connect()
	if err != nil {
		return err
	}
	defer db.Close()
	statement, err := db.Prepare("update ShortLink set hits = hits + 1 where Shortened =?")
	if err != nil {
		return errors.New("Unable to update hits in database")
	}
	statement.Exec(link)
	return nil
}

func createQuery(oldlink bool) string {
	if oldlink {
		return "select Shortened,Original,Expiry, Created,Hits from ShortLink where Original = ?"
	}
	return "select Shortened,Original,Expiry, Created,Hits from ShortLink where Shortened = ?"
}

func findLink(link string, oldlink bool) (ShortLink, error) {
	if link == "" {
		return ShortLink{}, errors.New("Invalid link")
	}
	db, err := Connect()
	if err != nil {
		return ShortLink{}, err
	}
	defer db.Close()
	result, err := db.Query(createQuery(oldlink), link)

	if err != nil {
		return ShortLink{}, err
	}
	object := ShortLink{}
	var shorten string
	var original string
	var expiry int
	var created time.Time
	var vhits int
	for result.Next() {
		result.Scan(&shorten, &original, &expiry, &created, &vhits)
		object.Shortened = shorten
		object.Original = original
		object.Expiry = expiry
		object.Created = created
		object.Hits = vhits
	}
	return object, nil
}
