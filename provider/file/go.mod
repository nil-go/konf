module github.com/ktong/konf/provider/file

go 1.21

require github.com/fsnotify/fsnotify v1.7.0

require ( // for test
	github.com/stretchr/testify v1.8.4
	go.uber.org/goleak v1.3.0
)

require ( // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
