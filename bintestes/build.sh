
go install github.com/teknus/Radio/format_msg
go build ../server/server.go
go build ../client_listenner/client_listenner.go
go build ../client_control/client_control.go
./server 1234 Afterlife128.mp3 still_waiting_128.mp3 10sec.mp3
