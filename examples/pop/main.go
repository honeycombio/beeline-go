package main

import (
	"fmt"
	"os"

	"github.com/gobuffalo/pop"
	beeline "github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnypop"
	"github.com/honeycombio/beeline-go/wrappers/hnysqlx"
	"github.com/jmoiron/sqlx"
)

// this program expects there to be a `donut` database already running on
// localhost via mysql. The database should have one table `flavors` that has an
// id field, a flavor field, and an updated field. It should have a few
// records in there for good measure.

// the example counts how many rows are in the DB, adds a new one, then counts
// again.

type flavor struct {
	ID      int    `db:"id"`
	Flavor  string `db:"flavor"`
	Updated int    `db:"updated"`
}

func main() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sqlx",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})
	defer beeline.Close()

	// open up a raw sqlx connection to the database and wrap it
	odb, err := sqlx.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	db := hnysqlx.WrapDB(odb)
	// override the hnysqlx type for DB calls
	db.Builder.AddField("meta.type", "pop")

	// make a pop connection, then replace the default pop store with the
	// beeline-wrapped store implementation
	deets := &pop.ConnectionDetails{
		Dialect:  "mysql",
		Database: "donut",
		Host:     "localhost",
		User:     "root",
	}
	p, err := pop.NewConnection(deets)
	if err != nil {
		fmt.Println("err", err)
	}
	p.Store = &hnypop.DB{
		DB: db,
	}
	p.Open()

	var before = make([]*flavor, 0)
	var after = make([]*flavor, 0)

	p.Select("id", "flavor").All(&before)
	fmt.Printf("got back %d rows before adding cherry\n", len(before))

	newFlavor := &flavor{
		ID:     len(before) + 1,
		Flavor: "cherry",
	}
	err = p.Create(newFlavor)
	if err != nil {
		fmt.Printf("Error creating new flavor: %s\n", err)
		os.Exit(1)
	}

	p.Select("id", "flavor").All(&after)
	fmt.Printf("got back %d rows after adding cherry\n", len(after))
}
