# unlinksnoop
unlinksnoop is very focused debugging tool.  It's main goal is to
monitor deletion of files.  If file deletion is attempted then
unlinksnoop will log about this event.  unlinksnoop uses eBPF for
low-overhead system-wide monitoring.

unlinksnoop is compiled as static binary which is completely
standalone and if started via systemd it will continue running and
monitoring until the system is shutdown.  To achieve this behavior,
unlinksnoop will copy itself into /dev/shm and re-execute from there
in order to not block umount of /usr.

unlinksnoop will log about deletion of files to the journal. If you
want to have log events from late shutdown you can set syslog server
IP address using -s flag or on kernel command line
using unlinksnoop.syslog= option.

# Dependencies
- bpftool
- clang
- glibc-static
- go

# Build instructions
```shell
$ git clone https://github.com/msekletar/unlinksnoop
$ cd unlinksnoop
$ make download-libs
$ make
$ sudo make install
```

# License
- MIT
- MIT or GPL-2.0 for eBPF code (`probe.bpf.c`)