# PrivateBot
**A bot to let users have private voice channels in Discord.**


## Building/Running
1. `dep ensure`
2. `go build cmd/privatebot/main.go` or `go run cmd/privatebot/main.go`
3. run the compiled binary (if `go build` was used in step 2.
4. Add the bot token in the config.json
5. Start the bot again.

## Configruation / Setup
- The bot requires admin permissions.
- The server should be set up with a waiting room where nobody can talk.
- The server should have private channels with a userlimit of 1.

1. Add the bot to the server.
2. Join the waiting room and type `/setwaitingroom`
3. Done, for users to add their friends, they need to get them in the waiting room and use `/join @friend1 @friend2`
