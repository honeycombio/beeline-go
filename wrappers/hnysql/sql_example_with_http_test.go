package hnysql_test

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	honeycomb "github.com/honeycombio/honeycomb-go-magic"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnynethttp"
	"github.com/honeycombio/honeycomb-go-magic/wrappers/hnysql"
	libhoney "github.com/honeycombio/libhoney-go"
)

func main() {
	// Initialize honeycomb. The only required field is WriteKey.
	honeycomb.Init(honeycomb.Config{
		WriteKey: "abcabc123123",
		Dataset:  "http+sql",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		STDOUT: true,
	})

	// open a regular sqlx connection and wrap it
	odb, err := sql.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	if err != nil {
		fmt.Printf("connection err: %s\n", err)
		return
	}
	db := hnysql.WrapDB(libhoney.NewBuilder(), odb)

	// hand it to the app for use in the handler
	a := &app{}
	a.db = db

	globalmux := http.NewServeMux()
	globalmux.HandleFunc("/hello/", a.hello)

	// wrap the globalmux with the honeycomb middleware to send one event per
	// request
	http.ListenAndServe(":8080", hnynethttp.WrapMuxHandler(globalmux))
}

type app struct {
	db *hnysql.DB
}

func (a *app) hello(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// get all flavors from the DB
	rows, err := a.db.QueryContext(ctx, "SELECT flavor FROM flavors GROUP BY flavor")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// collect them
	flavors := []string{}
	for rows.Next() {
		var flavor string
		if err := rows.Scan(&flavor); err != nil {
			log.Fatal(err)
		}
		flavors = append(flavors, flavor)
	}
	// add some custom fields to the Honeycomb event
	honeycomb.AddField(ctx, "flavors_count", len(flavors))
	honeycomb.AddField(ctx, "flavors", flavors)

	// send our response to the caller
	io.WriteString(w,
		fmt.Sprintf("Hello world! Choose one of these %d flavors:\n", len(flavors)))
	for _, flavor := range flavors {
		io.WriteString(w, flavor+"\n")
	}
}

// A call to the hello endpoint produces two events, one for the HTTP request
// and one for the SQL call. They look like this:
//
// {
//   "data": {
//     "duration_ms": 2.735045,
//     "flavors": ["chocolate","mint","rose","vanilla"],
//     "flavors_count": 4,
//     "meta.localhostname": "cobbler",
//     "meta.type": "http request",
//     "mux.handler.name": "main.(*app).(main.hello)-fm",
//     "mux.handler.pattern": "/hello/",
//     "mux.handler.type": "http.HandlerFunc",
//     "request.content_length": 0,
//     "request.header.user_agent": "curl/7.54.0",
//     "request.host": "",
//     "request.method": "GET",
//     "request.path": "/hello/foo",
//     "request.proto": "HTTP/1.1",
//     "request.remote_addr": "[::1]:52317",
//     "response.status_code": 200
//     "trace.trace_id": "a0eca504-a652-46da-b968-07dd076e2d0c",
//   },
//   "time": "2018-04-06T22:42:18.449138413-07:00"
// }
// {
//   "data": {
//     "sql.call": "QueryContext",
//     "duration_ms": 1.75518,
//     "meta.localhostname": "cobbler",
//     "meta.type": "sql",
//     "sql.open_conns": 0,
//     "sql.query": "SELECT flavor FROM flavors GROUP BY flavor"
//     "trace.trace_id": "a0eca504-a652-46da-b968-07dd076e2d0c",
//   },
//   "time": "2018-04-06T22:42:18.449620729-07:00"
// }

// An example that sends both http and sql events. Run and visit the '/hello/'
// endpoint to create an event.
func Example() {} // This tells godocs that this file is an example.
