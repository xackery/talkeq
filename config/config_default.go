package config

var defaultConfig = `# TalkEQ Configuration

# Enable debug when a crash occurs that is not self apparent
# Not recommended on normal use, very verbose
debug = false

# Keep all connections alive? 
# If false, endpoint disconnects will not self repair
# Not recommended to turn off except in advanced cases
keep_alive = true

# How long before retrying to connect (requires keep_alive = true)
# default: 10s
keep_alive_retry = "10s"

# Users by ID are mapped to their display names via the raw text file called users database
# If users database file does not exist, a new one is created
# This file is actively monitored. if you edit it while talkeq is running, it will reload the changes instantly
# This file overrides the IGN: playerName role tags in discord
# If a user is found on this list, it will fall back to check for IGN tags
users_database = "./users.txt"

# ** Only supported by NATS **
# Guilds by ID are mapped to their database ID via the raw text file called guilds database
# If guilds database file does not exist, and NATS is enabled, a new one is created
# This file is actively monitored. if you edit it while talkeq is running, it will reload the changes instantly
guilds_database = "./guilds.txt"

[discord]

	# Enable Discord
	enabled = true

	# Status to show below bot. e.g. "Playing EQ: 123 Online"
	# {{.PlayerCount}} to show playercount
	bot_status = "EQ: {{.PlayerCount}} Online"

	# Required. Found at https://discordapp.com/developers/ under your app's main page
	client_id = ""

	# Required. Found at https://discordapp.com/developers/ under your app's bot's section
	bot_token = ""

	# Required. In Discord, right click the circle button representing your server, and Copy ID, and paste it here.
	server_id = ""

	[discord.ooc]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# Out of character messages will appear on this discord channel
		send_channel_id = ""
		listen_channel_id = ""

	[discord.auction]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# Out of character messages will appear on this discord channel
		send_channel_id = ""
		listen_channel_id = ""
	
	[discord.guild]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# guild chat messages will appear on this discord channel
		# Note: not supported with telnet (eqemu) at this time
		send_channel_id = ""
		listen_channel_id = ""
	
	[discord.shout]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# shout messages will appear on this discord channel
		# Note: not supported with telnet (eqemu) at this time
		send_channel_id = ""
		listen_channel_id = ""
	
	[discord.general]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# general chat messages will appear on this discord channel
		# Note: not supported with telnet (eqemu) at this time
		send_channel_id = ""
		listen_channel_id = ""

	[discord.admin]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# admin messages will appear on this discord channel
		# Note: not supported with telnet (eqemu) at this time
		send_channel_id = ""
		listen_channel_id = ""

	[discord.peq_editor_sql_log]
		# Optional. In Discord, right click a channel name and Copy ID. Paste it here.
		# if you use the PEQ editor, and have sql log enabled, this is the channel it sends to
		# Note: requires peq editor settings to be set
		send_channel_id = ""

[telnet]

	# Enable Telnet (eqemu server owners)
	enabled = false

	# if you are using a very old version of telnet, enable this.
	# default: false 
	legacy = false
	
	# Optional. (eqemu server owners). Specify where telnet is located. 
	# Akka's installer by default will use 127.0.0.1:9000
	host = "127.0.0.1:9000"

	# Optional. Username to connect to telnet with.
	# If you run talkeq on the same server as you run eqemu, 
	# by default username and password fields are not used or required (telnet listens to localhost only)
	username = ""

	# Optional. Password to connect to telnet with.
	# If you run talkeq on the same server as you run eqemu, 
	# by default username and password fields are not used or required (telnet listens to localhost only)
	password = ""

	# Optional. Converts item URLs to provided field. defaults to allakhazam. To disable, change to ""
	# default: "http://everquest.allakhazam.com/db/item.html?item="
	item_url = "http://everquest.allakhazam.com/db/item.html?item="

	# Optional. Annunce when a server changes state to OOC channel (Server UP/Down)
	announce_server_status = true

	# How long to wait for messages. (Advanced users only)
	# defaut: 10s
	message_deadline = "10s"

	# if a OOC message uses prefix WTS or WTB, convert them into auction
	convert_ooc_auction = true

# EQ Log is used to parse everquest client logs. Primarily for live EQ, non server owners
[eqlog]

	# Enable EQ client EQLog parsing
	enabled = false

	# Path to find EQ Logs. Supports both / and \\, but not single \
	# (If you copy paste from Explorer, be sure to escape all backslashes)
	# example: c:\\Program Files\\Everquest\\Logs\\eqlog_CharacterName_Server.txt
	path = ""

	# if a general chat message uses prefix WTS or WTB, convert them into auction
	convert_general_auction = true

	# listen for /auction (auction messages)
	listen_auction = true

	# listen for /ooc (out of character messages)
	listen_ooc = true

	# Listen for /1 (general chat messages)
	listen_general = true

	# Listen for /shout (shout messages)
	listen_shout = true

	# Listen for /guild (guild messages)
	listen_guild = true


# NATS is a custom alternative to telnet 
# that a very limited number of eqemu
# servers utilize. Chances are, you can ignore.
[nats]

	# Enable NATS (eqemu server owners)
	enabled = false
	
	# Specify where NATS is located. 
	# default 127.0.0.1:4222
	host = "127.0.0.1:4222"

	# if a OOC message uses prefix WTS or WTB, convert them into auction
	convert_ooc_auction = true

	# Optional. Converts item URLs to provided field. defaults to allakhazam. To disable, change to ""
	# default: "http://everquest.allakhazam.com/db/item.html?item="
	item_url = "http://everquest.allakhazam.com/db/item.html?item="
	
[peq_editor.sql]

	# Enable PEQ Editor SQL log parsing
	enabled = false

	# SQL Directory Path to find SQL Logs
	# default: /var/www/peq/peqphpeditor/logs
	path = "/var/www/peq/peqphpeditor/logs"

	# File Pattern of SQL Log files, only needs to be changed if you edit it to a custom value 
	# default: sql_log_{{.Month}}-{{.Year}}.sql
	file_pattern = "sql_log_{{.Month}}-{{.Year}}.sql"

# SQL Report can be used to show stats on discord
# An ideal way to set this up is create a private voice channel
# Then bind it to various queries

[sql_report]
	# Enable SQL Reporting
	enabled = false

	# host for database
	# default: 127.0.0.1:3306
	host = "127.0.0.1:3306"

	# username to connect to database with.
	# default: eqemu
	username = "eqemu"

	# password to connect to database with.
	# default: eqemupass
	password = "eqemupass"

	# database to connect to
	# default: eqemu
	database = "eqemu"
	
[[sql_report.entries]]
	# Voice channel id to show the results on
	channel_id = "676282331627257856"

	# SQL Query to run
	query = "SELECT count(id) FROM accounts;"

	# Pattern to show on channel. 
	# Variables: {{.Data}}
	pattern = "Accounts: {{.Data}}"

	# how often to run the query. Minium: 30s
	refresh = "30m"

[[sql_report.entries]]
	channel_id = "676282331627257856"
	query = "SELECT level2 FROM character_data cd INNER JOIN account a ON a.id = cd.account_id WHERE a.status = 0 ORDER BY level2 DESC LIMIT 1"
	pattern = "Best Run: {{.Data}}"
	refresh = "5m"

[[sql_report.entries]]
	channel_id = "678525065905831968"
	query = "SELECT count(id) FROM character_data WHERE zone_id != 386"
	pattern = "In Dungeon: {{.Data}}"
	refresh = "60s"

[[sql_report.entries]]
	channel_id = "678525065905831968"
	query = "SELECT count(id) FROM character_data WHERE zone_id = 386"
	pattern = "In Hub: {{.Data}}"
	refresh = "60s"
`
