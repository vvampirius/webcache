package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

const VERSION = `0.2`

var (
	ErrorLog = log.New(os.Stderr, `error#`, log.Lshortfile)
	DebugLog = log.New(os.Stdout, `debug#`, log.Lshortfile)
)

func helpText() {
	fmt.Println(`# https://github.com/vvampirius/webcache`)
	fmt.Printf("\nUsage: %s [options] <path>\n\n", os.Args[0])
	flag.PrintDefaults()
}

func Pong(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `PONG`)
}

func main() {
	listen := flag.String("l", ":8080", "Listen <[address]:port>")
	passwordsFile := flag.String("p", "", "Passwords JSON file")
	help := flag.Bool("h", false, "print this help")
	ver := flag.Bool("v", false, "Show version")
	flag.Parse()

	if *help {
		helpText()
		os.Exit(0)
	}

	if *ver {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	storagePath := flag.Arg(0)
	if storagePath == `` {
		helpText()
		os.Exit(1)
	}

	fmt.Println("Starting version %s...", VERSION)

	if err := os.MkdirAll(storagePath, 0700); err != nil {
		ErrorLog.Fatalln(err.Error())
	}

	insecure := false
	auth := NewAuth()
	if *passwordsFile != `` {
		if err := auth.LoadPasswords(*passwordsFile); err != nil {
			ErrorLog.Fatalln(err.Error())
		}
	} else {
		fmt.Println(`Warning! Passwords file not specified! Running in insecure mode.`)
		insecure = true
	}

	if err := ScheduleCacheRemove(storagePath); err != nil {
		os.Exit(1)
	}

	core := NewCore(storagePath, auth, insecure)

	server := http.Server{Addr: *listen}
	http.HandleFunc(`/ping`, Pong)
	http.HandleFunc("/", core.MainHttpHandler)
	if err := server.ListenAndServe(); err != nil {
		ErrorLog.Fatalln(err.Error())
	}
}
