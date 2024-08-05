# Lintian SSG

A very simple static site generator to replace `lintian.debian.org`.

```sh
sudo apt install golang lintian
go generate // only from a git checkout
go build
./lintian-ssg
```

The result sould be in the `out` directory.
