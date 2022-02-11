all: vmlinux.h headers probe.o static-libs unlinksnoop

unlinksnoop: main.go
	CGO_CFLAGS="-I$(CURDIR)/headers/usr/include" CGO_LDFLAGS="-L$(CURDIR) -lbpf -lz -lelf"  go build -ldflags='-w -extldflags "-static"' -v -o unlinksnoop main.go

vmlinux.h:
	bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h

probe.o: headers vmlinux.h probe.bpf.c
	clang -g -O2 -I$(CURDIR)/headers/usr/include -target bpf -c probe.bpf.c -o probe.o

headers:
	mkdir -p headers
	make -C libbpf*/src install_headers install_uapi_headers DESTDIR=$(CURDIR)/headers

libbpf.a:
	cd libbpf-*/; make -C src libbpf.a; cp src/libbpf.a ../

libelf.a:
	cd elfutils-*/; ./configure --disable-debuginfod --disable-libdebuginfod; make -C libelf libelf.a; cp libelf/libelf.a ../

libz.a:
	cd zlib-*/; ./configure; make libz.a; cp libz.a ../

.PHONY: update-modules clean install uninstall static-libs download-libs headers

static-libs: libbpf.a libelf.a libz.a

download-libs:
	curl -L https://sourceware.org/elfutils/ftp/0.186/elfutils-0.186.tar.bz2 | tar xjf -
	curl -L https://github.com/madler/zlib/archive/v1.2.11.tar.gz | tar xzf -
	curl -L https://github.com/libbpf/libbpf/archive/v0.6.1.tar.gz | tar xzf -

update-modules:
	go mod tidy

clean:
	rm -rf updates.img vmlinux.h unlinksnoop probe.o libbpf.a libz.a libelf.a headers *.tar.gz *.tar.bz2 elfutils-* libbpf-* zlib-*

install:
	mkdir -p $(DESTDIR)/usr/lib/systemd/system/ $(DESTDIR)/usr/libexec/unlinksnoop $(DESTDIR)/usr/bin
	cp -f unlinksnoop.service $(DESTDIR)/usr/lib/systemd/system/
	cp -f probe.o $(DESTDIR)/usr/libexec/unlinksnoop/
	cp -f unlinksnoop $(DESTDIR)/usr/bin/
	systemctl --root=$(DESTDIR) enable unlinksnoop.service

uninstall:
	systemctl --root=$(DESTDIR) --now disable unlinksnoop.service
	rm -f $(DESTDIR)/usr/lib/systemd/system/unlinksnoop.service $(DESTDIR)/usr/bin/unlinksnoop
	rm -rf $(DESTDIR)/usr/libexec/unlinksnoop
	systemctl daemon-reload

updates.img: all unlinksnoop.service
	$(eval DIR:=$(shell mktemp -d))
	make install DESTDIR=$(DIR)
	bash -c "cd $(DIR) ; find . | cpio -o -c | gzip > $(CURDIR)/updates.img"
	rm -rf $(DIR)