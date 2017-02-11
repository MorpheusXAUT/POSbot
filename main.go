package main

import (
	"flag"
	"github.com/MorpheusXAUT/logzruz"
	"github.com/Sirupsen/logrus"
	"github.com/rifflock/lfshook"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	Version string = "1.0.1"
)

var (
	log      *logrus.Entry
	shutdown chan int
)

func init() {
	log = logrus.WithField("uninitialised", true)
	shutdown = make(chan int, 1)
}

func main() {
	var (
		configFile  = flag.String("config", "posbot.cfg", "Path to POSbot config file")
		logLevel    = flag.Int("log", 4, "Log level (0-5) for message severity to display. Higher level displays more messages")
		environment = flag.String("env", "dev", "Environment to run bot in (dev/prod)")
	)
	flag.Parse()

	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.Level(*logLevel))
	if strings.EqualFold(*environment, "prod") || strings.EqualFold(*environment, "production") {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
	log = logrus.WithFields(logrus.Fields{
		"app":     "POSbot",
		"version": Version,
	})

	if len(*configFile) <= 0 {
		log.Fatal("No config file provided, required")
		os.Exit(1)
		return
	}

	config, err := parseConfigFile(*configFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to parse config file")
		os.Exit(1)
		return
	}

	if config.Logging.Files.Enabled {
		pathMap := make(lfshook.PathMap)
		for lvl, path := range config.Logging.Files.Paths {
			if len(path) == 0 {
				log.WithField("lvl", lvl).Warn("Log file path is empty")
				continue
			}

			level, err := logrus.ParseLevel(lvl)
			if err != nil {
				log.WithFields(logrus.Fields{
					"lvl":  lvl,
					"path": path,
				}).WithError(err).Warn("Failed to parse level for config file")
				continue
			}

			pathMap[level] = path
		}

		logrus.AddHook(lfshook.NewHook(pathMap))
		log.Debug("Initialised local file logging")
	}

	if config.Logging.Logzio.Enabled {
		logzCtx := logrus.Fields{
			"app":         "POSbot",
			"version":     Version,
			"environment": *environment,
		}

		for k, v := range config.Logging.Logzio.Context {
			logzCtx[k] = v
		}

		logz, err := logzruz.NewHook(logzruz.HookOptions{
			App:         "POSbot",
			BufferCount: 10,
			Context:     logzCtx,
			Token:       config.Logging.Logzio.Token,
		})
		if err != nil {
			log.WithError(err).Error("Failed to initialise logz.io hook")
		} else {
			logrus.AddHook(logz)
			log.Debug("Initialised logz.io hook")
		}
	}

	log.Info("POSbot startup initiated")
	log.WithField("configFile", *configFile).Debug("Creating new bot from config file")

	bot, err := NewBot(config)
	if err != nil {
		log.WithField("err", err).Fatal("Failed to create bot")
		os.Exit(1)
		return
	}

	log.Info("POSbot startup complete")

	interrupt := make(chan os.Signal, 3)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-interrupt
		log.WithField("signal", sig).Debug("Received signal, shutting down")
		shutdown <- 2
	}()

	code := <-shutdown

	log.WithField("code", code).Info("POSbot shutting down")

	bot.Shutdown()
	os.Exit(code)
	return
}
