VERSION=0.1.0
BUILD=$(shell git rev-list --count HEAD)

dpkg:
	mkdir -p deb/nagios_watcher/usr/local/bin
	cp $(GOPATH)/bin/nagios_watcher deb/nagios_watcher/usr/local/bin
	fpm -s dir -t deb -n nagios_watcher -v $(VERSION)-$(BUILD) -C deb/nagios_watcher .

build:
	go build

install:
	go install

clean:
	rm -rf deb
	rm -f *.deb
	rm -f nagios_watcher
	rm -f $(GOPATH)/bin/*

all: install dpkg
