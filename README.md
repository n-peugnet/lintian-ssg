# Lintian SSG

[![build][build-img]][build-url]

A very simple static site generator to replace `lintian.debian.org`,
currently hosted at <https://lintian.club1.fr/> as a demo.

## Build

The only build requirement is Go >= `1.19`.

```sh
go generate   # to include lintian-ssg version, only from a git checkout
go build
go test ./... # optionally, to run tests
```

## Usage

Lintian must be installed.

```
Usage of lintian-ssg:
  --base-url string
        URL, including the scheme and final slash, where the root of the website will be
        located. This will be used to emit the canonical URL of each page and the sitemap.
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
