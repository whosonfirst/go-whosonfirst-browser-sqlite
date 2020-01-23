package main

import (
	"context"
	_ "github.com/mattn/go-sqlite3"
	sql_reader "github.com/whosonfirst/go-reader-database-sql"
	"github.com/whosonfirst/go-whosonfirst-browser"
	wof_uri "github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"strconv"
	"fmt"	
)

func main() {

	/*
	sql_reader.URI_READFUNC = func(uri string) (string, error) {

		id, err := wof_uri.IdFromPath(uri)

		if err != nil {
			return "", err
		}

		str_id := strconv.FormatInt(id, 10)
		return str_id, nil
	}
	*/
	
	sql_reader.URI_QUERYFUNC = func(r *sql_reader.SQLReader, uri string) (string, []interface{}, error) {

		id, uri_args, err := wof_uri.ParseURI(uri)

		if err != nil {
			return "", nil, err
		}

		str_id := strconv.FormatInt(id, 10)

		q := fmt.Sprintf("SELECT %s FROM %s WHERE %s=?", r.Value, r.Table, r.Key)		

		q_args := []interface{}{
			str_id,
		}

		if uri_args.IsAlternate {

			str_alt, err := uri_args.AltGeom.String()

			if err != nil {
				return "", nil, err
			}
			
			q = fmt.Sprintf("%s AND source=?", q)
			q_args = append(q_args, str_alt)
		}
		
		return q, q_args, nil
	}
	
	ctx := context.Background()
	err := browser.Start(ctx)

	if err != nil {
		log.Fatal(err)
	}

}
