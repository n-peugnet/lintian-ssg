# Lintian SSG

[![build][build-img]][build-url]

A very simple static site generator to replace `lintian.debian.org`,
currently hosted at <https://lintian.club1.fr/> as a demo.

```sh
sudo apt install golang lintian
go generate   # to include lintian-ssg version, only from a git checkout
go build
go test ./... # optionally, to run tests
./lintian-ssg
```

The result sould be in the `out` directory.

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
