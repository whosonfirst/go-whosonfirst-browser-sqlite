package main

import (
	"context"
	"github.com/whosonfirst/go-whosonfirst-browser"
	"log"
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"	
	sql_reader "github.com/whosonfirst/go-reader-database-sql"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	
)

func main() {

	sql_reader.URI_READFUNC = func(uri string) (string, error) {
		id, _ := wof_uri.IdFromPath(uri)		
		str_id := strconv.FormatInt(id, 10)
		return str_id, nil
	}
	
	ctx := context.Background()
	err := browser.Start(ctx)

	if err != nil {
		log.Fatal(err)
	}

}
