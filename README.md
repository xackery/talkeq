# TalkEQ

[![GoDoc](https://godoc.org/github.com/xackery/talkeq?status.svg)](https://godoc.org/github.com/xackery/talkeq) [![Go Report Card](https://goreportcard.com/badge/github.com/xackery/talkeq)](https://goreportcard.com/report/github.com/xackery/talkeq) [![Build Status](https://travis-ci.org/xackery/talkeq.svg)](https://travis-ci.org/Xackery/talkeq.svg?branch=master) [![Coverage Status](https://coveralls.io/repos/github/xackery/talkeq/badge.svg?branch=master)](https://coveralls.io/github/xackery/talkeq?branch=master)

TalkEQ bridges your conversations from Everquest to Discord.


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


## Talk To and From Everquest

TalkEQ bridges links between everquest and other services. Based as a rewrite of [DiscordEQ](https://github.com/xackery/discordeq).

### Supported Endpoints

* Discord
* Telnet (eqemu server)

### Work In Progress Endpoints

* eqlog (logging via everquest's log system)