package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/filetug/filetug/pkg/filetug"
	"github.com/filetug/filetug/pkg/filetug/charmui"
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/profiling"
	"github.com/rivo/tview"
)

var (
	cpuProfile  = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile  = flag.String("memprofile", "", "write memory profile to `file`")
	pprofAddr   = flag.String("pprof", "", "start pprof http server on `address` (e.g. localhost:6060)")
	showVersion = flag.Bool("version", false, "print FileTug version and exit")
)

var httpListenAndServe = http.ListenAndServe
var osExit = os.Exit
var pprofStopCPUProfile = pprof.StopCPUProfile
var version = "dev"
var versionOutput io.Writer = os.Stdout

func main() {
	app := newMVPFileTugApp()
	if app != nil {
		run(app)
	}
}

// charmApplication keeps runtime profiling alive for the direct Bubble Tea
// process, then relies on Bubble Tea v2 to restore terminal state on exit.
type charmApplication struct {
	path     string
	cleanups []func()
}

func (a charmApplication) Run() error {
	defer func() {
		for index := len(a.cleanups) - 1; index >= 0; index-- {
			a.cleanups[index]()
		}
	}()
	return charmui.RunLocal(a.path)
}

func newMVPFileTugApp() (app application) {
	flag.Parse()
	if *showVersion {
		_, _ = fmt.Fprintf(versionOutput, "filetug %s\n", version)
		return nil
	}
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
	cleanups := make([]func(), 0, 2)
	if *cpuProfile != "" {
		cleanup := profiling.DoCPUProfiling(*cpuProfile)
		cleanups = append(cleanups, cleanup)
	}
	if *memProfile != "" {
		cleanup := profiling.DoMemProfiling(*memProfile)
		cleanups = append(cleanups, cleanup)
	}
	return charmApplication{path: initialPath, cleanups: cleanups}
}

func newFileTugApp() (app navigator.App) {
	flag.Parse()
	if *showVersion {
		_, _ = fmt.Fprintf(versionOutput, "filetug %s\n", version)
		return nil
	}
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
