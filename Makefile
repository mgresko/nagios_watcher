VERSION=0.1.0
BUILD=$(shell git rev-list --count HEAD)

dpkg:
	mkdir -p deb/nagios_watcher/usr/local/bin
	cp $(GOPATH)/bin/nagios_watcher deb/nagios_watcher/usr/local/bin
	fpm -s dir -t deb -n nagios_watcher -v $(VERSION)-$(BUILD)

build:
	go build

install:
	go install
