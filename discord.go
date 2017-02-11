package main

import (
	"fmt"
	"github.com/MorpheusXAUT/durafmt"
	"github.com/MorpheusXAUT/eveapi"
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"strconv"
	"strings"
	"time"
)

const (
	DiscordEmbedColorBlue   = 395860
	DiscordEmbedColorGreen  = 549640
	DiscordEmbedColorOrange = 16750848
	DiscordEmbedColorRed    = 15011085
	DiscordEmbedColorWhite  = 16777215
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

	if strings.EqualFold(message.Content, "!pos") || strings.HasPrefix(strings.ToLower(message.Content), "!pos ") {
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
		log.WithField("author", message.Author.Username).Info("Processing POS help Discord command")

		monitored, err := b.getMonitoredStarbaseIDs()
		if err != nil {
			b.recordCommandError("help")
			log.WithField("author", message.Author.Username).WithError(err).Warn("Failed to get monitored POS IDs")
		}

		b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("Hey <@%s>, I'm **POSbot**, glad to meet you :slight_smile: I am keeping track of EVE Online POSes for you. At the moment, I'm monitoring %d POSes.", message.Author.ID, len(monitored)))
		b.discord.ChannelMessageSend(message.ChannelID, "You can use various commands to query information about POS statuses, but I'll also shout at you if something is about to go wrong :smile:")
		b.discord.ChannelMessageSend(message.ChannelID, "A list of POSes can be displayed via `!pos list`, `!pos fuel` will show an overview of fuel for monitored POSes. `!pos details POSID` tells you more about a specific starbase. `!pos` or `!pos help` displays this help message. That's about it for now!")
		if isAdmin {
			b.discord.ChannelMessageSend(message.ChannelID, "Oh wait, you're super \"important\" :nerd: You can also use `!pos stats` to display performance stats, `!pos restart` to restart the bot or `!pos shutdown` to shut it down completely :skull:")
		}

		b.recordCommandUsage("help")
		log.WithField("author", message.Author.Username).Info("Processed POS help Discord command")
		return
	} else if len(messageParts) >= 2 {
		switch strings.ToLower(messageParts[1]) {
		case "details":
			log.WithField("author", message.Author.Username).Info("Processing POS details Discord command")
			if len(messageParts) < 3 {
				b.recordCommandError("details")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s>: You'll have to tell me which POS you want to know more about...", message.Author.ID))
				return
			}

			starbaseID, err := strconv.ParseInt(messageParts[2], 10, 64)
			if err != nil || starbaseID <= 0 {
				log.WithField("starbaseID", starbaseID).WithError(err).Debug("Failed to parse starbaseID for Discord POS details command")
				b.recordCommandError("details")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s>: Seems like you've provided an invalid POS ID %q :poop:", message.Author.ID, messageParts[2]))
				return
			}

			b.handleDiscordPOSDetailsCommand(message.ChannelID, message.Author.ID, int(starbaseID))
			log.WithField("author", message.Author.Username).Info("Processed POS details Discord command")
			return
		case "fuel":
			log.WithField("author", message.Author.Username).Info("Processing POS details Discord command")
			b.handleDiscordPOSFuelCommand(message.ChannelID, message.Author.ID)
			log.WithField("author", message.Author.Username).Info("Processed POS fuel Discord command")
			return
		case "list":
			log.WithField("author", message.Author.Username).Info("Processing POS list Discord command")
			b.handleDiscordPOSListCommand(message.ChannelID, message.Author.ID)
			log.WithField("author", message.Author.Username).Info("Processed POS list Discord command")
			return
		case "restart":
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Info("Processing POS restart Discord command")
			if !isAdmin {
				log.WithFields(logrus.Fields{
					"author":  message.Author.Username,
					"isAdmin": isAdmin,
				}).Info("Non-admin attempted to execute POS restart Discord command, ignoring")
				b.recordCommandError("restart")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("You don't have permission to do that, <@%s> :rage: I'll just be ignoring you, alright? :zipper_mouth:", message.Author.ID))
				return
			}

			b.handleDiscordPOSRestartCommand(message.ChannelID, message.Author.ID)
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Info("Processed POS restart Discord command")
			return
		case "shutdown":
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Info("Processing POS shutdown Discord command")
			if !isAdmin {
				log.WithFields(logrus.Fields{
					"author":  message.Author.Username,
					"isAdmin": isAdmin,
				}).Info("Non-admin attempted to execute POS shutdown Discord command, ignoring")
				b.recordCommandError("shutdown")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("You don't have permission to do that, <@%s> :rage: I'll just be ignoring you, alright? :zipper_mouth:", message.Author.ID))
				return
			}

			b.handleDiscordPOSShutdownCommand(message.ChannelID, message.Author.ID)
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Info("Processed POS shutdown Discord command")
			return
		case "stats":
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Info("Processing POS stats Discord command")
			if !isAdmin {
				log.WithFields(logrus.Fields{
					"author":  message.Author.Username,
					"isAdmin": isAdmin,
				}).Info("Non-admin attempted to execute POS stats Discord command, ignoring")
				b.recordCommandError("stats")
				b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("You don't have permission to do that, <@%s> :rage: I'll just be ignoring you, alright? :zipper_mouth:", message.Author.ID))
				return
			}

			b.handleDiscordPOSStatsCommand(message.ChannelID, message.Author.ID)
			log.WithFields(logrus.Fields{
				"author":  message.Author.Username,
				"isAdmin": isAdmin,
			}).Info("Processed POS stats Discord command")
			return
		}
	}

	b.discord.ChannelMessageSend(message.ChannelID, fmt.Sprintf("<@%s> seems to be drunk, there's no command like this :thinking:", message.Author.ID))
}

func (b *Bot) handleDiscordPOSDetailsCommand(channelID string, userID string, starbaseID int) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
	b.recordCommandError("details")
}

func (b *Bot) handleDiscordPOSFuelCommand(channelID string, userID string) {
	err := b.updateMonitoredStarbaseDetails()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to update monitored starbase details for Discord command")
		b.recordCommandError("fuel")
		b.discord.ChannelMessageSend(channelID, fmt.Sprintf("It appears like I can't update the POS details at the moment :neutral_face: My deepest apologies, <@%s>", userID))
		return
	}

	monitored, err := b.getMonitoredStarbaseIDs()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to retrieve monitored starbase IDs for Discord command")
		b.recordCommandError("fuel")
		b.discord.ChannelMessageSend(channelID, fmt.Sprintf("It appears like I can't retrieve a list of monitored POSes at the moment :neutral_face: My deepest apologies, <@%s>", userID))
		return
	}

	b.discord.ChannelMessageSend(channelID, fmt.Sprintf("<@%s> is currently monitoring **%d** POSes.", b.discord.State.User.ID, len(monitored)))
	b.discord.ChannelTyping(channelID)

	for i, id := range monitored {
		pos, err := b.getPOSFromStarbaseID(id)
		if err != nil {
			log.WithFields(logrus.Fields{
				"userID":     userID,
				"starbaseID": id,
			}).WithError(err).Warn("Failed to get POS for Discord command")
			continue
		}

		fields := make([]*discordgo.MessageEmbedField, 0)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  strings.Replace(pos.LocationName, "Moon", ":full_moon_with_face:", -1),
			Inline: true,
		})

		_, strState := formatStarbaseStateForDiscord(pos.State)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "State",
			Value:  fmt.Sprintf("%s %s", strState, pos.State),
			Inline: true,
		})

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Size",
			Value:  pos.Size.String(),
			Inline: true,
		})

		fuelStatus := 0
		for _, fuel := range pos.Fuel {
			remaining, err := durafmt.ParseString(fmt.Sprintf("%fh", fuel.TimeRemaining))
			if err != nil {
				log.WithFields(logrus.Fields{
					"userID":     userID,
					"starbaseID": pos.ID,
					"fuelTypeID": fuel.TypeID,
				}).WithError(err).Warn("Failed to parse remaining fuel duration")
				continue
			}
			constantly := "no"
			if fuel.ConstantlyRequired {
				constantly = "yes"
			}

			remain := remaining.Short()
			if fuel.ConstantlyRequired {
				if int(fuel.TimeRemaining) <= b.config.EVE.FuelCriticalThreshold {
					remain = fmt.Sprintf("__**%s**__", remain)
					fuelStatus = 2
				} else if int(fuel.TimeRemaining) <= b.config.EVE.FuelWarningThreshold {
					remain = fmt.Sprintf("**%s**", remain)
					if fuelStatus < 1 {
						fuelStatus = 1
					}
				}
			}

			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("Fuel *%s*", fuel.TypeName),
				Value:  fmt.Sprintf("*quantity*: %d, *remaining*: %s, *used/h*: %d, *constantly required*: %s", fuel.Quantity, remain, fuel.Required, constantly),
				Inline: false,
			})
		}

		color := DiscordEmbedColorGreen
		if fuelStatus == 1 {
			color = DiscordEmbedColorOrange
		} else if fuelStatus == 2 {
			color = DiscordEmbedColorRed
		}

		embed := &discordgo.MessageEmbed{
			Color:       color,
			Title:       fmt.Sprintf(":stars: POS %d/%d", i+1, len(monitored)),
			Description: fmt.Sprintf("POS owned by **%s**", pos.OwnerName),
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("POS cached for %v", pos.CachedUntil.Sub(time.Now().UTC())),
			},
		}

		b.discord.ChannelMessageSendEmbed(channelID, embed)
		b.discord.ChannelTyping(channelID)
	}

	b.discord.ChannelMessageSend(channelID, fmt.Sprintf("I will shout at you if a POS should fall under %dh fuel remaining (warning, *orange*) and absolutely flip out at %dh fuel left (critical, *red*) :hugging:", b.config.EVE.FuelWarningThreshold, b.config.EVE.FuelCriticalThreshold))
	b.recordCommandUsage("fuel")
}

func (b *Bot) handleDiscordPOSListCommand(channelID string, userID string) {
	starbases, err := b.retrieveStarbaseList()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to retrieve starbase list for Discord command")
		b.recordCommandError("list")
		b.discord.ChannelMessageSend(channelID, fmt.Sprintf("It appears like I can't retrieve a list of POSes at the moment :neutral_face: My deepest apologies, <@%s>", userID))
		return
	}

	b.discord.ChannelMessageSend(channelID, fmt.Sprintf("There is currently **%d** POSes visible to <@%s>, including both monitored and ignored structures.", len(starbases.Starbases), b.discord.State.User.ID))
	b.discord.ChannelTyping(channelID)

	for i, starbase := range starbases.Starbases {
		fields := make([]*discordgo.MessageEmbedField, 0)

		location, err := b.getLocationNameFromMoonID(starbase.MoonID)
		if err != nil {
			log.WithFields(logrus.Fields{
				"userID":     userID,
				"starbaseID": starbase.ID,
				"locationID": starbase.LocationID,
			}).WithError(err).Warn("Failed to retrieve location name for starbase list")
			location = fmt.Sprintf("*unknown location - %d*", starbase.LocationID)
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  strings.Replace(location, "Moon", ":full_moon_with_face:", -1),
			Inline: true,
		})

		color, strState := formatStarbaseStateForDiscord(starbase.State)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "State",
			Value:  fmt.Sprintf("%s %s", strState, starbase.State),
			Inline: true,
		})

		monitored := b.isStarbaseMonitored(starbase.ID)
		strMonitored := ":white_check_mark:"
		if !monitored {
			strMonitored = ":x:"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Monitored",
			Value:  strMonitored,
			Inline: true,
		})

		corporationName, err := b.getCorporationNameFromID(starbase.StandingOwnerID)
		if err != nil {
			log.WithFields(logrus.Fields{
				"userID":        userID,
				"starbaseID":    starbase.ID,
				"corporationID": starbase.StandingOwnerID,
			}).WithError(err).Warn("Failed to get corporation name from ID")
			corporationName = fmt.Sprintf("*unknown corporation - %d*", starbase.StandingOwnerID)
		}

		embed := &discordgo.MessageEmbed{
			Color:       color,
			Title:       fmt.Sprintf(":stars: POS %d/%d", i+1, len(starbases.Starbases)),
			Description: fmt.Sprintf("POS owned by **%s**", corporationName),
			Fields:      fields,
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("POS overview cached for %v", starbases.CachedUntil.Time.Sub(time.Now().UTC())),
			},
		}

		b.discord.ChannelMessageSendEmbed(channelID, embed)
		b.discord.ChannelTyping(channelID)
	}

	b.discord.ChannelMessageSend(channelID, "You can request additional information about a POS - like it's current fuel status - using `!pos details POSID`.")
	b.recordCommandUsage("list")
}

func (b *Bot) handleDiscordPOSRestartCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
	b.recordCommandError("restart")
}

func (b *Bot) handleDiscordPOSShutdownCommand(channelID string, userID string) {
	b.discord.ChannelMessageSend(channelID, "Not implemented yet :innocent:")
	b.recordCommandError("shutdown")
}

func (b *Bot) handleDiscordPOSStatsCommand(channelID string, userID string) {
	fields := make([]*discordgo.MessageEmbedField, 0)

	log.WithField("userID", userID).Debug("Gathering host info for Discord command")
	hostInfo, err := host.Info()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to get host info")
	} else {
		uptime, _ := time.ParseDuration(fmt.Sprintf("%ds", hostInfo.Uptime))
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Host",
			Value:  fmt.Sprintf("**Hostname**: %s, **OS**: %s %s, **uptime**: %v", hostInfo.Hostname, hostInfo.Platform, hostInfo.PlatformFamily, uptime),
			Inline: false,
		})
	}

	log.WithField("userID", userID).Debug("Gathering host CPU stats for Discord command")
	cpuCount, err := cpu.Counts(false)
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to get host CPU count")
	} else {
		cpuUsage, err := cpu.Percent(0, false)
		if err != nil {
			log.WithField("userID", userID).WithError(err).Warn("Failed to get host CPU usage")
		} else {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "CPU",
				Value:  fmt.Sprintf("**Number of cores**: %d, **usage**: %.2f", cpuCount, cpuUsage[0]),
				Inline: false,
			})
		}
	}

	log.WithField("userID", userID).Debug("Gathering host load average stats for Discord command")
	loadAvg, err := load.Avg()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to get host load average stats")
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Load average",
			Value:  fmt.Sprintf("**1 min**: %.2f%%, **5 min**: %.2f%%, **15 min**: %.2f%%", loadAvg.Load1, loadAvg.Load5, loadAvg.Load15),
			Inline: false,
		})
	}

	log.WithField("userID", userID).Debug("Gathering host memory stats for Discord command")
	vMem, err := mem.VirtualMemory()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to get host virtual memory stats")
	} else {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Virtual memory",
			Value:  fmt.Sprintf("**Total**: %s, **available**: %s, **used**: %s (%.2f%%), **free**: %s", humanize.Bytes(vMem.Total), humanize.Bytes(vMem.Available), humanize.Bytes(vMem.Used), vMem.UsedPercent, humanize.Bytes(vMem.Free)),
			Inline: false,
		})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "POSbot",
		Value:  fmt.Sprintf("**Version**: %s, **uptime**: %v", Version, time.Since(b.startTime)),
		Inline: false,
	})

	stats, err := b.retrieveCommandStats()
	if err != nil {
		log.WithField("userID", userID).WithError(err).Warn("Failed to retrieve command stats")
		b.recordCommandError("stats")
		b.discord.ChannelMessageSend(channelID, ":poop: Seems like there was an error processing this command :poop:")
		return
	}

	for command, stat := range stats {
		descUsage := "times"
		descError := "times"
		if stat.Usage == 1 {
			descUsage = "time"
		}
		if stat.Error == 1 {
			descError = "time"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Command usage `!pos %s`", command),
			Value:  fmt.Sprintf("**Usage**: %d %s, **error**: %d %s", stat.Usage, descUsage, stat.Error, descError),
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Color:       DiscordEmbedColorWhite,
		Title:       ":bar_chart: **POSbot** stats",
		Description: "Runtime and command usage stats for **POSbot**",
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Stats generated at: %s", time.Now().UTC().Format(time.RFC1123)),
		},
	}

	b.discord.ChannelMessageSendEmbed(channelID, embed)
	b.recordCommandUsage("stats")
}

func formatStarbaseStateForDiscord(state eveapi.StarbaseState) (int, string) {
	switch state {
	case eveapi.StarbaseStateOnline:
		return DiscordEmbedColorGreen, ":satellite_orbital:"
	case eveapi.StarbaseStateOnlining:
		return DiscordEmbedColorGreen, ":construction_site:"
	case eveapi.StarbaseStateAnchored:
		return DiscordEmbedColorOrange, ":anchor:"
	case eveapi.StarbaseStateUnanchored:
		return DiscordEmbedColorRed, ":warning:"
	case eveapi.StarbaseStateReinforced:
		return DiscordEmbedColorRed, ":space_invader:"
	default:
		return DiscordEmbedColorRed, ":x:"
	}
}
