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
	"github.com/filetug/filetug/pkg/filetug/navigator"
	"github.com/filetug/filetug/pkg/profiling"
	"github.com/rivo/tview"
	"github.com/strongo/buildinfo"
)

var (
	cpuProfile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	memProfile = flag.String("memprofile", "", "write memory profile to `file`")
	pprofAddr  = flag.String("pprof", "", "start pprof http server on `address` (e.g. localhost:6060)")
)

// showVersion backs both the "-version" flag and its "-v" shorthand; flag
// has no native alias syntax, so both flag.BoolVar registrations below bind
// this one variable.
var showVersion bool

func init() {
	flag.BoolVar(&showVersion, "version", false, "print FileTug version and exit")
	flag.BoolVar(&showVersion, "v", false, "print FileTug version and exit (shorthand)")
}

var httpListenAndServe = http.ListenAndServe
var osExit = os.Exit
var pprofStopCPUProfile = pprof.StopCPUProfile
var versionOutput io.Writer = os.Stdout

func main() {
	app := newFileTugApp()
	if app != nil {
		run(app)
	}
}

func newFileTugApp() (app navigator.App) {
	flag.Parse()
	if showVersion {
		_, _ = fmt.Fprintf(versionOutput, "%s\n", buildinfo.Get("filetug").Short())
		return nil
	}
	initialPath = ""
	if flag.NArg() > 0 {
		// "filetug version" (no other positional args) prints the long
		// build identity and exits, matching the fleet's `<binary> version`
		// convention. This shadows a real file or directory literally
		// named "version" opened via a bare positional arg; a user with
		// such a path must invoke it as e.g. `filetug ./version` instead.
		if flag.NArg() == 1 && flag.Arg(0) == "version" {
			_, _ = fmt.Fprintf(versionOutput, "%s\n", buildinfo.Get("filetug").Long())
			return nil
		}
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
