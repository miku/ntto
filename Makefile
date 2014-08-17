SHELL := /bin/bash
TARGETS = ntto

# http://docs.travis-ci.com/user/languages/go/#Default-Test-Script
test:
	go get -d && go test -v

imports:
	goimports -w .

fmt:
	go fmt ./...

all: fmt test
	go build

install:
	go install

clean:
	go clean
	rm -f coverage.out
	rm -f ntto
	rm -f ntto-*.x86_64.rpm
	rm -f debian/ntto*.deb
	rm -rf debian/ntto/usr

cover:
	go get -d && go test -v	-coverprofile=coverage.out
	go tool cover -html=coverage.out

ntto:
	go build cmd/ntto/ntto.go

# ==== packaging

deb: $(TARGETS)
	mkdir -p debian/ntto/usr/sbin
	cp ntto debian/ntto/usr/sbin
	cd debian && fakeroot dpkg-deb --build ntto .

REPOPATH = /usr/share/nginx/html/repo/CentOS/6/x86_64

publish: rpm
	cp ntto-*.rpm $(REPOPATH)
	createrepo $(REPOPATH)

rpm: $(TARGETS)
	mkdir -p $(HOME)/rpmbuild/{BUILD,SOURCES,SPECS,RPMS}
	cp ./packaging/ntto.spec $(HOME)/rpmbuild/SPECS
	cp ntto $(HOME)/rpmbuild/BUILD
	./packaging/buildrpm.sh ntto
	cp $(HOME)/rpmbuild/RPMS/x86_64/ntto*.rpm .
