package reader

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	wof_reader "github.com/whosonfirst/go-reader"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"regexp"
	"strings"
)

type readFunc func(*SQLReader, string) (string, error)

type queryFunc func(*SQLReader, string) (string, []interface{}, error)

var VALID_TABLE *regexp.Regexp
var VALID_KEY *regexp.Regexp
var VALID_VALUE *regexp.Regexp

var URI_READFUNC readFunc
var URI_QUERYFUNC queryFunc

func init() {

	VALID_TABLE = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	VALID_KEY = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	VALID_VALUE = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

	r := NewSQLReader()
	wof_reader.Register("sql", r)
}

type SQLReader struct {
	wof_reader.Reader
	conn  *sql.DB
	Table string
	Key   string
	Value string
}

func NewSQLReader() wof_reader.Reader {

	r := SQLReader{}
	return &r
}

// sql://sqlite/geojson/id/body/?dsn=....

func (r *SQLReader) Open(ctx context.Context, uri string) error {

	u, err := url.Parse(uri)

	if err != nil {
		return err
	}

	q := u.Query()

	driver := u.Host
	path := u.Path

	path = strings.TrimLeft(path, "/")
	parts := strings.Split(path, "/")

	if len(parts) != 3 {
		return errors.New("Invalid path")
	}

	table := parts[0]
	key := parts[1]
	value := parts[2]

	dsn := q.Get("dsn")

	if dsn == "" {
		return errors.New("Missing dsn parameter")
	}

	conn, err := sql.Open(driver, dsn)

	if err != nil {
		return err
	}

	if !VALID_TABLE.MatchString(table) {
		return errors.New("Invalid table")
	}

	if !VALID_KEY.MatchString(key) {
		return errors.New("Invalid key")
	}

	if !VALID_VALUE.MatchString(value) {
		return errors.New("Invalid value")
	}

	r.conn = conn
	r.Table = table
	r.Key = key
	r.Value = value

	return nil
}

func (r *SQLReader) Read(ctx context.Context, raw_uri string) (io.ReadCloser, error) {

	uri := raw_uri
	
	if URI_READFUNC != nil {

		new_uri, err := URI_READFUNC(r, raw_uri)

		if err != nil {
			return nil, err
		}

		uri = new_uri
	}

	q := fmt.Sprintf("SELECT %s FROM %s WHERE %s=?", r.Value, r.Table, r.Key)
	
	q_args := []interface{}{
		uri,
	}
	
	if URI_QUERYFUNC != nil {

		new_q, new_args, err := URI_QUERYFUNC(r, raw_uri)

		if err != nil {
			return nil, err
		}

		q = new_q
		q_args = new_args
	}

	log.Println("Q", q, q_args)
	
	row := r.conn.QueryRowContext(ctx, q, q_args)

	var body string
	err := row.Scan(&body)

	if err != nil {
		return nil, err
	}

	sr := strings.NewReader(body)
	fh := ioutil.NopCloser(sr)

	return fh, nil
}
