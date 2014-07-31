package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/howeyc/fsnotify"
)

// command line options
var OrgPath = flag.String("config-dir", "/etc/nagios.sync", "path to nagios config files")
var dryrun = flag.Bool("dryrun", false, "enable dry-run (doesn't actually restart nagios)")
var debug = flag.Bool("debug", false, "enable debug to console")
var refresh_time = flag.Int("refresh", 1, "Number of minutes to wait before restarting")
var trigger_file = flag.String("trigger", "/tmp/nagios_config_fail", "path to trigger file for failed config test")
var init_file = flag.String("init-file", "/etc/init.d/nagios3", "path to nagios init script")
var logfile = flag.String("logfile", "/var/log/nagios_watcher.log", "log file to use")

func setup_logging() *os.File {
	logf, err := os.OpenFile(*logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("error opening log file: ", *logfile)
	}
	if *debug {
		multi := io.MultiWriter(logf, os.Stdout)
		log.SetOutput(multi)
	} else {
		log.SetOutput(logf)
	}
	return logf
}

func main() {
	// parse command line args
	flag.Parse()

	// setup logging
	logFile := setup_logging()

	// setup watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// make channel for goroutine
	done := make(chan bool)
	sigs := make(chan os.Signal, 1)
	refresh := make(chan bool, 1)

	// catch USR2 and dump watchers
	signal.Notify(sigs, syscall.SIGUSR2, syscall.SIGHUP)

	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGUSR2:
				log.Println("Dumping watchers: ", sig)
				log.Printf("\n%v\n", watcher)
			case syscall.SIGHUP:
				log.Println("Reloading log file")
				logFile.Close()
				logFile = setup_logging()
			}
		}
	}()

	// run refresh timer in goroutine
	go func() {
		counter := 0
		// create timer
		timer := time.NewTicker(time.Duration(*refresh_time) * time.Minute)
		for {
			select {
			case <-refresh:
				counter += 1
			case <-timer.C:
				if counter > 0 {
					// Test the nagios config
					log.Println("Testing config")
					out, err := NagiosTestConfig()
					if err != nil {
						log.Printf("Nagios Config is broken!\n\n%s", err)
						err = ioutil.WriteFile(*trigger_file, out, 0644)
					} else {
						// if NagiosTestConfig runs successfully and we
						// previously triggered a failure recover here
						if _, err := os.Stat(*trigger_file); err == nil {
							log.Println("Config has recovered")
							err = os.Remove(*trigger_file)
							if err != nil {
								log.Println("failed to cleanup trigger file", err)
							}
						}
						log.Println("Restarting Nagios")
						if !*dryrun {
							out, err = NagiosRestart()
							if err != nil {
								log.Printf("Failed to restart Nagios: %s\n", err)
							}
						}
					}
					// reset counter
					counter = 0
				}
			}
		}
	}()

	walkFn := func(path string, info os.FileInfo, err error) error {
		stat, err := os.Stat(path)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			log.Println("watching directory: ", path)
			watcher.Watch(path)
		}

		if err != nil {
			return err
		}
		return nil
	}

	// filesytem events
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				log.Println("event:", ev)
				// ignore temp files
				// get the file name without path
				_, filename := filepath.Split(ev.Name)
				// get the file name extension
				ext := filepath.Ext(filename)
				// get the first character from file name
				leading_char := string(strings.Split(filename, "")[0])
				if (ext != ".cfg") && (leading_char == ".") {
					log.Println("skipping temporary file")
					break
				}

				if ev.IsDelete() {
					log.Println("removing watcher: ", ev.Name)
					watcher.RemoveWatch(ev.Name)
				} else {
					stat, err := os.Stat(ev.Name)
					if err != nil {
						log.Println(err)
						break
					}
					if stat.IsDir() && ev.IsCreate() {
						// new directory has been created walk
						// this new structure and add any other directories
						err = filepath.Walk(ev.Name, walkFn)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						log.Println("Doing nothing")
					}
				}
				refresh <- true
			case err := <-watcher.Error:
				log.Println("error:", err)
			}
		}
	}()

	err = filepath.Walk(*OrgPath, walkFn)
	if err != nil {
		log.Fatal(err)
	}

	<-done

	watcher.Close()
}
