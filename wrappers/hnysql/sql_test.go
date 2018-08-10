package hnysql_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnysql"
)

func Example() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sql",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})
	// and make sure we close to force flushing all pending events before shutdown
	defer beeline.Close()

	// open a regular sql.DB connection
	odb, err := sql.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	if err != nil {
		fmt.Printf("connection err: %s\n", err)
		return
	}

	// replace it with a wrapped hnysql.DB
	db := hnysql.WrapDB(odb)
	// and start up a trace to capture all the calls
	ctx := beeline.StartSpan(context.Background(), "start")
	defer beeline.EndSpan(ctx)

	// from here on, all SQL calls will emit events.

	db.ExecContext(ctx, "insert into flavors (flavor) values ('rose')")
	fv := "rose"
	rows, err := db.QueryContext(ctx, "SELECT id FROM flavors WHERE flavor=?", fv)
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
