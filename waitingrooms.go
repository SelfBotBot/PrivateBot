package PrivateBot

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

// Data abstract containing methods for saving and loading.

// ConfigSaveLocation the location to save the config to.
var SaveLocation = "config.json"

// DefaultConfigSavedError an error returned if the default config is saved.
var DefaultConfigSavedError = errors.New("the default config has been saved, please edit it")

// Data an abstract struct used for it's functions to save and load config files.
type Data struct{}

func (d *Data) save(saveLoc string, inter interface{}) error {
	// Make all the directories
	if err := os.MkdirAll(filepath.Dir(saveLoc), os.ModeDir|0775); err != nil {
		return err
	}

	data, err := json.MarshalIndent(inter, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(saveLoc, data, 0660)
}

func (d *Data) load(saveLoc string, inter interface{}) error {

	if _, err := os.Stat(saveLoc); os.IsNotExist(err) {
		return DefaultConfigSavedError
	} else if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(saveLoc)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, inter); err != nil {
		return err
	}

	return nil

}

var DefaultConfig = &WaitingRooms{
	Data{}, make(map[string]string), "TOKEN", new(sync.Mutex),
}

type WaitingRooms struct {
	Data
	Rooms map[string]string
	Token string
	lock  *sync.Mutex
}

// Save saves the config.
func (r *WaitingRooms) Save() error {
	saveLoc, envThere := os.LookupEnv("CONFIG_LOC")
	if !envThere {
		saveLoc = SaveLocation
	}

	return r.save(saveLoc, r)
}

// Load loads the config.
func (r *WaitingRooms) Load() error {

	saveLoc, envThere := os.LookupEnv("CONFIG_LOC")
	if !envThere {
		saveLoc = SaveLocation
	}

	if err := r.load(saveLoc, r); err == DefaultConfigSavedError {
		if err := DefaultConfig.Save(); err != nil {
			return err
		}
		return DefaultConfigSavedError
	} else if err != nil {
		return err
	}

	return nil

}

func (r *WaitingRooms) AddRoom(GuildID string, ChannelID string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Rooms[GuildID] = ChannelID
	return r.Save()

}

func (r *WaitingRooms) GetRoom(GuildID string) (string, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	channel, ok := r.Rooms[GuildID]
	return channel, ok
}