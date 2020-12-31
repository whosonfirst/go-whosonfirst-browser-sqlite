cli:
	go build -mod vendor -o bin/whosonfirst-browser cmd/whosonfirst-browser/main.go

debug:
	go run -mod vendor cmd/whosonfirst-browser/main.go -enable-all -proxy-tiles -nextzen-api-key $(APIKEY) -reader-source 'sql://sqlite3/geojson/id/body?dsn=$(DSN)' -search-database-uri 'sqlite://?dsn=$(DSN)'
