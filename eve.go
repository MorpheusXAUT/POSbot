package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/morpheusxaut/eveapi"
	"github.com/pkg/errors"
)

func (b *Bot) monitorStarbaseFuel() {

}

func (b *Bot) retrieveStarbaseList() (*eveapi.StarbaseList, error) {
	starbases, err := b.retrieveCachedStarbaseList()
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrap(err, "Failed to retrieve cached starbase list")
	}

	if err != redis.ErrNil && starbases != nil {
		return starbases, nil
	}

	log.Debug("Retrieving starbase list from EVE API")
	starbases, err = b.eve.CorpStarbaseList()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase list from EVE API")
	}

	err = b.cacheStarbaseList(starbases)
	if err != nil {
		log.WithError(err).Warn("Failed to cache starbase list")
	}

	return starbases, nil
}

func (b *Bot) retrieveStarbaseDetails(starbaseID int) (*eveapi.StarbaseDetails, error) {
	starbase, err := b.retrieveCachedStarbaseDetails(starbaseID)
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrap(err, "Failed to retrieve cached starbase details")
	}

	if err != redis.ErrNil && starbase != nil {
		return starbase, nil
	}

	log.WithField("starbaseID", starbaseID).Debug("Retrieving starbase details from EVE API")
	starbase, err = b.eve.CorpStarbaseDetails(starbaseID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase details from EVE API")
	}

	err = b.cacheStarbaseDetails(starbase, starbaseID)
	if err != nil {
		log.WithField("starbaseID", starbaseID).WithError(err).Warn("Failed to cache starbase details")
	}

	return starbase, nil
}
