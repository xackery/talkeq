package model

import "regexp"

// Alphanumeric sanitizes incoming data to only be valid
func Alphanumeric(data string) (sData string) {
	sData = data
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	sData = re.ReplaceAllString(sData, "")
	return
}
