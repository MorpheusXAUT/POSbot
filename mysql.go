package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

var (
	mysqlRequiredTableNames []string = []string{"mapDenormalize"}
)

func (b *Bot) getLocationNameFromMoonID(moonID int) (string, error) {
	log.WithField("moonID", moonID).Debug("Retrieving location name for moon ID from MySQL")

	var name string
	err := b.mysql.Get(&name, "SELECT itemName FROM mapDenormalize WHERE itemID = ?", moonID)
	if err != nil {
		return "", errors.Wrap(err, "Failed to query location name")
	}

	log.WithFields(logrus.Fields{
		"moonID":       moonID,
		"locationName": name,
	}).Debug("Retrieved location name for moon ID from MySQL")
	return name, nil
}
