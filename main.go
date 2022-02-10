package main

import (
	"flag"
	"io/ioutil"
	"log"
	"log/syslog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

var dir *string = flag.String("d", "/", "directory to monitor")

func main() {
	flag.Parse()
	if *dir != "/" {
		fi, err := os.Stat(*dir)
		if err == nil && !fi.IsDir() {
			log.Fatalf("path to monitor is not a directory")
		}
	}

	arg0 := filepath.Base(os.Args[0])

	if arg0[0] != '@' {
		binary, err := ioutil.ReadFile("/proc/self/exe")
		if err != nil {
			log.Fatalf("failed to read in current executable into memory: %v", err)
		}

		if err = ioutil.WriteFile("/dev/shm/@"+arg0, binary, 0755); err != nil {
			log.Fatalf("failed to copy executable to /dev/shm: %v", err)
		}

		err = syscall.Exec("/dev/shm/@rmmon", []string{"@rmmon", "-d", *dir}, nil)
		if err != nil {
			log.Fatalf("failed to reexectue: %v", err)
		}
	} else {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatalf("failed to setup fsnotify: %v", err)
		}
		defer watcher.Close()

		// Try to get information about syslog server from kernel commandline
		cmdline, err := ioutil.ReadFile("/proc/cmdline")
		if err != nil {
			log.Printf("failed to read /proc/cmdline: %v", err)
		}
		parts := strings.Fields(string(cmdline))
		var logServer string

		for _, p := range parts {
			if strings.HasPrefix(p, "rmmon.syslog=") {
				s := strings.Split(p, "=")
				if len(s) == 2 {
					logServer = s[1]
				}
			}
		}

		go func() {
			var slog *log.Logger

			syslogWriter, err := syslog.Dial("udp", logServer+":514", syslog.LOG_WARNING|syslog.LOG_DAEMON, "rmmon")
			if err == nil {
				slog = log.New(syslogWriter, "", 0)
			}

			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Remove == fsnotify.Remove {
						log.Println("removed:", event.Name)

						if slog != nil {
							slog.Println("removed:", event.Name)
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Printf("directory watcher received an error: %v", err)

					if slog != nil {
						slog.Printf("directory watcher received an error: %v", err)
					}
				}
			}
		}()

		err = watcher.Add(*dir)
		if err != nil {
			log.Fatalf("failed to watch directory \"%v\" for changes: %v", *dir, err)
		}

		done := make(chan bool)
		<-done
	}
}
