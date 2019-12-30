# TalkEQ

[![GoDoc](https://godoc.org/github.com/xackery/talkeq?status.svg)](https://godoc.org/github.com/xackery/talkeq) [![Go Report Card](https://goreportcard.com/badge/github.com/xackery/talkeq)](https://goreportcard.com/report/github.com/xackery/talkeq)

TalkEQ bridges links between everquest and other services. Extends [DiscordEQ](https://github.com/xackery/discordeq).

### Supported Services

* Discord
* Telnet (eqemu server)
* eqlog (logging via everquest's log system)


### Supported Channels

Name|Discord|Telnet|EQLog
---|---|---|---
OOC|Y|Y|Y
Auction|Y|N|Y
Shout|Y|N|Y
Guild|Y|N|Y
General|Y|N|Y

## Linking Discord

* Go to https://discordapp.com/developers/ and sign in
* Click New Application the top right area
* Write anything you wish for the app name, click Create App
* Copy the CLIENT ID into your talkeq.conf's discord client_id section
* Select your server, and allow it.
* On the left pane, click Bot
* Press Add Bot, Yes, do it!
* Press the copy button in the Token section
* Uncheck the Public Bot option
* Replace on this link's {CLIENT_ID} field with the client ID you obtained earlier. https://discordapp.com/oauth2/authorize?&client_id={CLIENT_ID}&scope=bot&permissions=268504064 
* Open the link and authorize your bot to access your server.
* Ensure the bot now appears offline on your server's general channel



## Enabling Players to talk from Discord to EQ
* (Admin-level accounts on Discord can only do the following steps.)
* Inside discord go to Server Settings.
* Go to Roles.
* Create a new role, with the name: `IGN: <username>`. The `IGN:` prefix is required for DiscordEQ to detect a player and is used to identify the player in game, For example, to identify the discord user `Xackery` as `Shin`, Create a role named `IGN: Shin`, right click the user Xackery, and assign the role to them.
* If the above user chats inside the assigned channel, their message will appear in game as `Shin says from discord, 'Their Message Here'`

