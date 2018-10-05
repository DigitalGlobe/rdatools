package cmd

import (
	"log"
	"os/user"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"golang.org/x/oauth2"
)

func NewConfig() Config {
	// Get GBDX creds.
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	credPath := filepath.Join(usr.HomeDir, ".gbdx", "credentials.toml")
	confMap := make(map[string]Config)
	if _, err := toml.DecodeFile(credPath, &confMap); err != nil {
		log.Fatalln("failed decoding credentials", err)
	}
	conf, ok := confMap["default"]
	if !ok {
		log.Fatalln("no default gbdx credentials found to use")
	}
	return conf
}

type Config struct {
	Username string        `mapstructure:"gbdx_username" toml:"gbdx_username"`
	Password string        `mapstructure:"gbdx_password" toml:"gbdx_password"`
	Token    *oauth2.Token `mapstructure:"gbdx_token" toml:"gbdx_token"`
}
