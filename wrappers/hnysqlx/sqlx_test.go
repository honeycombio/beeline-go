package hnysqlx_test

import (
	"context"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnysqlx"
)

func main() {
	// Initialize honeycomb. The only required field is WriteKey.
	honeycomb.Init(honeycomb.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sqlx",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	// open a regular sqlx connection
	odb, err := sqlx.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	if err != nil {
		fmt.Println("connection err")
	}

	// replace it with a wrapped hnysqlx.DB
	db := hnysqlx.WrapDB(odb)

	ctx := context.Background()
	db.MustExecContext(ctx, "insert into flavors (flavor) values ('rose')")
	fv := "rose"
	rows, err := db.QueryxContext(ctx, "SELECT id FROM flavors WHERE flavor=?", fv)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d is %s\n", id, fv)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}

func Example() {} // This tells godocs that this file is an example.
