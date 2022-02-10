
rmmon: main.go
	go build -o rmmon

.PHONY: update-modules clean install uninstall

update-modules: 
	go mod tidy

clean:
	rm -f rmmon

install:
	cp -f rmmon.service /etc/systemd/system/
	cp -f rmmon /usr/local/bin/
	systemctl daemon-reload

uninstall:
	-systemctl stop rmmon.service
	-systemctl --signal=TERM kill rmmon.service
	rm -f /etc/systemd/system/rmmon.service /usr/local/bin/rmmon
	systemctl daemon-reload


updates.img: rmmon rmmon.service
	$(eval DIR:=$(shell mktemp -d))
	mkdir -p $(DIR)/etc/systemd/system
	mkdir -p $(DIR)/etc/systemd/system/basic.target.wants
	mkdir -p $(DIR)/usr/local/bin/
	cp -f rmmon $(DIR)/usr/local/bin/
	cp -f rmmon.service $(DIR)/etc/systemd/system/
	ln -s /etc/systemd/system/rmmon.service $(DIR)/etc/systemd/system/basic.target.wants/rmmon.service
	bash -c "cd $(DIR) ; find . | cpio -o -c | gzip > $(CURDIR)/updates.img"
	rm -rf $(DIR)