package browser

import (
	"context"
	"errors"
	"fmt"
	"github.com/aaronland/go-http-bootstrap"
	"github.com/aaronland/go-http-server"
	"github.com/aaronland/go-http-tangramjs"
	"github.com/sfomuseum/go-flags/flagset"
	tzhttp "github.com/sfomuseum/go-http-tilezen/http"
	"github.com/whosonfirst/go-cache"
	"github.com/whosonfirst/go-reader"
	_ "github.com/whosonfirst/go-reader-cachereader"
	"github.com/whosonfirst/go-whosonfirst-browser/v3/assets/templates"
	"github.com/whosonfirst/go-whosonfirst-browser/v3/http"
	"github.com/whosonfirst/go-whosonfirst-search/fulltext"
	"html/template"
	"io/ioutil"
	"log"
	gohttp "net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TO DO: move flag stuff into a separate function and pass in a flags.Flags thing here...

func Start(ctx context.Context) error {

	fs := flagset.NewFlagSet("browser")

	server_uri := fs.String("server-uri", "http://localhost:8080", "A valid aaronland/go-http-server URI.")

	static_prefix := fs.String("static-prefix", "", "Prepend this prefix to URLs for static assets.")

	path_templates := fs.String("templates", "", "An optional string for local templates. This is anything that can be read by the 'templates.ParseGlob' method.")

	data_source := fs.String("reader-source", "whosonfirst-data://", "A valid go-reader Reader URI string.")
	cache_source := fs.String("cache-source", "gocache://", "A valid go-cache Cache URI string.")

	nextzen_api_key := fs.String("nextzen-api-key", "", "A valid Nextzen API key (https://developers.nextzen.org/).")
	nextzen_style_url := fs.String("nextzen-style-url", "/tangram/refill-style.zip", "A valid Tangram scene file URL.")
	nextzen_tile_url := fs.String("nextzen-tile-url", tangramjs.NEXTZEN_MVT_ENDPOINT, "A valid Nextzen MVT tile URL.")

	proxy_tiles := fs.Bool("proxy-tiles", false, "Proxy (and cache) Nextzen tiles.")
	proxy_tiles_url := fs.String("proxy-tiles-url", "/tiles/", "The URL (a relative path) for proxied tiles.")
	proxy_tiles_cache := fs.String("proxy-tiles-cache", "gocache://", "A valid tile proxy DSN string.")
	proxy_tiles_timeout := fs.Int("proxy-tiles-timeout", 30, "The maximum number of seconds to allow for fetching a tile from the proxy.")

	enable_all := fs.Bool("enable-all", false, "Enable all the available output handlers.")
	enable_graphics := fs.Bool("enable-graphics", false, "Enable the 'png' and 'svg' output handlers.")
	enable_data := fs.Bool("enable-data", false, "Enable the 'geojson' and 'spr' and 'select' output handlers.")

	enable_png := fs.Bool("enable-png", false, "Enable the 'png' output handler.")
	enable_svg := fs.Bool("enable-svg", false, "Enable the 'svg' output handler.")

	enable_geojson := fs.Bool("enable-geojson", true, "Enable the 'geojson' output handler.")
	enable_geojsonld := fs.Bool("enable-geojson-ld", true, "Enable the 'geojson-ld' output handler.")
	enable_spr := fs.Bool("enable-spr", true, "Enable the 'spr' (or \"standard places response\") output handler.")
	enable_select := fs.Bool("enable-select", false, "Enable the 'select' output handler.")

	select_pattern := fs.String("select-pattern", "properties(?:.[a-zA-Z0-9-_]+){1,}", "A valid regular expression for sanitizing select parameters.")

	enable_html := fs.Bool("enable-html", true, "Enable the 'html' (or human-friendly) output handlers.")

	enable_search_api := fs.Bool("enable-search-api", true, "Enable the (API) search handlers.")
	enable_search_api_geojson := fs.Bool("enable-search-api-geojson", false, "Enable the (API) search handlers to return results as GeoJSON.")

	enable_search_html := fs.Bool("enable-search-html", true, "Enable the (human-friendly) search handlers.")

	enable_search := fs.Bool("enable-search", false, "Enable both the API and human-friendly search handlers.")

	search_database_uri := fs.String("search-database-uri", "", "A valid whosonfirst/go-whosonfist-search/fulltext URI.")

	path_png := fs.String("path-png", "/png/", "The path that PNG requests should be served from.")
	path_svg := fs.String("path-svg", "/svg/", "The path that SVG requests should be served from.")
	path_geojson := fs.String("path-geojson", "/geojson/", "The path that GeoJSON requests should be served from.")
	path_geojsonld := fs.String("path-geojson-ld", "/geojson-ld/", "The path that GeoJSON-LD requests should be served from.")
	path_spr := fs.String("path-spr", "/spr/", "The path that SPR requests should be served from.")
	path_select := fs.String("path-select", "/select/", "The path that 'select' requests should be served from.")

	path_search_api := fs.String("path-search-api", "/search/spr/", "The path that API 'search' requests should be served from.")
	path_search_html := fs.String("path-search-html", "/search/", "The path that API 'search' requests should be served from.")

	path_id := fs.String("path-id", "/id/", "The that Who's On First documents should be served from.")

	flagset.Parse(fs)

	err := flagset.SetFlagsFromEnvVarsWithFeedback(fs, "BROWSER", true)

	if err != nil {
		return err
	}

	if *enable_all {
		*enable_graphics = true
		*enable_data = true
		*enable_html = true
		*enable_search = true
	}

	if *enable_search {
		*enable_search_api = true
		*enable_search_api_geojson = true
		*enable_search_html = true
	}

	if *enable_graphics {
		*enable_png = true
		*enable_svg = true
	}

	if *enable_data {
		*enable_geojson = true
		*enable_geojsonld = true
		*enable_spr = true
		*enable_select = true
	}

	if *enable_search_html {
		*enable_html = true
	}

	if *enable_html {
		*enable_geojson = true
		*enable_png = true
	}

	if *cache_source == "tmp://" {

		now := time.Now()
		prefix := fmt.Sprintf("go-whosonfirst-browser-%d", now.Unix())

		tempdir, err := ioutil.TempDir("", prefix)

		if err != nil {
			return err
		}

		log.Println(tempdir)
		defer os.RemoveAll(tempdir)

		*cache_source = fmt.Sprintf("fs://%s", tempdir)
	}

	cr_q := url.Values{}
	cr_q.Set("reader", *data_source)
	cr_q.Set("cache", *cache_source)

	cr_uri := url.URL{}
	cr_uri.Scheme = "cachereader"
	cr_uri.RawQuery = cr_q.Encode()

	cr, err := reader.NewReader(ctx, cr_uri.String())

	if err != nil {
		return err
	}

	// start of sudo put me in a package

	t := template.New("whosonfirst-browser").Funcs(template.FuncMap{
		"Add": func(i int, offset int) int {
			return i + offset
		},
	})

	if *path_templates != "" {

		t, err = t.ParseGlob(*path_templates)

		if err != nil {
			return err
		}

	} else {

		for _, name := range templates.AssetNames() {

			body, err := templates.Asset(name)

			if err != nil {
				return err
			}

			t, err = t.Parse(string(body))

			if err != nil {
				return err
			}
		}
	}

	// end of sudo put me in a package

	if *static_prefix != "" {

		*static_prefix = strings.TrimRight(*static_prefix, "/")

		if !strings.HasPrefix(*static_prefix, "/") {
			return errors.New("Invalid -static-prefix value")
		}
	}

	mux := gohttp.NewServeMux()

	ping_handler, err := http.PingHandler()

	if err != nil {
		return err
	}

	mux.Handle("/ping", ping_handler)

	if *enable_png {

		png_opts, err := http.NewDefaultRasterOptions()

		if err != nil {
			return err
		}

		png_handler, err := http.RasterHandler(cr, png_opts)

		if err != nil {
			return err
		}

		mux.Handle(*path_png, png_handler)
	}

	if *enable_svg {

		svg_opts, err := http.NewDefaultSVGOptions()

		if err != nil {
			return err
		}

		svg_handler, err := http.SVGHandler(cr, svg_opts)

		if err != nil {
			return err
		}

		mux.Handle(*path_svg, svg_handler)
	}

	if *enable_spr {

		spr_handler, err := http.SPRHandler(cr)

		if err != nil {
			return err
		}

		mux.Handle(*path_spr, spr_handler)
	}

	if *enable_geojson {

		geojson_handler, err := http.GeoJSONHandler(cr)

		if err != nil {
			return err
		}

		mux.Handle(*path_geojson, geojson_handler)
	}

	if *enable_geojsonld {

		geojsonld_handler, err := http.GeoJSONLDHandler(cr)

		if err != nil {
			return err
		}

		mux.Handle(*path_geojsonld, geojsonld_handler)
	}

	if *enable_select {

		if *select_pattern == "" {
			return errors.New("Missing -select-pattern parameter.")
		}

		pat, err := regexp.Compile(*select_pattern)

		if err != nil {
			return err
		}

		select_opts := &http.SelectHandlerOptions{
			Pattern: pat,
		}

		select_handler, err := http.SelectHandler(cr, select_opts)

		if err != nil {
			return err
		}

		mux.Handle(*path_select, select_handler)
	}

	if *enable_search_api {

		search_db, err := fulltext.NewFullTextDatabase(ctx, *search_database_uri)

		if err != nil {
			return err
		}

		search_opts := http.SearchAPIHandlerOptions{
			Database: search_db,
		}

		if *enable_search_api_geojson {

			search_opts.EnableGeoJSON = true

			geojson_reader, err := reader.NewReader(ctx, *data_source)

			if err != nil {
				return err
			}

			search_opts.GeoJSONReader = geojson_reader

			/*
				if resolver_uri != "" {

				resolver_func, err := geojson.NewSPRPathResolverFunc(ctx, resolver_uri)

				if err != nil {
					return err
				}

				api_pip_opts.SPRPathResolver = resolver_func
			*/
		}

		search_handler, err := http.SearchAPIHandler(search_opts)

		if err != nil {
			return err
		}

		mux.Handle(*path_search_api, search_handler)
	}

	if *enable_html {

		if *proxy_tiles {

			tile_cache, err := cache.NewCache(ctx, *proxy_tiles_cache)

			if err != nil {
				return err
			}

			timeout := time.Duration(*proxy_tiles_timeout) * time.Second

			proxy_opts := &tzhttp.TilezenProxyHandlerOptions{
				Cache:   tile_cache,
				Timeout: timeout,
			}

			proxy_handler, err := tzhttp.TilezenProxyHandler(proxy_opts)

			if err != nil {
				return err
			}

			// the order here is important - we don't have a general-purpose "add to
			// mux with prefix" function here, like we do in the tangram handler so
			// we set the nextzen tile url with *proxy_tiles_url and then update it
			// (*proxy_tiles_url) with a prefix if necessary (20190911/thisisaaronland)

			*nextzen_tile_url = fmt.Sprintf("%s{z}/{x}/{y}.mvt", *proxy_tiles_url)

			if *static_prefix != "" {

				*proxy_tiles_url = filepath.Join(*static_prefix, *proxy_tiles_url)

				if !strings.HasSuffix(*proxy_tiles_url, "/") {
					*proxy_tiles_url = fmt.Sprintf("%s/", *proxy_tiles_url)
				}
			}

			mux.Handle(*proxy_tiles_url, proxy_handler)
		}

		bootstrap_opts := bootstrap.DefaultBootstrapOptions()

		tangramjs_opts := tangramjs.DefaultTangramJSOptions()
		tangramjs_opts.Nextzen.APIKey = *nextzen_api_key
		tangramjs_opts.Nextzen.StyleURL = *nextzen_style_url
		tangramjs_opts.Nextzen.TileURL = *nextzen_tile_url

		endpoints := &http.Endpoints{
			Data:  *path_geojson,
			Png:   *path_png,
			Svg:   *path_svg,
			Spr:   *path_spr,
			Id:    *path_id,
			Index: "/",
		}

		if *enable_search_html {
			endpoints.Search = *path_search_html
		}

		index_opts := http.IndexHandlerOptions{
			Templates: t,
			Endpoints: endpoints,
		}

		index_handler, err := http.IndexHandler(index_opts)

		if err != nil {
			return err
		}

		index_handler = bootstrap.AppendResourcesHandlerWithPrefix(index_handler, bootstrap_opts, *static_prefix)

		mux.Handle("/", index_handler)

		id_opts := http.IDHandlerOptions{
			Templates: t,
			Endpoints: endpoints,
		}

		id_handler, err := http.IDHandler(cr, id_opts)

		if err != nil {
			return err
		}

		id_handler = bootstrap.AppendResourcesHandlerWithPrefix(id_handler, bootstrap_opts, *static_prefix)
		id_handler = tangramjs.AppendResourcesHandlerWithPrefix(id_handler, tangramjs_opts, *static_prefix)

		mux.Handle(*path_id, id_handler)

		if *enable_search_html {

			search_db, err := fulltext.NewFullTextDatabase(ctx, *search_database_uri)

			if err != nil {
				return err
			}

			search_opts := http.SearchHandlerOptions{
				Templates: t,
				Endpoints: endpoints,
				Database:  search_db,
			}

			search_handler, err := http.SearchHandler(search_opts)

			if err != nil {
				return err
			}

			search_handler = bootstrap.AppendResourcesHandlerWithPrefix(search_handler, bootstrap_opts, *static_prefix)
			search_handler = tangramjs.AppendResourcesHandlerWithPrefix(search_handler, tangramjs_opts, *static_prefix)

			mux.Handle(*path_search_html, search_handler)
		}

		err = bootstrap.AppendAssetHandlersWithPrefix(mux, *static_prefix)

		if err != nil {
			return err
		}

		err = tangramjs.AppendAssetHandlersWithPrefix(mux, *static_prefix)

		if err != nil {
			return err
		}

		err = http.AppendStaticAssetHandlersWithPrefix(mux, *static_prefix)

		if err != nil {
			return err
		}

	}

	s, err := server.NewServer(ctx, *server_uri)

	if err != nil {
		return err
	}

	log.Printf("Listening on %s\n", s.Address())

	return s.ListenAndServe(ctx, mux)
}
