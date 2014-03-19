
GOPATH=$(PWD)
INSTALLDIR=/usr/bin

GO_DEPS=src/github.com/fsouza/go-dockerclient

build: bin/longshoreman

install: build
	sudo cp bin/longshoreman $(INSTALLDIR)/longshoreman

bin/longshoreman: deps
	[ -d bin ] || mkdir bin
	go build -o bin/longshoreman main.go

deps: $(GO_DEPS)

src/%:
	go get $(subst src/,,$@)

# creates a debian package for longshoreman
# to install `sudo dpkg -i longshoreman.deb`
dpkg: build
	dpkg-deb --version >/dev/null 2>&1 || (echo "Unable to run 'dpkg-deb'" && exit 1)
	rm -rf deb/work
	mkdir -p deb/work/usr/bin
	mkdir -p deb/work/DEBIAN
	mkdir -p deb/work/usr/share/doc/longshoreman
	cp bin/longshoreman deb/work/usr/bin/longshoreman
	cp deb/control deb/work/DEBIAN/control
	echo "whatever" > deb/work/usr/share/doc/longshoreman/copyright
	echo "whatever" > deb/work/usr/share/doc/longshoreman/changelog
	dpkg-deb --build deb/work
	mv deb/work.deb longshoreman.deb

clean:
	rm bin/longshoreman
	rm -rf deb/work

realclean: clean
	rm -rf src
