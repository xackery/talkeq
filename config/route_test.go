package config

import (
	"reflect"
	"testing"
	"text/template"
)

func TestRoute_MessagePatternTemplate(t *testing.T) {
	type fields struct {
		IsEnabled              bool
		Trigger                Trigger
		Target                 string
		ChannelID              string
		GuildID                string
		MessagePattern         string
		messagePatternTemplate *template.Template
	}
	tests := []struct {
		name   string
		fields fields
		want   *template.Template
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Route{
				IsEnabled:              tt.fields.IsEnabled,
				Trigger:                tt.fields.Trigger,
				Target:                 tt.fields.Target,
				ChannelID:              tt.fields.ChannelID,
				GuildID:                tt.fields.GuildID,
				MessagePattern:         tt.fields.MessagePattern,
				messagePatternTemplate: tt.fields.messagePatternTemplate,
			}
			if got := r.MessagePatternTemplate(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Route.MessagePatternTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoute_LoadMessagePattern(t *testing.T) {
	type fields struct {
		IsEnabled              bool
		Trigger                Trigger
		Target                 string
		ChannelID              string
		GuildID                string
		MessagePattern         string
		messagePatternTemplate *template.Template
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Route{
				IsEnabled:              tt.fields.IsEnabled,
				Trigger:                tt.fields.Trigger,
				Target:                 tt.fields.Target,
				ChannelID:              tt.fields.ChannelID,
				GuildID:                tt.fields.GuildID,
				MessagePattern:         tt.fields.MessagePattern,
				messagePatternTemplate: tt.fields.messagePatternTemplate,
			}
			if err := r.LoadMessagePattern(); (err != nil) != tt.wantErr {
				t.Errorf("Route.LoadMessagePattern() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
