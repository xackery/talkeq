package userdb

import (
	"fmt"
	"testing"
)

func Test_reload(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "reload", wantErr: true},
		{name: "user_test.txt", path: "test/user_test.txt", wantErr: false},
		{name: "user_test.toml", path: "test/user_test.toml", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usersDatabasePath = tt.path
			if err := reload(); (err != nil) != tt.wantErr {
				t.Errorf("reload() error = %v, wantErr %v", err, tt.wantErr)
			}
			mu.RLock()
			fmt.Printf("%s: %+v\n", tt.path, users)
			mu.RUnlock()
		})
	}
}
