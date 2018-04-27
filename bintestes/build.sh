
go install github.com/teknus/Radio/format_msg
go build ../server/server.go
go build ../client_listenner/cliente_listenner.go
go build ../client_control/client_control.go
./server 1234
