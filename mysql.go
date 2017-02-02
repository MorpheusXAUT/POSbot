package main

import "github.com/pkg/errors"

func (b *Bot) getLocationNameFromMoonID(moonID int) (string, error) {
	var name string
	err := b.mysql.Get(&name, "SELECT itemName FROM mapDenormalize WHERE itemID = ?", moonID)
	if err != nil {
		return "", errors.Wrap(err, "Failed to query location name")
	}

	return name, nil
}
