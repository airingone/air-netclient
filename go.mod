module github.com/airingone/air-netclient

go 1.13

require (
	github.com/airingone/air-etcd v1.0.4
	github.com/airingone/air-netserver v0.0.0-20200917150859-aade18368c78
	github.com/airingone/config v1.0.8
	github.com/airingone/log v1.0.1
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
)

replace github.com/airingone/air-netserver v0.0.0-20200917150859-aade18368c78 => /Users/air/go/src/airingone/air-netserver

replace github.com/airingone/air-etcd v1.0.4 => /Users/air/go/src/airingone/air-etcd

replace github.com/airingone/config v1.0.8 => /Users/air/go/src/airingone/config
