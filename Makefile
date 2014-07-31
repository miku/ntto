all:
	go fmt nttoldj.go
	go build nttoldj.go

clean:
	rm -f nttoldj
