# Lintian SSG

A very simple static site generator to replace `lintian.debian.org`.

```sh
sudo apt install golang lintian
go generate   # to include lintian-ssg version, only from a git checkout
go build
go test ./... # optionally, to run tests
./lintian-ssg
```

The result sould be in the `out` directory.

## Recommended Apache config

```apache
# For a more friendly 404 error page
ErrorDocument 404 /404.html
# To allow access .html files without their extension
Options +MultiViews
```
