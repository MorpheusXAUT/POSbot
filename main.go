package main

import (
	"flag"
	"fmt"
	"github.com/MorpheusXAUT/POSbot/util"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const (
	Version string = "0.0.1"
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

	log.Info("POSbot startup initiated")
	log.WithField("configFile", *configFile).Debug("Creating new bot from config file")

	bot, err := NewBotFromConfigFile(*configFile)
	if err != nil {
		log.WithField("err", err).Fatal("Failed to create bot from config file")
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

func monitorStarbaseFuel(eve eveapi.API, discord *discordgo.Session) {
	starbases, err := eve.CorpStarbaseList()
	if err != nil {
		fmt.Printf("Failed to load corp starbase list: %v\n", err)
		return
	}

	fmt.Printf("Result cached until %v\n", starbases.CachedUntil)

	for _, base := range starbases.Starbases {
		starbase, err := eve.CorpStarbaseDetails(base.ID)
		if err != nil {
			fmt.Printf("Failed to load details for starbase #%d with state %s\n", base.ID, base.State)
			continue
		}

		requiredFuel := util.FuelRequiredForStarbase(base.TypeID)
		fmt.Printf("Starbase #%d has state %s with usage flags %d\n", base.ID, base.State, starbase.GeneralSettings.UsageFlags)
		for _, fuel := range starbase.Fuel {
			fmt.Printf("Starbase #%d has %d units of fuel #%d\n", base.ID, fuel.Quantity, fuel.TypeID)
			if fuel.TypeID == requiredFuel.TypeID {
				fmt.Printf("Required %d units per hour, %d hours left\n", requiredFuel.Amount, fuel.Quantity/requiredFuel.Amount)
			}
		}
	}
}
