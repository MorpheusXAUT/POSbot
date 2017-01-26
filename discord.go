package main

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
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
	if !b.isRelevantDiscordChannel(message) {
		return
	}

	if strings.HasPrefix(strings.ToLower(message.Content), "!pos") {
		b.handleDiscordPOSCommand(message)
	}
}

func (b *Bot) getChannelFromMessage(message *discordgo.MessageCreate) (*discordgo.Channel, error) {
	return b.discord.Channel(message.ChannelID)
}

func (b *Bot) getGuildFromMessage(message *discordgo.MessageCreate) (*discordgo.Guild, error) {
	channel, err := b.discord.Channel(message.ChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get channel from message")
	}

	return b.discord.Guild(channel.GuildID)
}

func (b *Bot) getGuildMember(guildID string, userID string) (*discordgo.Member, error) {
	return b.discord.GuildMember(guildID, userID)
}

func (b *Bot) hasBotAdminRole(message *discordgo.MessageCreate) bool {
	guild, err := b.getGuildFromMessage(message)
	if err != nil {
		log.WithField("author", message.Author.Username).WithError(err).Warn("Failed to get guild from message")
		return false
	}

	member, err := b.getGuildMember(guild.ID, message.Author.ID)
	if err != nil {
		log.WithField("author", message.Author.Username).WithError(err).Warn("Failed to get guild member")
		return false
	}

	admin := false
	for _, role := range member.Roles {
		if strings.EqualFold(role, b.config.Discord.BotAdminRoleID) {
			admin = true
			break
		}
	}

	return admin
}

func (b *Bot) isRelevantDiscordChannel(message *discordgo.MessageCreate) bool {
	channel, err := b.getChannelFromMessage(message)
	if err != nil {
		log.WithFields(logrus.Fields{
			"channelID": message.ChannelID,
			"author":    message.Author.Username,
		}).Debug("Failed to get channel from message")
		return false
	}

	return strings.EqualFold(channel.GuildID, b.config.Discord.GuildID) && strings.EqualFold(channel.ID, b.config.Discord.ChannelID)
}

func (b *Bot) handleDiscordPOSCommand(message *discordgo.MessageCreate) {
	b.discord.ChannelTyping(message.ChannelID)

	messageParts := strings.Split(message.Content, " ")
	isAdmin := b.hasBotAdminRole(message)

	if len(messageParts) == 1 || (len(messageParts) == 2 && strings.EqualFold(messageParts[1], "help")) {
		log.WithField("author", message.Author.Username).Debug("Processing POS help Discord command")

		monitored, err := b.getMonitoredStarbaseIDs()
		if err != nil {
			log.WithField("author", message.Author.Username).WithError(err).Warn("Failed to get monitored POS IDs")
		}

		b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Hey <@%s>, I'm **POSbot**, glad to meet you :slight_smile: I am keeping track of EVE Online POSes for you. At the moment, I'm monitoring %d POSes.", message.Author.ID, len(monitored)))
		b.discord.ChannelMessageSend(message.ChannelID, "You can use various commands to query information about POS statuses, but I'll also shout at you if something is about to go wrong :smile:")
		b.discord.ChannelMessageSend(message.ChannelID, "A list of POSes can be displayed via `!pos list`, `!pos fuel` will show an overview of fuel for monitored POSes. `!pos details POSID` tells you more about a specific starbase. `!pos` or `!pos help` displays this help message. That's about it for now!")
		if isAdmin {
			b.discord.ChannelMessageSend(message.ChannelID, "Oh wait, you're super \"important\" :nerd: You can also use `!pos stats` to display performance stats, `!pos restart` to restart the bot or `!pos shutdown` to shut it down completely :skull:")
		}

		log.WithField("author", message.Author.Username).Debug("Processed POS help Discord command")
		return
	} else if len(messageParts) >= 2 {
		switch strings.ToLower(messageParts[1]) {
		case "details":
			log.WithField("author", message.Author.Username).Debug("Processing POS details Discord command")
			if len(messageParts) < 3 {
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s>: You'll have to tell me which POS you want to know more about...", message.Author.ID))
				return
			}

			starbaseID, err := strconv.ParseInt(messageParts[2], 10, 64)
			if err != nil || starbaseID <= 0 {
				log.WithField("starbaseID", starbaseID).WithError(err).Debug("Failed to parse starbaseID for Discord POS details command")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s>: Seems like you've provided an invalid POS ID %q :poop:", message.Author.ID, messageParts[2]))
				return
			}

			b.handleDiscordPOSDetailsCommand(message.ChannelID, message.Author.ID, int(starbaseID))
			log.WithField("author", message.Author.Username).Debug("Processed POS details Discord command")
			return
		case "fuel":
			log.WithField("author", message.Author.Username).Debug("Processing POS details Discord command")
			b.handleDiscordPOSFuelCommand(message.ChannelID, message.Author.ID)
			log.WithField("author", message.Author.Username).Debug("Processed POS fuel Discord command")
			return
		case "list":
			log.WithField("author", message.Author.Username).Debug("Processing POS list Discord command")
			b.handleDiscordPOSListCommand(message.ChannelID, message.Author.ID)
			log.WithField("author", message.Author.Username).Debug("Processing POS list Discord command")
			return
		case "restart":
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Debug("Processing POS restart Discord command")
			if !isAdmin {
				log.WithFields(logrus.Fields{
					"author":  message.Author.Username,
					"isAdmin": isAdmin,
				}).Debug("Non-admin attempted to execute POS restart Discord command, ignoring")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("You don't have permission to do that, <@%s> :rage: I'll just be ignoring you, alright? :zipper_mouth:", message.Author.ID))
				return
			}

			b.handleDiscordPOSRestartCommand(message.ChannelID, message.Author.ID)
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Debug("Processed POS restart Discord command")
			return
		case "shutdown":
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Debug("Processing POS shutdown Discord command")
			if !isAdmin {
				log.WithFields(logrus.Fields{
					"author":  message.Author.Username,
					"isAdmin": isAdmin,
				}).Debug("Non-admin attempted to execute POS shutdown Discord command, ignoring")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("You don't have permission to do that, <@%s> :rage: I'll just be ignoring you, alright? :zipper_mouth:", message.Author.ID))
				return
			}

			b.handleDiscordPOSShutdownCommand(message.ChannelID, message.Author.ID)
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Debug("Processed POS shutdown Discord command")
			return
		case "stats":
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Debug("Processing POS stats Discord command")
			if !isAdmin {
				log.WithFields(logrus.Fields{
					"author":  message.Author.Username,
					"isAdmin": isAdmin,
				}).Debug("Non-admin attempted to execute POS stats Discord command, ignoring")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("You don't have permission to do that, <@%s> :rage: I'll just be ignoring you, alright? :zipper_mouth:", message.Author.ID))
				return
			}

			b.handleDiscordPOSStatsCommand(message.ChannelID, message.Author.ID)
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Debug("Processed POS stats Discord command")
			return
		}
	}

	b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s> seems to be drunk, there's no command like this :thinking:", message.Author.ID))
}

func (b *Bot) handleDiscordPOSDetailsCommand(channelID string, userID string, starbaseID int) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
}

func (b *Bot) handleDiscordPOSFuelCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
}

func (b *Bot) handleDiscordPOSListCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
}

func (b *Bot) handleDiscordPOSRestartCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
}

func (b *Bot) handleDiscordPOSShutdownCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
}

func (b *Bot) handleDiscordPOSStatsCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
}
