package main

import (
	"fmt"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/MorpheusXAUT/evesi"
	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gregjones/httpcache"
	httpredis "github.com/gregjones/httpcache/redis"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

const (
	UserAgent string = "POSbot v" + Version + " - github.com/MorpheusXAUT/POSbot"
)

type Bot struct {
	discord *discordgo.Session
	esi     *evesi.APIClient
	eve     eveapi.API
	http    *http.Client
	mysql   *sqlx.DB
	redis   *redis.Pool

	config    *Config
	startTime time.Time
	stop      chan bool
	ticker    *time.Ticker
}

func NewBot(config *Config) (*Bot, error) {
	bot := &Bot{
		config:    config,
		startTime: time.Now().UTC(),
		stop:      make(chan bool, 1),
	}

	var err error

	log.Info("Initialising Redis connection")
	redisOptions := make([]redis.DialOption, 0)
	if len(bot.config.Redis.Password) > 0 {
		redisOptions = append(redisOptions, redis.DialPassword(bot.config.Redis.Password))
	}
	if bot.config.Redis.Database >= 0 {
		redisOptions = append(redisOptions, redis.DialDatabase(bot.config.Redis.Database))
	}

	bot.redis = &redis.Pool{
		MaxIdle:     50,
		MaxActive:   0,
		Wait:        false,
		IdleTimeout: 90 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", bot.config.Redis.Address, redisOptions...)
		},
	}

	r := bot.redis.Get()

	_, err = r.Do("PING")
	r.Close()
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to ping Redis server")
	}

	log.Info("Creating httpcache client")
	transport := httpcache.NewTransport(httpredis.NewWithClient(bot.redis.Get()))
	bot.http = &http.Client{
		Transport: transport,
		Timeout:   time.Second * 90,
	}

	log.Info("Initialising ESI connection")
	bot.esi = evesi.NewAPIClient(bot.http, UserAgent)

	log.Info("Initialising EVE connection")
	bot.eve = eveapi.API{
		Server: eveapi.Tranquility,
		APIKey: eveapi.Key{
			ID:    bot.config.EVE.KeyID,
			VCode: bot.config.EVE.KeyVCode,
		},
		UserAgent: UserAgent,
		Client:    bot.http,
		Debug:     false,
	}

	_, err = bot.eve.ServerStatus()
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to query EVE server status")
	}

	log.Info("Initialising MySQL connection")
	bot.mysql, err = sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", bot.config.MySQL.Username, bot.config.MySQL.Password, bot.config.MySQL.Address, bot.config.MySQL.Database))
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to open connection to MySQL server")
	}

	err = bot.mysql.Ping()
	if err != nil {
		bot.redis.Close()
		return nil, errors.Wrap(err, "Failed to ping MySQL server")
	}

	query, args, err := sqlx.In("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name IN (?)", bot.config.MySQL.Database, mysqlRequiredTableNames)
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to prepare required MySQL tables check")
	}

	var tableCount int
	err = bot.mysql.Get(&tableCount, query, args...)
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to check required MySQL tables")
	}
	if tableCount != len(mysqlRequiredTableNames) {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Missing required MySQL tables")
	}

	log.Info("Initialising Discord connection")
	bot.discord, err = discordgo.New(fmt.Sprintf("Bot %s", bot.config.Discord.Token))
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to create Discord session")
	}

	bot.discord.AddHandler(bot.onDiscordReady)
	bot.discord.AddHandler(bot.onDiscordGuildCreate)
	bot.discord.AddHandler(bot.onDiscordMessageCreate)

	err = bot.discord.Open()
	if err != nil {
		bot.redis.Close()
		bot.mysql.Close()
		return nil, errors.Wrap(err, "Failed to open Discord session")
	}

	go bot.monitoringLoop()
	bot.ticker = time.NewTicker(time.Second * time.Duration(bot.config.EVE.MonitorInterval))
	go bot.checkStarbaseFuel() // trigger once to avoid having to wait MonitorInterval seconds first

	return bot, nil
}

func NewBotFromConfigFile(configFile string) (*Bot, error) {
	config, err := parseConfigFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create bot from config file")
	}

	return NewBot(config)
}

func (b *Bot) Shutdown() {
	log.Info("Clean bot shutdown initiated")

	b.ticker.Stop()
	b.stop <- true

	if b.config.Discord.Debug {
		b.discord.ChannelMessageSend(b.config.Discord.ChannelID, ":robot: POSbot shutting down :skull_crossbones:")
	}

	b.discord.Close()
	b.redis.Close()
	b.mysql.Close()

	log.WithField("logzruzForceFlush", true).Info("Clean bot shutdown completed")
}

func (b *Bot) monitoringLoop() {
	for {
		select {
		case <-b.stop:
			log.Debug("Stopping monitoring loop")
			return
		case <-b.ticker.C:
			b.checkStarbaseFuel()
			break
		}
	}
}
