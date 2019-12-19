package main

import (
	"context"
	_ "github.com/mattn/go-sqlite3"
	sql_reader "github.com/whosonfirst/go-reader-database-sql"
	"github.com/whosonfirst/go-whosonfirst-browser"
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"strconv"
)

func main() {

	sql_reader.URI_READFUNC = func(uri string) (string, error) {

		id, err := wof_uri.IdFromPath(uri)

		if err != nil {
			return "", err
		}

		str_id := strconv.FormatInt(id, 10)
		return str_id, nil
	}

	ctx := context.Background()
	err := browser.Start(ctx)

	if err != nil {
		log.Fatal(err)
	}

}
