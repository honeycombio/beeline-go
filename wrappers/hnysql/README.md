# Example instrumentation

```
import (
  gosql "database/sql"
  
  sql "github.com/honeycombio/honeycomb-go-magic/wrappers/hnysql"
  "github.com/honeycombio/libhoney-go"
)
```

```
odb, err := gosql.Open("mysql", "root:@tcp...")
if err != nil {
  log.Fatal(err)
}
db := sql.WrapDB(libhoney.NewBuilder(), odb)
defer db.Close()
```
