module github.com/A2AGateway/a2a-connector

go 1.21

require (
	github.com/A2AGateway/a2a-protocol v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/A2AGateway/a2a-protocol => ../a2a-protocol
