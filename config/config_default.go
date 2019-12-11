package config

var defaultConfig = `# TalkEQ Configuration

# Enable debug when a crash occurs that is not self apparant
# Not recommended on normal use, very verbose
debug = false

# Keep all connections alive? 
# If false, endpoint disconnects will not self repair
# Not recommended to turn off except in advanced cases
keep_alive = true

[discord]

# Enable Discord
enabled = true

# Required. Found at https://discordapp.com/developers/ under your app's main page
client_id = ""

# Required. Found at https://discordapp.com/developers/ under your app's bot's section
bot_token = ""

# Required. In Discord, right click the circle button representing your server, and Copy ID, and paste it here.
server_id = ""

# Required. In Discord, right click a channel name and Copy ID. Paste it here.
# Out of character messages will appear on this discord channel
ooc_send_channel_id = ""

# Optional. (eqemu server owners) Which channel to listen for out of character messages
# Will relay them via telnet to your EQEMU server
ooc_listen_channel_id = ""

[telnet]

# Enable Telnet (eqemu server owners)
enabled = false

# Optional. (eqemu server owners). Specify where telnet is located. 
# Akka's installer by default will use 127.0.0.1:22
host = "127.0.0.1:22"

# Optional. (eqemu server owners) 
# If you run talkeq on the same server as you run eqemu, 
# by default username and password fields are not used or required (telnet listens to localhost only)
username = ""

# Optional. (eqemu server owners)
password = ""

# Optional. Converts item URLs to provided field. defaults to allakhazam. To disable, change to ""
# default: "http://everquest.allakhazam.com/db/item.html?item="
item_url = "http://everquest.allakhazam.com/db/item.html?item="


[eqlog]

# Enable EQLog parsing
enabled = false

# Path to find EQ Logs. Supports both / and \\, but not \
# (If you copy paste from Explorer, be sure to double pad all backslashes, e.g. c:\\Program Files\\Everquest\\Logs)
path = ""

# Optional. Converts item URLs to provided field. defaults to allakhazam. To disable, change to ""
# default: "http://everquest.allakhazam.com/db/item.html?item="
item_url = "http://everquest.allakhazam.com/db/item.html?item="


`
