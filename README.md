# rmmon
rmmon is very narowly focused debugging tool. Its main goal is to monitor given directory for removal of files
or directories. If a file or directory is deleted from the monitored path rmmon will log about this event.

rmmon is compiled as static binary which is completely standalone and once started via systemd it will continue
running and monitoring until the system is shutdown. To achieve this behavior, rmmon will copy itself into
/dev/shm and reexecute from there in order to not block umount of /usr. Also, rmmon.service has KillMode= set to
"none".

rmmon will log about removal to the journal and also it will try to send the message to the syslog server
(over UDP port 514). To enable this functionality please specify "rmmon.syslog=<SERVER_ADDRESS>" on the
kernel comand line.

# How to build rmmon?
Simply run "go build" in the root directory or type "make". In case the build failed, make sure that you have
updated the Go dependency cache. To do that you can run "make update-modules".

# Requirements
- Go 1.17
- systemd
