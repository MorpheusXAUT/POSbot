package main

import (
	"encoding/json"
	"fmt"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"strings"
)

const (
	RedisKeyStarbaseList    = "posbotStarbaseList"
	RedisKeyStarbaseDetails = "posbotStarbaseDetails"
)

func (b *Bot) recordCommandUsage(command string) {
	r := b.redis.Get()
	defer r.Close()

	_, err := r.Do("INCR", fmt.Sprintf("posbotCommandUsage%s", command))
	if err != nil {
		log.WithField("command", command).WithError(err).Warn("Failed to record command usage in redis")
	}
}

func (b *Bot) recordCommandError(command string) {
	r := b.redis.Get()
	defer r.Close()

	_, err := r.Do("INCR", fmt.Sprintf("posbotCommandError%s", command))
	if err != nil {
		log.WithField("command", command).WithError(err).Warn("Failed to record command error in redis")
	}
}

func (b *Bot) retrieveCommandStats() (map[string]struct{ Usage, Error int }, error) {
	r := b.redis.Get()
	defer r.Close()

	stats := make(map[string]struct{ Usage, Error int })

	usage, err := redis.Strings(r.Do("KEYS", "posbotCommandUsage*"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get command usage keys from redis")
	}

	for _, key := range usage {
		if len(key) <= 18 {
			// Missing command "name" in key
			continue
		}

		count, err := redis.Int(r.Do("GET", key))
		if err != nil {
			log.WithField("key", key).WithError(err).Warn("Failed to retrieve command usage count from redis")
			continue
		}

		stats[key[18:]] = struct{ Usage, Error int }{Usage: count, Error: 0}
	}

	error, err := redis.Strings(r.Do("KEYS", "posbotCommandError*"))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get command error keys from redis")
	}

	for _, key := range error {
		if len(key) <= 18 {
			// Missing command "name" in key
			continue
		}

		count, err := redis.Int(r.Do("GET", key))
		if err != nil {
			log.WithField("key", key).WithError(err).Warn("Failed to retrieve command error count from redis")
			continue
		}

		s, ok := stats[key[18:]]
		if !ok {
			s = struct{ Usage, Error int }{Usage: 0, Error: 0}
		}
		s.Error = count
		stats[key[18:]] = s
	}

	return stats, nil
}

func (b *Bot) retrieveCachedStarbaseList() (*eveapi.StarbaseList, error) {
	r := b.redis.Get()
	defer r.Close()

	data, err := redis.Bytes(r.Do("GET", RedisKeyStarbaseList))
	if err == redis.ErrNil {
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase list from redis")
	}

	starbases := &eveapi.StarbaseList{}
	if err = json.Unmarshal(data, starbases); err != nil {
		log.Debug(err)
		return nil, errors.Wrap(err, "Failed to parse starbase list from redis")
	}

	return starbases, nil
}

func (b *Bot) cacheStarbaseList(starbases *eveapi.StarbaseList) error {
	r := b.redis.Get()
	defer r.Close()

	data, err := json.Marshal(starbases)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal starbase list to JSON")
	}

	expiry := starbases.CachedUntil.Sub(starbases.CurrentTime.Time)
	if expiry.Seconds() <= 0 {
		log.WithField("expiry", expiry).Debug("Starbase list has expiry equal or below 0 seconds, not caching")
		return nil
	}

	reply, err := redis.String(r.Do("SET", RedisKeyStarbaseList, data, "EX", expiry.Seconds()))
	if err != nil {
		return errors.Wrap(err, "Failed to store starbase list in redis")
	} else if !strings.EqualFold(reply, "OK") {
		return errors.New("Failed to store starbase list in redis")
	}

	return nil
}

func (b *Bot) retrieveCachedStarbaseDetails(starbaseID int) (*eveapi.StarbaseDetails, error) {
	r := b.redis.Get()
	defer r.Close()

	data, err := redis.Bytes(r.Do("GET", fmt.Sprintf("%s%d", RedisKeyStarbaseDetails, starbaseID)))
	if err == redis.ErrNil {
		return nil, err
	} else if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase details from redis")
	}

	starbase := &eveapi.StarbaseDetails{}
	if err = json.Unmarshal(data, starbase); err != nil {
		log.Debug(err)
		return nil, errors.Wrap(err, "Failed to parse starbase details from redis")
	}

	return starbase, nil
}

func (b *Bot) cacheStarbaseDetails(starbase *eveapi.StarbaseDetails, starbaseID int) error {
	r := b.redis.Get()
	defer r.Close()

	data, err := json.Marshal(starbase)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal starbase details to JSON")
	}

	expiry := starbase.CachedUntil.Sub(starbase.CurrentTime.Time)
	if expiry.Seconds() <= 0 {
		log.WithField("expiry", expiry).Debug("Starbase details has expiry equal or below 0 seconds, not caching")
		return nil
	}

	reply, err := redis.String(r.Do("SET", fmt.Sprintf("%s%d", RedisKeyStarbaseDetails, starbaseID), data, "EX", expiry.Seconds()))
	if err != nil {
		return errors.Wrap(err, "Failed to store starbase details in redis")
	} else if !strings.EqualFold(reply, "OK") {
		return errors.New("Failed to store starbase details in redis")
	}

	return nil
}
