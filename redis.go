package main

import (
	"encoding/json"
	"fmt"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"strings"
	"time"
)

const (
	RedisKeyStarbaseList    = "posbot:starbase:list"
	RedisKeyStarbaseDetails = "posbot:starbase:details"
	RedisKeyPOS             = "posbot:pos"
	RedisKeyCommandUsage    = "posbot:command:usage"
	RedisKeyCommandError    = "posbot:command:error"
	RedisKeyNotification    = "posbot:notification"
)

func (b *Bot) recordCommandUsage(command string) {
	r := b.redis.Get()
	defer r.Close()

	_, err := r.Do("INCR", fmt.Sprintf("%s:%s", RedisKeyCommandUsage, command))
	if err != nil {
		log.WithField("command", command).WithError(err).Warn("Failed to record command usage in redis")
	}
}

func (b *Bot) recordCommandError(command string) {
	r := b.redis.Get()
	defer r.Close()

	_, err := r.Do("INCR", fmt.Sprintf("%s:%s", RedisKeyCommandError, command))
	if err != nil {
		log.WithField("command", command).WithError(err).Warn("Failed to record command error in redis")
	}
}

func (b *Bot) retrieveCommandStats() (map[string]struct{ Usage, Error int }, error) {
	log.Debug("Retrieving command stats from redis")

	r := b.redis.Get()
	defer r.Close()

	stats := make(map[string]struct{ Usage, Error int })

	usageKeys := make([]string, 0)
	cursor := 0
	for {
		repl, err := redis.Values(r.Do("SCAN", cursor, "MATCH", fmt.Sprintf("%s:*", RedisKeyCommandUsage)))
		if err != nil || len(repl) < 2 {
			return nil, errors.New("Failed to scan command usage keys from redis")
		}

		var keys []string
		if _, err = redis.Scan(repl, &cursor, &keys); err != nil {
			return nil, errors.New("Failed to parse scanned command usage keys from redis")
		}

		usageKeys = append(usageKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	for _, key := range usageKeys {
		if len(key) < 21 {
			continue // missing command name in key
		}

		count, err := redis.Int(r.Do("GET", key))
		if err != nil {
			log.WithField("key", key).WithError(err).Warn("Failed to retrieve command usage count from redis")
			continue
		}

		stats[key[21:]] = struct{ Usage, Error int }{Usage: count, Error: 0}
	}

	errorKeys := make([]string, 0)
	cursor = 0
	for {
		repl, err := redis.Values(r.Do("SCAN", cursor, "MATCH", fmt.Sprintf("%s:*", RedisKeyCommandError)))
		if err != nil || len(repl) < 2 {
			return nil, errors.New("Failed to scan command error keys from redis")
		}

		var keys []string
		if _, err = redis.Scan(repl, &cursor, &keys); err != nil {
			return nil, errors.New("Failed to parse scanned command error keys from redis")
		}

		errorKeys = append(errorKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	for _, key := range errorKeys {
		if len(key) < 21 {
			continue // missing command name in key
		}

		count, err := redis.Int(r.Do("GET", key))
		if err != nil {
			log.WithField("key", key).WithError(err).Warn("Failed to retrieve command error count from redis")
			continue
		}

		s, ok := stats[key[21:]]
		if !ok {
			s = struct{ Usage, Error int }{Usage: 0, Error: 0}
		}
		s.Error = count
		stats[key[21:]] = s
	}

	log.Debug("Received command stats from redis")
	return stats, nil
}

func (b *Bot) retrieveCachedStarbaseList() (*eveapi.StarbaseList, error) {
	log.Debug("Retrieving cached starbase list from redis")

	r := b.redis.Get()
	defer r.Close()

	data, err := redis.Bytes(r.Do("GET", RedisKeyStarbaseList))
	if err == redis.ErrNil {
		log.Debug("Starbase list not cached in redis")
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase list from redis")
	}

	starbases := &eveapi.StarbaseList{}
	if err = json.Unmarshal(data, starbases); err != nil {
		return nil, errors.Wrap(err, "Failed to parse starbase list from redis")
	}

	log.WithFields(logrus.Fields{
		"count":       len(starbases.Starbases),
		"cachedUntil": starbases.CachedUntil,
	}).Debug("Retrieved cached starbase list from redis")
	return starbases, nil
}

func (b *Bot) cacheStarbaseList(starbases *eveapi.StarbaseList) error {
	log.WithFields(logrus.Fields{
		"count":       len(starbases.Starbases),
		"cachedUntil": starbases.CachedUntil,
	}).Debug("Caching starbase list in redis")

	r := b.redis.Get()
	defer r.Close()

	data, err := json.Marshal(starbases)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal starbase list to JSON")
	}

	expiry := starbases.CachedUntil.Time.Sub(time.Now().UTC())
	if expiry.Seconds() <= 0 {
		log.WithFields(logrus.Fields{
			"expiry":      expiry,
			"cachedUntil": starbases.CachedUntil,
		}).Debug("Starbase list has expiry equal or below 0 seconds, not caching")
		return nil
	}

	reply, err := redis.String(r.Do("SET", RedisKeyStarbaseList, data, "EX", int(expiry.Seconds())))
	if err != nil {
		return errors.Wrap(err, "Failed to store starbase list in redis")
	} else if !strings.EqualFold(reply, "OK") {
		return errors.New("Failed to store starbase list in redis")
	}

	log.WithFields(logrus.Fields{
		"count":       len(starbases.Starbases),
		"cachedUntil": starbases.CachedUntil,
	}).Debug("Cached starbase list in redis")
	return nil
}

func (b *Bot) retrieveCachedStarbaseDetails(starbaseID int) (*eveapi.StarbaseDetails, error) {
	log.WithField("starbaseID", starbaseID).Debug("Retrieving cached starbase details from redis")

	r := b.redis.Get()
	defer r.Close()

	data, err := redis.Bytes(r.Do("GET", fmt.Sprintf("%s:%d", RedisKeyStarbaseDetails, starbaseID)))
	if err == redis.ErrNil {
		log.WithField("starbaseID", starbaseID).Debug("Starbase details not cached in redis")
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase details from redis")
	}

	starbase := &eveapi.StarbaseDetails{}
	if err = json.Unmarshal(data, starbase); err != nil {
		return nil, errors.Wrap(err, "Failed to parse starbase details from redis")
	}

	log.WithFields(logrus.Fields{
		"starbaseID":  starbaseID,
		"cachedUntil": starbase.CachedUntil,
	}).Debug("Retrieved cached starbase details from redis")
	return starbase, nil
}

func (b *Bot) cacheStarbaseDetails(starbase *eveapi.StarbaseDetails, starbaseID int) error {
	log.WithFields(logrus.Fields{
		"starbaseID":  starbaseID,
		"cachedUntil": starbase.CachedUntil,
	}).Debug("Caching starbase details in redis")

	r := b.redis.Get()
	defer r.Close()

	data, err := json.Marshal(starbase)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal starbase details to JSON")
	}

	expiry := starbase.CachedUntil.Time.Sub(time.Now().UTC())
	if expiry.Seconds() <= 0 {
		log.WithFields(logrus.Fields{
			"expiry":      expiry,
			"cachedUntil": starbase.CachedUntil,
		}).Debug("Starbase details have expiry equal or below 0 seconds, not caching")
		return nil
	}

	reply, err := redis.String(r.Do("SET", fmt.Sprintf("%s:%d", RedisKeyStarbaseDetails, starbaseID), data, "EX", int(expiry.Seconds())))
	if err != nil {
		return errors.Wrap(err, "Failed to store starbase details in redis")
	} else if !strings.EqualFold(reply, "OK") {
		return errors.New("Failed to store starbase details in redis")
	}

	log.WithFields(logrus.Fields{
		"starbaseID":  starbaseID,
		"cachedUntil": starbase.CachedUntil,
	}).Debug("Cached starbase details in redis")
	return nil
}

func (b *Bot) retrieveCachedPOS(starbaseID int) (*POS, error) {
	log.WithField("starbaseID", starbaseID).Debug("Retrieving cached POS from redis")

	r := b.redis.Get()
	defer r.Close()

	data, err := redis.Bytes(r.Do("GET", fmt.Sprintf("%s:%d", RedisKeyPOS, starbaseID)))
	if err == redis.ErrNil {
		log.WithField("starbaseID", starbaseID).Debug("POS not cached in redis")
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve POS from redis")
	}

	pos := &POS{}
	if err = json.Unmarshal(data, pos); err != nil {
		return nil, errors.Wrap(err, "Failed to parse POS from redis")
	}

	log.WithFields(logrus.Fields{
		"starbaseID":  pos.ID,
		"cachedUntil": pos.CachedUntil,
	}).Debug("Retrieved cached POS from redis")
	return pos, nil
}

func (b *Bot) cachePOS(pos *POS) error {
	log.WithFields(logrus.Fields{
		"starbaseID":  pos.ID,
		"cachedUntil": pos.CachedUntil,
	}).Debug("Caching POS in redis")

	r := b.redis.Get()
	defer r.Close()

	data, err := json.Marshal(pos)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal starbase list to JSON")
	}

	expiry := pos.CachedUntil.Sub(time.Now().UTC())
	if expiry.Seconds() <= 0 {
		log.WithFields(logrus.Fields{
			"expiry":      expiry,
			"cachedUntil": pos.CachedUntil,
		}).Debug("POS has expiry equal or below 0 seconds, not caching")
		return nil
	}

	reply, err := redis.String(r.Do("SET", fmt.Sprintf("%s:%d", RedisKeyPOS, pos.ID), data, "EX", int(expiry.Seconds())))
	if err != nil {
		return errors.Wrap(err, "Failed to store POS in redis")
	} else if !strings.EqualFold(reply, "OK") {
		return errors.New("Failed to store POS in redis")
	}

	log.WithFields(logrus.Fields{
		"starbaseID":  pos.ID,
		"cachedUntil": pos.CachedUntil,
	}).Debug("Cached POS in redis")
	return nil
}

func (b *Bot) recordNotification(starbaseID int, fuelTypeID int, notification int) {
	r := b.redis.Get()
	defer r.Close()

	expiry := 3600
	if notification == 1 {
		expiry = b.config.Discord.NotificationWarning
	} else if notification == 2 {
		expiry = b.config.Discord.NotificationCritical
	}
	_, err := r.Do("SET", fmt.Sprintf("%s:%d:%d", RedisKeyNotification, starbaseID, fuelTypeID), notification, "EX", expiry)
	if err != nil {
		log.WithFields(logrus.Fields{
			"starbaseID":   starbaseID,
			"fuelTypeID":   fuelTypeID,
			"notification": notification,
		}).WithError(err).Warn("Failed to record notification in redis")
	}
}

func (b *Bot) shouldSendNotification(starbaseID int, fuelTypeID, notification int) bool {
	r := b.redis.Get()
	defer r.Close()

	sent, err := redis.Int(r.Do("GET", fmt.Sprintf("%s:%d:%d", RedisKeyNotification, starbaseID, fuelTypeID)))
	if err == redis.ErrNil {
		b.recordNotification(starbaseID, fuelTypeID, notification)
		return true
	} else if err != nil {
		log.WithFields(logrus.Fields{
			"starbaseID":   starbaseID,
			"fuelTypeID":   fuelTypeID,
			"notification": notification,
		}).WithError(err).Warn("Failed to check notification in redis")
		return true
	}

	if sent < notification {
		b.recordNotification(starbaseID, fuelTypeID, notification)
		return true
	}

	return false
}
