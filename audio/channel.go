package audio

import "sync"

type ChannelId int

const (
	ChannelIdDefault  ChannelId = iota
	ChannelIdMusic    ChannelId = iota
	ChannelIdAmbience ChannelId = iota
	ChannelIdSfx      ChannelId = iota
	ChannelIdUi       ChannelId = iota
	ChannelIdDialog   ChannelId = iota
	// ChannelIdLast is for when you want to define additional channels yourself
	ChannelIdLast ChannelId = iota
)

type channelSettings struct {
	volume float32
	paused bool
}

func (cid ChannelId) SetVolume(volume float32) {
	settings := getChannelSettings(cid)
	settings.volume = volume
	setChannelSettings(cid, settings)
}

func (cid ChannelId) Pause() {
	settings := getChannelSettings(cid)
	settings.paused = true
	setChannelSettings(cid, settings)
}

func (cid ChannelId) Resume() {
	settings := getChannelSettings(cid)
	settings.paused = false
	setChannelSettings(cid, settings)
}

var channelSettingsMap = make(map[ChannelId]channelSettings)
var settingsLock sync.RWMutex

func getChannelSettings(id ChannelId) channelSettings {
	settingsLock.RLock()
	s, ok := channelSettingsMap[id]
	settingsLock.RUnlock()
	if ok {
		return s
	}
	return channelSettings{
		volume: 1,
	}
}
func setChannelSettings(id ChannelId, s channelSettings) {
	settingsLock.Lock()
	channelSettingsMap[id] = s
	settingsLock.Unlock()
}
