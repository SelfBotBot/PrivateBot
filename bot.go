package PrivateBot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SelfBotBot/PrivateBot/discordio"
	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session          *discordgo.Session
	WaitingRooms     *WaitingRooms
	PrivateRoomUsers map[string]map[string]string //map[GuildID]map[UserID]ChannelID
}

func New(rooms *WaitingRooms) (*Bot, error) {
	ret := &Bot{
		WaitingRooms:     rooms,
		PrivateRoomUsers: make(map[string]map[string]string),
	}
	var err error
	ret.Session, err = discordgo.New("Bot " + rooms.Token)

	//ret.Session.LogLevel = 3

	ret.Session.AddHandler(ret.botCommand)
	ret.Session.AddHandler(ret.create)
	ret.Session.AddHandler(ret.ready)
	ret.Session.AddHandler(ret.voiceUpdate)

	return ret, err
}

func (b *Bot) create(s *discordgo.Session, event *discordgo.GuildCreate) {
	fmt.Println("GuildCreate for " + event.ID)
	b.SetupGuild(event.Guild)
}

func (b *Bot) ready(s *discordgo.Session, _ *discordgo.Ready) {
	s.UpdateStatus(0, "/private | /priv")
}

func (b *Bot) voiceUpdate(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	var waitingRoom string
	if room, ok := b.WaitingRooms.GetRoom(vsu.GuildID); !ok {
		return
	} else {
		waitingRoom = room
	}

	users, ok := b.PrivateRoomUsers[vsu.GuildID]
	if !ok {
		users = make(map[string]string)
		b.PrivateRoomUsers[vsu.GuildID] = users
	}

	if vsu.ChannelID == "" {
		delete(users, vsu.UserID)
	} else {
		if voiceChan, err := s.Channel(vsu.ChannelID); err != nil || voiceChan.Type != discordgo.ChannelTypeGuildVoice {
			return
		} else {
			if voiceChan.UserLimit != 1 {
				delete(users, vsu.UserID)
			} else {
				users[vsu.UserID] = voiceChan.ID
			}
		}
	}

	channels, err := s.GuildChannels(vsu.GuildID)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, channel := range channels {
		if channel.Type != discordgo.ChannelTypeGuildVoice || channel.UserLimit != 1 {
			continue
		}

		allBots := true
		for k, v := range users {
			if v != channel.ID {
				continue
			}

			user, err := s.User(k)
			if err == nil {
				if !user.Bot {
					allBots = false
					break
				}
			}
		}

		if allBots {
			for k, v := range users {
				if v == channel.ID {
					if err := s.GuildMemberMove(vsu.GuildID, k, waitingRoom); err != nil {
						fmt.Println(err)
					}
				}
			}
		}
	}

}

func (b *Bot) botCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	c, err := s.State.Channel(m.ChannelID)
	if err != nil {
		return
	}

	g, err := s.State.Guild(c.GuildID)
	if err != nil {
		return
	}

	if strings.HasPrefix(m.Content, "/join ") {
		if len(m.Mentions) == 0 {
			s.ChannelMessageSend(c.ID, "You've selected nobody to join you!\n")
			return
		}

		var waitingRoomID string
		var ok bool
		if waitingRoomID, ok = b.WaitingRooms.Rooms[c.GuildID]; !ok {
			s.ChannelMessageSend(c.ID, "There is no waiting room configured!\nUse `/setwaitingroom` whilst in the waiting room.\n")
			return
		}

		channel, err := b.FindUserInGuild(m.Author.ID, g.ID)
		if err != nil {
			s.ChannelMessageSend(c.ID, "Unable to find you in VC.\n```"+err.Error()+"```")
			return
		}

		if voiceChan, err := s.Channel(channel); err != nil {
			s.ChannelMessageSend(c.ID, "Unable to find the voice channel you are in.\n```"+err.Error()+"```")
			return
		} else if voiceChan.Type != discordgo.ChannelTypeGuildVoice || voiceChan.UserLimit != 1 {
			s.ChannelMessageSend(c.ID, "That's not a private channel!")
			return
		}

		failedUsers := make(map[*discordgo.User]string)
		for _, user := range m.Mentions {
			if userChan, err := b.FindUserInGuild(user.ID, g.ID); err != nil {
				failedUsers[user] = "Unable to find user in VC - " + err.Error()
			} else if userChan != waitingRoomID {
				failedUsers[user] = "User isn't in the waiting room."
			} else {
				if err := b.Session.GuildMemberMove(g.ID, user.ID, channel); err != nil {
					failedUsers[user] = "Unable to move user - " + err.Error()
				}
			}
		}

		writer := discordio.NewMessageWriter(s, m)
		if len(failedUsers) != 0 {
			for k, v := range failedUsers {
				writer.Write([]byte(discordio.Escape(k.Username) + ": " + v))
			}
			if err := writer.Close(); err != nil {
				fmt.Println(err)
			}
		}

		return
	}

	if strings.EqualFold(m.Content, "/setwaitingroom") {

		member, err := s.GuildMember(g.ID, m.Author.ID)
		if err != nil {
			fmt.Println(err)
			return
		}

		roles, err := s.GuildRoles(g.ID)
		if err != nil {
			fmt.Println(err)
			return
		}

		found := false
		for _, role := range roles {
			for _, role2 := range member.Roles {
				if role.ID == role2 {
					if (role.Permissions & discordgo.PermissionAdministrator) > 0 {
						found = true
						break
					}
				}
			}
		}

		if !found {
			return
		}

		channel, err := b.FindUserInGuild(m.Author.ID, g.ID)
		if err != nil {
			s.ChannelMessageSend(c.ID, "Unable to find you in VC.\n```"+err.Error()+"```")
			return
		}

		if err := b.WaitingRooms.AddRoom(g.ID, channel); err != nil {
			s.ChannelMessageSend(c.ID, "Unable to set the channel as a waiting room..\n```"+err.Error()+"```")
			return
		} else {
			b.SetupGuild(g)
			s.MessageReactionAdd(c.ID, m.ID, "✅")
		}
	}
}

// FindUserInGuild finds the provided userid in the guildID's voice channel and provides the channelID
func (b *Bot) FindUserInGuild(UserID string, GuildID string) (ChannelID string, err error) {
	g, err := b.Session.State.Guild(GuildID)
	if err != nil {
		fmt.Printf("%#v\n", b.Session.State.Guilds)
		return
	}

	for _, vs := range g.VoiceStates {
		if vs.UserID == UserID {
			ChannelID = vs.ChannelID
			return
		}
	}

	err = errors.New("no user in channel")
	return
}

func (b *Bot) SetupGuild(guild *discordgo.Guild) {
	if _, ok := b.WaitingRooms.GetRoom(guild.ID); !ok {
		return
	}

	users, ok := b.PrivateRoomUsers[guild.ID]
	if !ok {
		users = make(map[string]string)
		b.PrivateRoomUsers[guild.ID] = users
	}

	for _, channel := range guild.Channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice && channel.UserLimit == 1 {
			for _, state := range guild.VoiceStates {
				if state.ChannelID == channel.ID {
					users[state.UserID] = state.ChannelID
				}
			}
		}
	}
}
