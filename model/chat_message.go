package model

// ChatMessage wraps a chat message
type ChatMessage struct {
	// SourceEndpoint is what endpoint this message originated from
	SourceEndpoint string
	// DestinationEndpoint is used by manager to send to various endpoints
	DestinationEndpoint string
	//used by discord, this is the channel ID to chat on
	ChannelID string
	Message   string
	//Channel number is used in game to represent various types of chat.
	ChannelNumber int
	//Author of the message
	Author string
}
