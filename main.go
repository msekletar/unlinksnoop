package main

import "C"
import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	bpf "github.com/aquasecurity/libbpfgo"
)

const (
	INT_SIZE      = 4
	COMM_SIZE     = 16
	FILENAME_SIZE = 1024

	PROBE_PATH = "/usr/libexec/unlinksnoop/probe.o"
)

type unlinkEvent struct {
	pid            int
	comm, filename string
}

var files *string = flag.String("f", "", "Comma-separated list of filenames to filter")
var syslogServer *string = flag.String("s", "", "Syslog server (tcp/514) where to send logs")

func (e unlinkEvent) String() string {
	return fmt.Sprintf("%v,%v,%v", e.comm, e.pid, e.filename)
}

func setupLogging() *log.Logger {
	cmdline, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		log.Printf("failed to read /proc/cmdline: %v", err)
	}
	parts := strings.Fields(string(cmdline))
	var logServer = *syslogServer

	for _, p := range parts {
		if strings.HasPrefix(p, "unlinksnoop.syslog=") {
			s := strings.Split(p, "=")
			if len(s) == 2 {
				logServer = s[1]
			}
		}
	}

	if len(logServer) > 0 {
		syslogWriter, err := syslog.Dial("tcp", logServer+":514", syslog.LOG_WARNING|syslog.LOG_DAEMON, "unlinksnoop")
		if err == nil {
			return log.New(syslogWriter, "", 0)
		}
	}

	return log.Default()
}

func main() {
	flag.Parse()
	filenames := strings.Split(*files, ",")

	arg0 := filepath.Base(os.Args[0])

	if os.Getppid() == 1 && arg0[0] != '@' {
		binary, err := ioutil.ReadFile("/proc/self/exe")
		if err != nil {
			log.Fatalf("failed to read in current executable into memory: %v", err)
		}

		if err = ioutil.WriteFile("/dev/shm/@"+arg0, binary, 0755); err != nil {
			log.Fatalf("failed to copy executable to /dev/shm: %v", err)
		}

		err = syscall.Exec("/dev/shm/@unlinksnoop", append([]string{"@unlinksnoop"}, os.Args[1:]...), nil)
		if err != nil {
			log.Fatalf("failed to reexectue: %v", err)
		}
	} else {
		logger := setupLogging()

		bpfModule, err := bpf.NewModuleFromFile(PROBE_PATH)
		if err != nil {
			logger.Fatalf("failed to create BPF module: %v", err)
		}
		defer bpfModule.Close()

		bpfModule.BPFLoadObject()
		prog, err := bpfModule.GetProgram("handle_unlink")
		if err != nil {
			logger.Fatalf("failed to load BPF proogram: %v", err)
		}

		_, err = prog.AttachTracepoint("syscalls", "sys_enter_unlinkat")
		if err != nil {
			logger.Fatalf("failed to attach BPF program to tracepoint: %v", err)
		}

		eventsChannel := make(chan []byte)
		ring, err := bpfModule.InitRingBuf("events", eventsChannel)
		if err != nil {
			logger.Fatalf("failed to initialize BPF ring buffer: %v", err)
		}

		ring.Start()
		defer func() {
			ring.Stop()
			ring.Close()
		}()

		logger.Println("comm,pid,filename")

		for {
			buf := <-eventsChannel
			var event unlinkEvent

			var pid *C.int
			var comm, filename *C.char

			pid = (*C.int)(unsafe.Pointer(&buf[0]))
			comm = (*C.char)(unsafe.Pointer(&buf[INT_SIZE]))
			filename = (*C.char)(unsafe.Pointer(&buf[INT_SIZE+COMM_SIZE]))

			event.pid = int(*pid)
			event.comm = C.GoString(comm)
			event.filename = C.GoString(filename)

			if len(filenames) > 0 {
				for _, f := range filenames {
					if strings.Contains(event.filename, f) {
						logger.Println(event)
					}
				}
			} else {
				logger.Println(event)
			}
		}
	}
}
