package main

import (
	"fmt"
	"github.com/MorpheusXAUT/durafmt"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
	"github.com/pkg/errors"
	"strings"
	"time"
)

func (b *Bot) checkStarbaseFuel() {
	log.Info("Checking starbase fuel")

	err := b.updateMonitoredStarbaseDetails()
	if err != nil {
		log.WithError(err).Error("Failed to update monitored starbase details")
		if b.config.Discord.Verbose {
			b.discord.ChannelMessage(b.config.Discord.ChannelID, ":warning: There was an error updating monitored POSes :warning:")
		}
		return
	}

	monitored, err := b.getMonitoredStarbaseIDs()
	if err != nil {
		log.WithError(err).Error("Failed to retrieve monitored starbases")
		if b.config.Discord.Verbose {
			b.discord.ChannelMessage(b.config.Discord.ChannelID, ":warning: There was an error retrieving monitored POSes :warning:")
		}
		return
	}

	for _, starbaseID := range monitored {
		log.WithField("starbaseID", starbaseID).Debug("Checking POS fuel status")

		pos, err := b.getPOSFromStarbaseID(starbaseID)
		if err != nil {
			log.WithField("starbaseID", starbaseID).WithError(err).Warn("Failed to get POS from starbaseID")
			if b.config.Discord.Verbose {
				b.discord.ChannelMessage(b.config.Discord.ChannelID, fmt.Sprintf(":warning: There was an error retrieving POS #%d :warning:", starbaseID))
			}
			continue
		}

		for _, fuel := range pos.Fuel {
			if !fuel.ConstantlyRequired {
				continue
			}

			remaining, err := durafmt.ParseString(fmt.Sprintf("%fh", fuel.TimeRemaining))
			if err != nil {
				log.WithFields(logrus.Fields{
					"starbaseID": pos.ID,
					"fuelTypeID": fuel.TypeID,
				}).WithError(err).Warn("Failed to parse remaining fuel duration")
				if b.config.Discord.Verbose {
					b.discord.ChannelMessage(b.config.Discord.ChannelID, fmt.Sprintf(":warning: There was an error parsing remaining fuel for POS #%d :warning:", starbaseID))
				}
				continue
			}

			if int(fuel.TimeRemaining) <= b.config.EVE.FuelCriticalThreshold {
				if b.shouldSendNotification(pos.ID, fuel.TypeID, 2) {
					b.discord.ChannelMessageSend(b.config.Discord.ChannelID, fmt.Sprintf("@everyone :rotating_light: POS at **%s** (owned by %s) only has __**%s**__ of fuel **%s** left. FIX THIS SHIT NOW :rage:", pos.LocationName, pos.OwnerName, remaining, fuel.TypeName))
					log.WithFields(logrus.Fields{
						"starbaseID":   pos.ID,
						"fuelTypeID":   fuel.TypeID,
						"notification": 2,
					}).Info("Notification for critical fuel status sent")
				} else {
					log.WithFields(logrus.Fields{
						"starbaseID":   pos.ID,
						"fuelTypeID":   fuel.TypeID,
						"notification": 2,
					}).Debug("Notification already sent, skipping critical fuel status")
				}
			} else if int(fuel.TimeRemaining) <= b.config.EVE.FuelWarningThreshold {
				if b.shouldSendNotification(pos.ID, fuel.TypeID, 1) {
					b.discord.ChannelMessageSend(b.config.Discord.ChannelID, fmt.Sprintf("@here :alarm_clock: POS at **%s** (owned by %s) has **%s** of fuel **%s** left, someone should probably check that :thinking:", pos.LocationName, pos.OwnerName, remaining, fuel.TypeName))
					log.WithFields(logrus.Fields{
						"starbaseID":   pos.ID,
						"fuelTypeID":   fuel.TypeID,
						"notification": 1,
					}).Info("Notification for warning fuel status sent")
				} else {
					log.WithFields(logrus.Fields{
						"starbaseID":   pos.ID,
						"fuelTypeID":   fuel.TypeID,
						"notification": 1,
					}).Debug("Notification already sent, skipping warning fuel status")
				}
			}
		}
	}

	log.Info("Finished checking starbase fuel")
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
	log.Debug("Retrieving starbase list")

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

	log.Debug("Retrieved starbase list from EVE API")
	return starbases, nil
}

func (b *Bot) retrieveStarbaseDetails(starbaseID int) (*eveapi.StarbaseDetails, error) {
	log.WithField("starbaseID", starbaseID).Debug("Retrieving starbase details")

	starbase, err := b.retrieveCachedStarbaseDetails(starbaseID)
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrap(err, "Failed to retrieve cached starbase details")
	}

	if err != redis.ErrNil && starbase != nil {
		log.WithField("starbaseID", starbaseID).Debug("Retrieved starbase details from cache")
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

	log.WithField("starbaseID", starbaseID).Debug("Retrieved starbase details from EVE API")
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

func (b *Bot) updateMonitoredStarbaseDetails() error {
	monitored, err := b.getMonitoredStarbaseIDs()
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve monitored starbases")
	}

	for _, id := range monitored {
		_, err = b.retrieveStarbaseDetails(id)
		if err != nil {
			log.WithField("starbaseID", id).WithError(err).Warn("Failed to retrieve starbase details")
			continue
		}
	}

	return nil
}

type POS struct {
	ID           int
	LocationID   int
	LocationName string
	OwnerID      int
	OwnerName    string
	State        eveapi.StarbaseState
	Monitored    bool
	CachedUntil  time.Time
	Size         POSSize
	Fuel         []POSFuel
}

type POSSize int

const (
	POSSizeSmall POSSize = iota
	POSSizeMedium
	POSSizeLarge
)

func (s POSSize) String() string {
	switch s {
	case POSSizeSmall:
		return "small"
	case POSSizeMedium:
		return "medium"
	case POSSizeLarge:
		return "large"
	default:
		return "*unknown POS size*"
	}
}

type POSFuel struct {
	Type               POSFuelType
	TypeID             int
	TypeName           string
	Quantity           int
	Required           int
	ConstantlyRequired bool
	TimeRemaining      float64
}

type POSFuelType int

const (
	POSFuelTypeFuelBlock POSFuelType = iota
	POSFuelTypeStrontium
)

func (b *Bot) getPOSFromStarbaseID(starbaseID int) (*POS, error) {
	log.WithField("starbaseID", starbaseID).Debug("Retrieving POS")

	pos, err := b.retrieveCachedPOS(starbaseID)
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrap(err, "Failed to retrieve cached POS")
	}

	if err != redis.ErrNil && pos != nil {
		log.WithField("starbaseID", starbaseID).Debug("Retrieved POS from cache")
		return pos, nil
	}

	starbases, err := b.retrieveStarbaseList()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase list")
	}

	var starbase *eveapi.Starbase = nil
	for _, base := range starbases.Starbases {
		if base.ID == starbaseID {
			starbase = base
		}
	}

	if starbase == nil {
		return nil, errors.New("Starbase not found")
	}

	starbaseDetails, err := b.retrieveStarbaseDetails(starbaseID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase details")
	}

	starbaseType, _, err := b.esi.UniverseApi.GetUniverseTypesTypeId(int32(starbase.TypeID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve starbase type details")
	}

	size := POSSizeLarge
	if strings.Contains(starbaseType.Name, "Small") {
		size = POSSizeSmall
	} else if strings.Contains(starbaseType.Name, "Medium") {
		size = POSSizeMedium
	}

	locationName, err := b.getLocationNameFromMoonID(starbase.MoonID)
	if err != nil {
		log.WithFields(logrus.Fields{
			"starbaseID": starbase.ID,
			"locationID": starbase.MoonID,
		}).WithError(err).Warn("Failed to retrieve location name for POS")
		locationName = fmt.Sprintf("*unknown location - %d*", starbase.MoonID)
	}

	corporationName, err := b.getCorporationNameFromID(starbase.StandingOwnerID)
	if err != nil {
		log.WithFields(logrus.Fields{
			"starbaseID":    starbase.ID,
			"corporationID": starbase.StandingOwnerID,
		}).WithError(err).Warn("Failed to get corporation name for POS")
		corporationName = fmt.Sprintf("*unknown corporation - %d*", starbase.StandingOwnerID)
	}

	posFuel := make([]POSFuel, 0)
	for _, fuel := range starbaseDetails.Fuel {
		typeName, _, err := b.esi.UniverseApi.GetUniverseTypesTypeId(int32(fuel.TypeID), nil)
		if err != nil {
			log.WithFields(logrus.Fields{
				"starbaseID": starbase.ID,
				"typeID":     fuel.TypeID,
			}).WithError(err).Warn("Failed to get fuel name for POS")
			return nil, errors.Wrap(err, "Failed to get fuel name for POS")
		}

		fuelType := POSFuelTypeFuelBlock
		constantlyRequired := false
		if strings.Contains(typeName.Name, "Fuel Block") {
			constantlyRequired = true
		} else if strings.Contains(typeName.Name, "Strontium") {
			fuelType = POSFuelTypeStrontium
		}

		required := requiredFuelForSize(fuelType, size)

		posFuel = append(posFuel, POSFuel{
			Type:               fuelType,
			TypeID:             fuel.TypeID,
			TypeName:           typeName.Name,
			Quantity:           fuel.Quantity,
			Required:           required,
			ConstantlyRequired: constantlyRequired,
			TimeRemaining:      float64(fuel.Quantity) / float64(required),
		})
	}

	cachedUntil := starbaseDetails.CachedUntil.Time
	if starbaseDetails.CachedUntil.Time.After(starbases.CachedUntil.Time) {
		cachedUntil = starbases.CachedUntil.Time
	}

	pos = &POS{
		ID:           starbase.ID,
		LocationID:   starbase.LocationID,
		LocationName: locationName,
		OwnerID:      starbase.StandingOwnerID,
		OwnerName:    corporationName,
		State:        starbase.State,
		Monitored:    true,
		CachedUntil:  cachedUntil,
		Size:         size,
		Fuel:         posFuel,
	}

	err = b.cachePOS(pos)
	if err != nil {
		log.WithField("starbaseID", starbaseID).WithError(err).Warn("Failed to cache POS")
	}

	log.WithField("starbaseID", starbaseID).Debug("Retrieved POS")
	return pos, nil
}

var (
	requiredStarbaseFuel map[POSSize]map[POSFuelType]int = map[POSSize]map[POSFuelType]int{
		POSSizeSmall: {
			POSFuelTypeFuelBlock: 10,
			POSFuelTypeStrontium: 100,
		},
		POSSizeMedium: {
			POSFuelTypeFuelBlock: 20,
			POSFuelTypeStrontium: 200,
		},
		POSSizeLarge: {
			POSFuelTypeFuelBlock: 40,
			POSFuelTypeStrontium: 400,
		},
	}
)

func requiredFuelForSize(fuelType POSFuelType, size POSSize) int {
	required, ok := requiredStarbaseFuel[size][fuelType]
	if !ok {
		log.WithFields(logrus.Fields{
			"posFuelType": fuelType,
			"posSize":     size,
		}).Warn("No POS fuel entry for this type/size found")
		return 0
	}

	return required
}
