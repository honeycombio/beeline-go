package hnysqlx_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnysqlx"
)

func Example() {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sqlx",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		// NOTE: This should *only* be set to true in development environments.
		// Setting to true is Production environments can cause problems.
		STDOUT: true,
	})
	// and make sure we close to force flushing all pending events before shutdown
	defer beeline.Close()

	// open a regular sqlx connection
	odb, err := sqlx.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	if err != nil {
		fmt.Println("connection err")
	}

	// replace it with a wrapped hnysqlx.DB
	db := hnysqlx.WrapDB(odb)
	// and start up a trace for these statements to join
	ctx, span := beeline.StartSpan(context.Background(), "start")
	defer span.Send()

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

func TestDBBindNamed(t *testing.T) {
	// Initialize beeline. The only required field is WriteKey.
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sqlx",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		// NOTE: This should *only* be set to true in development environments.
		// Setting to true is Production environments can cause problems.
		STDOUT: true,
	})
	// and make sure we close to force flushing all pending events before shutdown
	defer beeline.Close()

	// open a regular sqlx connection
	odb, err := sqlx.Open("mysql", "root:@tcp(127.0.0.1)/donut")
	if err != nil {
		fmt.Println("connection err")
	}

	// replace it with a wrapped hnysqlx.DB
	db := hnysqlx.WrapDB(odb)
	// and start up a trace for these statements to join
	_, span := beeline.StartSpan(context.Background(), "start")
	defer span.Send()

	originalQ := `select :named`
	originalArgs := struct {
		Named string `db:"named"`
	}{"namedValue"}

	q, args, err := db.BindNamed(originalQ, originalArgs)
	if err != nil {
		log.Fatal(err)
	}

	expectedQ, expectedArgs, err := db.GetWrappedDB().BindNamed(originalQ, originalArgs)
	if err != nil {
		log.Fatal(err)
	}

	if q != expectedQ {
		t.Errorf("expected query: %s, got: %s", expectedQ, q)
	}

	var argsOK bool
	if len(expectedArgs) == len(args) {
		argsOK = true
		for i, arg := range args {
			if arg != expectedArgs[i] {
				argsOK = false
				break
			}
		}
	}
	if !argsOK {
		t.Errorf("expected args: %v, got: %v", expectedArgs, args)
	}
}

func TestSQLXMiddleware(t *testing.T) {
	beeline.Init(beeline.Config{
		WriteKey: "abcabc123123",
		Dataset:  "sql",
		// for demonstration, send the event to STDOUT intead of Honeycomb.
		// Remove the STDOUT setting when filling in a real write key.
		// NOTE: This should *only* be set to true in development environments.
		// Setting to true is Production environments can cause problems.
		STDOUT: true,
	})
	// and make sure we close to force flushing all pending events before shutdown
	defer beeline.Close()

	// Open a mock sql connection.
	odb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer odb.Close()
	sqlxodb := sqlx.NewDb(odb, "sqlmock")

	mock.ExpectExec("insert into flavors.+").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT id FROM flavors.+").WillReturnRows(sqlmock.NewRows([]string{"1"}))

	// replace it with a wrapped hnysql.DB
	db := hnysqlx.WrapDB(sqlxodb)
	// and start up a trace to capture all the calls
	ctx, span := beeline.StartSpan(context.Background(), "start")
	defer span.Send()

	// from here on, all SQL calls will emit events.

	_, err = db.ExecContext(ctx, "insert into flavors (flavor) values ('rose')")
	assert.Nil(t, err)
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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
