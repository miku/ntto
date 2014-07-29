TARGETS = ntto

# http://docs.travis-ci.com/user/languages/go/#Default-Test-Script
test:
	go get -d && go test -v

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

# experimental deb building
deb: $(TARGETS)
	mkdir -p debian/ntto/usr/sbin
	cp ntto debian/ntto/usr/sbin
	cd debian && fakeroot dpkg-deb --build ntto .

REPOPATH = /usr/share/nginx/html/repo/CentOS/6/x86_64

publish: rpm
	cp ntto-*.rpm $(REPOPATH)
	createrepo $(REPOPATH)

# rpm building via vagrant -- assume this is the first VM,
# otherwise there will be port collisions
SSHCMD = ssh -o StrictHostKeyChecking=no -i vagrant.key vagrant@127.0.0.1 -p 2222
SCPCMD = scp -o port=2222 -o StrictHostKeyChecking=no -i vagrant.key

rpm: $(TARGETS)
	mkdir -p $(HOME)/rpmbuild/BUILD
	mkdir -p $(HOME)/rpmbuild/SOURCES
	mkdir -p $(HOME)/rpmbuild/SPECS
	mkdir -p $(HOME)/rpmbuild/RPMS
	cp ./packaging/ntto.spec $(HOME)/rpmbuild/SPECS
	cp ntto $(HOME)/rpmbuild/BUILD
	./packaging/buildrpm.sh ntto
	cp $(HOME)/rpmbuild/RPMS/x86_64/*rpm .

# Helper to build RPM on a RHEL6 VM, to link against glibc 2.12
vagrant.key:
	curl -sL "https://raw.githubusercontent.com/mitchellh/vagrant/master/keys/vagrant" > vagrant.key
	chmod 0600 vagrant.key

# # Don't forget to vagrant up :) - and add your public key to the guests authorized_keys
# vm-setup: vagrant.key
# 	$(SSHCMD) "sudo yum install -y sudo yum install http://ftp.riken.jp/Linux/fedora/epel/6/i386/epel-release-6-8.noarch.rpm"
# 	$(SSHCMD) "sudo yum install -y golang git rpm-build"
# 	$(SSHCMD) "mkdir -p /home/vagrant/src/github.com/miku"
# 	$(SSHCMD) "cd /home/vagrant/src/github.com/miku && git clone https://github.com/miku/marctools.git"

# rpm-compatible: vagrant.key
# 	$(SSHCMD) "cd /home/vagrant/src/github.com/miku/marctools && GOPATH=/home/vagrant go get"
# 	$(SSHCMD) "cd /home/vagrant/src/github.com/miku/marctools && git pull origin master && pwd && GOPATH=/home/vagrant make rpm"
# 	$(SCPCMD) vagrant@127.0.0.1:/home/vagrant/src/github.com/miku/marctools/*rpm .
