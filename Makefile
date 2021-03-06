VERSION=0.1.0
BUILD=$(shell git rev-list --count HEAD)

dpkg:
	mkdir -p deb/nagios_watcher/usr/local/bin
	mkdir -p deb/nagios_watcher/etc/init
	mkdir -p deb/nagios_watcher/etc/default
	cp $(GOPATH)/bin/nagios_watcher deb/nagios_watcher/usr/local/bin
	cp $(GOPATH)/src/nagios_watcher/nagios_watcher.init deb/nagios_watcher/etc/init/nagios_watcher.conf
	touch deb/nagios_watcher/etc/default/nagios_watcher
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
