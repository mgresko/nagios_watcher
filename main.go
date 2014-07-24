package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/howeyc/fsnotify"
)

func main() {
	OrgPath := flag.String("config-dir", "/etc/nagios.sync", "path to nagios config files")
	debug := flag.Bool("debug", false, "enable debug logging")
	refresh_time := flag.Int("refresh", 1, "Number of minutes to wait before restarting")
	trigger_file := flag.String("trigger", "/etc/nagios.sync/config_fail", "path to trigger file for failed config test")
	flag.Parse()

	if *debug {
		log.Println("debug logging enabled")
	}

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
	signal.Notify(sigs, syscall.SIGUSR2)

	go func() {
		for {
			select {
			case sig := <-sigs:
				log.Println("Dumping watchers: ", sig)
				log.Printf("\n%v\n", watcher)
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
				log.Println("timer triggered")
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
						out, err = NagiosRestart()
						if err != nil {
							log.Printf("Failed to restart Nagios: %s\n", err)
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
						log.Println("Not right")
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
