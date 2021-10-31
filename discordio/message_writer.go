package discordio

import (
	"fmt"
	"io"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type MessageWriter struct {
	io.WriteCloser
	Session   *discordgo.Session
	Message   *discordgo.MessageCreate
	TotalSent uint
	MaxSent   uint
	CodeBlock bool
	Messages  []string
	Size      int
}

func NewMessageWriter(session *discordgo.Session, message *discordgo.MessageCreate) *MessageWriter {
	return &MessageWriter{
		Session:   session,
		Message:   message,
		TotalSent: 0,
		MaxSent:   4,
		CodeBlock: true,
		Messages:  []string{},
		Size:      0,
	}
}

func Escape(s string) string {
	return strings.Replace(strings.Replace(s, "@", "\\@", -1), "`", "\\`", -1)
}

func (w *MessageWriter) Write(p []byte) (n int, err error) {
	input := string(p[:])
	lines := strings.Split(removeBadReturns(input), "\n")
	for k, v := range lines {
		if len(v)+w.Size+1 >= 1990 {
			_ = w.sendMessage()
		}

		w.Size = len(v) + w.Size + 1
		if k+1 == len(lines) {
			if p[len(p)-1] != '\n' {
				w.Messages = append(w.Messages, v)
				w.Size = len(v) - 1
				continue
			}
		}

		w.Messages = append(w.Messages, v+"\n")
	}

	return len(p), nil
}

func (w *MessageWriter) Close() error {
	return w.sendMessage()
}

func (w *MessageWriter) sendMessage() error {
	if w.TotalSent >= w.MaxSent {
		w.Size = 0
		w.Messages = []string{}
		return
	}

	msg := strings.Join(w.Messages, "")
	if msg == "" {
		return
	}

	if w.CodeBlock {
		msg = "```" + strings.Replace(msg, "`", "\\`", -1) + "```"
	}

	_, err := w.Session.ChannelMessageSend(w.Message.ChannelID, msg)
	if err != nil {
		return err
	}

	w.Size = 0
	w.Messages = []string{}
	w.TotalSent++
	return nil

}

func removeBadReturns(str string) string {
	str = strings.Replace(str, "\r\n", "\n", -1)
	str = strings.Replace(str, "\r", "\n", -1)
	return str
}
