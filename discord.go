package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
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
	b.discord.ChannelMessageSend(message.ChannelID, "Would you kindly fuck off, please? :innocent:")
}
