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

[telnet]

	# Enable Telnet (eqemu server owners)
	enabled = false

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

[eqlog]

	# Enable EQLog parsing
	enabled = false

	# Path to find EQ Logs. Supports both / and \\, but not single \
	# (If you copy paste from Explorer, be sure to escape all backslashes)
	# example: c:\\Program Files\\Everquest\\Logs)
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
`
