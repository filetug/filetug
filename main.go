package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/datatug/filetug/pkg/filetug"
	"github.com/rivo/tview"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
	pprofAddr  = flag.String("pprof", "", "start pprof http server on `address` (e.g. localhost:6060)")
)

func main() {
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(*pprofAddr, nil))
		}()
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer func() {
			_ = f.Close() // error handling omitted for brevity
		}()
		if err = pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	app := newApp()
	if *memprofile != "" {
		go func() {
			writeMemProfile := func() {
				f, err := os.Create(*memprofile)
				if err != nil {
					log.Fatal("could not create memory profile: ", err)
				}
				defer func() {
					_ = f.Close() // error handling omitted for brevity
				}()
				runtime.GC() // get up-to-date statistics
				if err = pprof.WriteHeapProfile(f); err != nil {
					log.Fatal("could not write memory profile: ", err)
				}
			}
			for {
				time.Sleep(time.Second)
				writeMemProfile()
			}
		}()
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			defer pprof.StopCPUProfile()
		}
	}()
	run(app)

}

var setupApp = filetug.SetupApp

var newApp = func() *tview.Application {
	app := tview.NewApplication()
	setupApp(app)
	return app
}

type application interface{ Run() error }

var run = func(app application) {
	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
