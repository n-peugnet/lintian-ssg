# Lintian SSG

[![build][build-img]][build-url] [![coverage][cover-img]][cover-url] [![goreport][report-img]][report-url]

A very simple static site generator to replace `lintian.debian.org`,
currently hosted at <https://lintian.club1.fr/> as a demo.

## Build

The only build requirement is Go >= `1.19`.

```sh
go generate ./version # to include version, only from a git checkout
go build
go test ./...         # optionally, to run tests
```

## Usage

Lintian must be installed.

```--help
Usage of lintian-ssg:
  --base-url string
        URL, including the scheme, where the root of the website will be located.
        This will be used in the sitemap and in the canonical URL of each page.
  --footer string
        Text to add to the footer, inline Markdown elements will be parsed.
  -h, --help
        Show this help and exit.
  --no-sitemap
        Disable sitemap.txt generation.
  -o, --output-dir string
        Path of the directory where to output the generated website. (default "out")
  --stats
        Display some statistics.
  --version
        Show version and exit.
```

## Recommended HTTP server configs

### Apache (global, vhost)

```apache
# For a more friendly 404 error page
ErrorDocument 404 /404.html

<Location "/tags/">
	# To allow access .html files without their extension
	Options +MultiViews
</Location>
```

### Nginx (server)

```nginx
# For a more friendly 404 error page
error_page 404 /404.html;

location /tags/ {
    # To allow access .html files without their extension
    try_files $uri.html $uri =404;
}
```

[build-img]: https://github.com/n-peugnet/lintian-ssg/actions/workflows/build.yml/badge.svg
[build-url]: https://github.com/n-peugnet/lintian-ssg/actions/workflows/build.yml
[cover-img]: https://img.shields.io/codecov/c/gh/n-peugnet/lintian-ssg?token=8RRU5MBX0V
[cover-url]: https://codecov.io/gh/n-peugnet/lintian-ssg
[report-img]: https://goreportcard.com/badge/github.com/n-peugnet/lintian-ssg
[report-url]: https://goreportcard.com/report/github.com/n-peugnet/lintian-ssg
