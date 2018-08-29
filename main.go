package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/thehadalone/metachat/metachat"
	"github.com/thehadalone/metachat/skype"
	"github.com/thehadalone/metachat/slack"
	"github.com/thehadalone/metachat/telegram"
)

type config struct {
	metachat.Config
	Skype    skype.Config    `json:"skype"`
	Slack    slack.Config    `json:"slack"`
	Telegram telegram.Config `json:"telegram"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Config file must be provided")
		os.Exit(1)
	}

	configPath := os.Args[1]
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	var config config
	err = json.Unmarshal(content, &config)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	skypeClient, err := skype.NewClient(config.Skype)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	slackClient, err := slack.NewClient(config.Slack)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	telegramClient, err := telegram.NewClient(config.Telegram)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	config.Config.Messengers = []metachat.Messenger{skypeClient, slackClient, telegramClient}

	meta, err := metachat.New(config.Config)
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}

	err = meta.Start()
	if err != nil {
		fmt.Printf("%+v", err)
		os.Exit(1)
	}
}
