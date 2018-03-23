package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/davecgh/go-spew/spew"
	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	libhoney "github.com/honeycombio/libhoney-go"
)

const writekey = "cf80cea35c40752b299755ad23d2082e"

func main() {
	honeycomb.NewHoneycombInstrumenter(writekey, "")
	odb, err := sql.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	db := honeycomb.WrapDB(libhoney.NewBuilder(), odb)
	if err != nil {
		fmt.Println("connection err")
		spew.Dump(err)
	}
	db.Exec("insert into flavors (flavor) values ('rose')")
	// if err != nil {
	// 	// fmt.Printf("whee got err %v\n", err)
	// } else {
	// 	// lii, _ := res.LastInsertId()
	// 	// fmt.Printf("res last insert id was %d\n", lii)
	// }
}
