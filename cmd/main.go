package main

import (
	"flag"
	"fmt"
	"github.com/zekihan/traefik-rate-limit/cmd/client"
	"github.com/zekihan/traefik-rate-limit/cmd/healthCheck"
	"github.com/zekihan/traefik-rate-limit/cmd/server"
	"github.com/zekihan/traefik-rate-limit/internal/config"
	"github.com/zekihan/traefik-rate-limit/internal/utils"
	"log"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to `file`")
	socketPath = flag.String("socket", "", "path to unix domain socket")
	logLevel   = flag.String("logLevel", "", "log level (debug, info, warn, error)") // Added logLevel flag
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected a subcommand")
		os.Exit(1)
	}

	cmd := os.Args[1]
	os.Args = os.Args[1:]
	flag.Parse()

	cfg := config.GetConfig()
	if *socketPath != "" {
		cfg.SocketPath = *socketPath
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}

	setLogger(cmd)
	cpu := setCpuProfile()
	defer cpu()

	switch cmd {
	case string(CommandServer):
		server.Run(cfg.SocketPath)
	case string(CommandClient):
		client.Run(cfg.SocketPath)
	case string(CommandHealthCheck):
		healthCheck.Run(cfg.SocketPath)
	case string(CommandVersion):
		fmt.Printf("%s\n%s\n", utils.Version, utils.GetStartupInfo())
	case string(CommandHelp):
		printHelp()
	default:
		printUnknownCommand(cmd)
		os.Exit(1)
	}

	mem := setMemoryProfile()
	defer mem()
}

func printHelp() {
	fmt.Printf("Available commands: [%s] [%s] [%s] [%s] [%s]\n", string(CommandServer), string(CommandClient), string(CommandHealthCheck), string(CommandVersion), string(CommandHelp))
}

func printUnknownCommand(cmd string) {
	fmt.Printf("Unknown command: %s\n", cmd)
	fmt.Printf("Available commands: [%s] [%s] [%s] [%s] [%s]\n", string(CommandServer), string(CommandClient), string(CommandHealthCheck), string(CommandVersion), string(CommandHelp))
}

func setLogger(cmd string) {
	var level slog.Level
	switch strings.ToLower(config.GetConfig().LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		log.Printf("Invalid log level '%s', defaulting to 'info'", *logLevel)
		level = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       level,
		ReplaceAttr: nil,
	}))
	logger = logger.With(slog.String("app", cmd))
	slog.SetDefault(logger)
}

func setCpuProfile() func() {
	deferred := make([]func(), 0)
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		deferred = append(deferred, func() {
			f.Close()
		})
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		deferred = append(deferred, func() {
			pprof.StopCPUProfile()
		})
	}
	return func() {
		for _, f := range deferred {
			f()
		}
	}
}

func setMemoryProfile() func() {
	deferred := make([]func(), 0)
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		deferred = append(deferred, func() {
			f.Close()
		}) // error handling omitted for example
		runtime.GC() // get up-to-date statistics
		// Lookup("allocs") creates a profile similar to go test -memprofile.
		// Alternatively, use Lookup("heap") for a profile
		// that has inuse_space as the default index.
		if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
	return func() {
		for _, f := range deferred {
			f()
		}
	}
}
