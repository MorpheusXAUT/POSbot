package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"strconv"
	"strings"
)

func (b *Bot) onDiscordReady(s *discordgo.Session, event *discordgo.Ready) {
	_ = s.UpdateStatus(0, "Starbase Online")
}

func (b *Bot) onDiscordGuildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		log.WithField("guildID", event.Guild.ID).Debug("Received GuildCreate event for unavaiable guild")
		return
	}

	targetFound := false
	if strings.EqualFold(event.Guild.ID, b.config.Discord.GuildID) {
		for _, channel := range event.Guild.Channels {
			if strings.EqualFold(channel.ID, b.config.Discord.ChannelID) {
				targetFound = true
				if b.config.Discord.Debug {
					s.ChannelMessageSend(channel.ID, ":robot: POSbot online and ready to serve :rocket:")
				}
			}
		}
	}

	if !targetFound {
		log.WithFields(logrus.Fields{
			"guildID":   b.config.Discord.GuildID,
			"channelID": b.config.Discord.ChannelID,
		}).Fatal("Target Discord guild or channel not found")
	}
}

func (b *Bot) onDiscordMessageCreate(s *discordgo.Session, message *discordgo.MessageCreate) {
	if !b.isRelevantDiscordChannel(s, message) {
		return
	}

	if strings.HasPrefix(strings.ToLower(message.Content), "!fuel") {
		b.handleDiscordFuelCommand(message)
	} else if strings.HasPrefix(strings.ToLower(message.Content), "!pos") {
		b.handleDiscordPOSCommand(message)
	}
}

func (b *Bot) getChannelFromMessage(s *discordgo.Session, message *discordgo.MessageCreate) (*discordgo.Channel, error) {
	return s.Channel(message.ChannelID)
}

func (b *Bot) isRelevantDiscordChannel(s *discordgo.Session, message *discordgo.MessageCreate) bool {
	channel, err := b.getChannelFromMessage(s, message)
	if err != nil {
		log.WithFields(logrus.Fields{
			"channelID": message.ChannelID,
			"author":    message.Author.Username,
		}).Debug("Failed to get channel from message")
		return false
	}

	return strings.EqualFold(channel.GuildID, b.config.Discord.GuildID) && strings.EqualFold(channel.ID, b.config.Discord.ChannelID)
}

func (b *Bot) handleDiscordFuelCommand(message *discordgo.MessageCreate) {
	b.discord.ChannelMessageSend(message.ChannelID, "How the fuck would I know? ¯\\_(ツ)_/¯")
}

func (b *Bot) handleDiscordPOSCommand(message *discordgo.MessageCreate) {
	b.discord.ChannelTyping(message.ChannelID)

	messageParts := strings.Split(message.Content, " ")

	if len(messageParts) == 1 || (len(messageParts) == 2 && strings.EqualFold(messageParts[1], "list")) {
		log.WithField("author", message.Author.Username).Debug("Retrieving POS list for Discord command")

		starbases, err := b.retrieveStarbaseList()
		if err != nil {
			log.WithError(err).Warn("Failed to retrieve POS list")
			b.discord.ChannelMessageSend(message.ChannelID, ":poop: Looks like there was an error retrieving the POS list :frowning:")
			return
		}

		b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Would you kindly fuck off, please? :innocent: BTW, there's %d POSes", len(starbases.Starbases)))
		log.WithField("author", message.Author.Username).Debug("Retrieved POS list for Discord command")
		return
	} else if len(messageParts) >= 2 {
		if strings.EqualFold(messageParts[1], "details") {
			if len(messageParts) < 3 {
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s>: You'll have to tell me which POS you want to know more about...", message.Author.ID))
				return
			}

			starbaseID, err := strconv.ParseInt(messageParts[2], 10, 64)
			if err != nil || starbaseID <= 0 {
				log.WithField("starbaseID", starbaseID).WithError(err).Debug("Failed to parse starbaseID for Discord POS command")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s>: Seems like you've provided an invalid POS ID :poop:", message.Author.ID))
				return
			}

			log.WithFields(logrus.Fields{
				"author":     message.Author.Username,
				"starbaseID": starbaseID,
			}).Debug("Retrieving POS details for Discord command")

			starbase, err := b.retrieveStarbaseDetails(int(starbaseID))
			if err != nil {
				log.WithError(err).Warn("Failed to retrieve POS details")
				b.discord.ChannelMessageSend(message.ChannelID, ":poop: Looks like there was an error retrieving the POS details :frowning:")
				return
			}

			b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Amazing. State: %s", starbase.State))
			log.WithFields(logrus.Fields{
				"author":     message.Author.Username,
				"starbaseID": starbaseID,
			}).Debug("Retrieved POS details for Discord command")
			return
		}
	}

	b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s> seems to be drunk, there's no command like this :thinking:", message.Author.ID))
}
