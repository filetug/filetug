package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/filetug/filetug/pkg/filetug"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/profiling"
	"github.com/rivo/tview"
)

var (
	cpuProfile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile = flag.String("memprofile", "", "write memory profile to `file`")
	pprofAddr  = flag.String("pprof", "", "start pprof http server on `address` (e.g. localhost:6060)")
)

var httpListenAndServe = http.ListenAndServe
var osExit = os.Exit
var pprofStopCPUProfile = pprof.StopCPUProfile

func main() {
	app := newFileTugApp()
	run(app)
}

func newFileTugApp() (app navigator.App) {
	flag.Parse()
	initialPath = ""
	if flag.NArg() > 0 {
		initialPath = flag.Arg(0)
	}

	if *pprofAddr != "" {
		go func() {
			err := httpListenAndServe(*pprofAddr, nil)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "pprof server error: %v\n", err)
			}
		}()
	}

	defer func() {
		if r := recover(); r != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Recovered from panic: %v\n", r)
			pprofStopCPUProfile()
			osExit(1)
		}
	}()

	if *cpuProfile != "" {
		stopCPUProfiling := profiling.DoCPUProfiling(*cpuProfile)
		defer stopCPUProfiling()
	}

	if *memProfile != "" {
		stopMemProfiling := profiling.DoMemProfiling(*memProfile)
		defer stopMemProfiling()
	}

	app = newApp()
	return
}

var setupApp = filetug.SetupApp
var setupAppAtPath = filetug.SetupAppAtPath
var initialPath string

var newApp = func() navigator.App {
	tvApp := tview.NewApplication()
	app := navigator.NewApp(tvApp)
	if initialPath == "" {
		setupApp(app)
	} else {
		setupAppAtPath(app, initialPath)
	}
	return app
}

type application interface{ Run() error }

var run = func(app application) {
	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}
