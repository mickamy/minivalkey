module github.com/mickamy/minivalkey/e2e

go 1.24

replace github.com/mickamy/minivalkey => ../

require (
	github.com/mickamy/minivalkey v0.0.0
	github.com/stretchr/testify v1.11.1
	github.com/valkey-io/valkey-go v1.0.67
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
