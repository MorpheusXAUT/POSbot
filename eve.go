package main

import (
	"github.com/garyburd/redigo/redis"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/pkg/errors"
)

func (b *Bot) monitorStarbaseFuel() {

}

func (b *Bot) isStarbaseMonitored(starbaseID int) bool {
	for _, id := range b.config.EVE.IgnoredStarbases {
		if starbaseID == id {
			return false
		}
	}
	return true
}

func (b *Bot) getMonitoredStarbaseIDs() ([]int, error) {
	starbases, err := b.retrieveStarbaseList()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase list")
	}

	monitored := make([]int, 0)
	for _, starbase := range starbases.Starbases {
		if b.isStarbaseMonitored(starbase.ID) {
			monitored = append(monitored, starbase.ID)
		}
	}

	return monitored, nil
}

func (b *Bot) retrieveStarbaseList() (*eveapi.StarbaseList, error) {
	starbases, err := b.retrieveCachedStarbaseList()
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrap(err, "Failed to retrieve cached starbase list")
	}

	if err != redis.ErrNil && starbases != nil {
		log.Debug("Retrieved starbase list from cache")
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

func (b *Bot) getCorporationNameFromID(corporationID int) (string, error) {
	names, _, err := b.esi.CorporationApi.GetCorporationsNames([]int64{int64(corporationID)}, nil)
	if err != nil {
		return "", errors.Wrap(err, "Failed to get corporation name from ID")
	}

	corporationName := ""
	for _, name := range names {
		if name.CorporationId == int32(corporationID) {
			corporationName = name.CorporationName
		}
	}

	if len(corporationName) == 0 {
		log.WithField("corporationID", corporationID).Warn("Did not find name for corporation")
		return "", errors.New("Did not find name for corporation")
	}

	return corporationName, nil
}