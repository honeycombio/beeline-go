package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/davecgh/go-spew/spew"
	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnysqlx"
	libhoney "github.com/honeycombio/libhoney-go"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"
const honeyEventContextKey = "honeycombEventContextKey"

func main() {
	honeycomb.NewHoneycombInstrumenter(writekey, "")
	odb, err := sqlx.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	db := hnysqlx.WrapDB(libhoney.NewBuilder(), odb)
	if err != nil {
		fmt.Println("connection err")
		spew.Dump(err)
	}
	ev := libhoney.NewEvent()
	ev.AddField("traceId", "trace-me")
	ctx := context.WithValue(context.Background(), honeyEventContextKey, ev)
	db.MustExecContext(ctx, "insert into flavors (flavor) values ('rose')")
	fv := "rose"
	rows, err := db.QueryxContext(ctx, "SELECT id FROM flavors WHERE flavor=?", fv)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s is %d\n", name, fv)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	//
}

// if err != nil {
// 	// fmt.Printf("whee got err %v\n", err)
// } else {
// 	// lii, _ := res.LastInsertId()
// 	// fmt.Printf("res last insert id was %d\n", lii)
// }
