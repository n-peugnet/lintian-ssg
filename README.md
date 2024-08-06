# Lintian SSG

A very simple static site generator to replace `lintian.debian.org`.

```bash
sudo apt install golang lintian
go generate // to include lintian-ssg version, only from a git checkout
go build
./lintian-ssg
```

The result sould be in the `out` directory.
