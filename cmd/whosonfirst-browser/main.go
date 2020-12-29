package main

import (
	"context"
	_ "github.com/mattn/go-sqlite3"
	sql_reader "github.com/whosonfirst/go-reader-database-sql"
	"github.com/whosonfirst/go-whosonfirst-browser/v3"
	_ "github.com/whosonfirst/go-whosonfirst-search-sqlite"	
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"strconv"
)

func main() {

	sql_reader.URI_READFUNC = func(raw_uri string) (string, error) {

		id, _, err := wof_uri.ParseURI(raw_uri)

		if err != nil {
			return "", err
		}

		str_id := strconv.FormatInt(id, 10)
		return str_id, nil
	}
	
	sql_reader.URI_QUERYFUNC = func(raw_uri string) (string, []interface{}, error) {

		_, uri_args, err := wof_uri.ParseURI(raw_uri)

		if err != nil {
			return "", nil, err
		}

		if !uri_args.IsAlternate {
			return "", nil, nil
		}

		alt_label, err := uri_args.AltGeom.String()

		if err != nil {
			return "", nil, err
		}

		where := "alt_label = ?"

		args := []interface{}{
			alt_label,
		}

		return where, args, nil
	}
	
	ctx := context.Background()
	err := browser.Start(ctx)

	if err != nil {
		log.Fatal(err)
	}

}
