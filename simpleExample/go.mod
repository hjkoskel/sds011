module simpleexample

go 1.23.2

require (
	github.com/fatih/color v1.18.0
	github.com/hjkoskel/listserialports v0.1.1
	github.com/hjkoskel/sds011 v0.0.0-20191117062440-5517d992fee6
	github.com/pkg/term v1.1.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.28.0 // indirect
)

replace github.com/hjkoskel/sds011 => ../
