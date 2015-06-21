package cache

import (
	"time"

	"github.com/Sirupsen/logrus"
)

// settingsQuery ...
type settingsQuery struct {
	Settings  *Settings      // set new settings if != nil
	ReplyChan chan *Settings // result of query
}

func (c *Cache) handleSettings(newSettings *Settings) *Settings {
	if newSettings != nil {
		if newSettings.MaxSize != c.settings.MaxSize {
			logrus.WithFields(logrus.Fields{
				"old": c.settings.MaxSize,
				"new": newSettings.MaxSize,
			}).Info("[cache] cache.MaxSize changed")

			c.settings.MaxSize = newSettings.MaxSize
		}

		if newSettings.GraphPrefix != c.settings.GraphPrefix {
			logrus.WithFields(logrus.Fields{
				"old": c.settings.GraphPrefix,
				"new": newSettings.GraphPrefix,
			}).Info("[cache] cache.GraphPrefix changed")

			c.settings.GraphPrefix = newSettings.GraphPrefix
		}
	}
	copy := *c.settings
	return &copy
}

func (c *Cache) handleSettingsQuery(query *settingsQuery) {
	query.ReplyChan <- c.handleSettings(query.Settings)
}

// Settings returns copy of cache settings object
func (c *Cache) Settings(newSettings *Settings) *Settings {
	if !c.isRunning {
		return c.handleSettings(newSettings)
	}

	replyChan := make(chan *Settings)
	q := &settingsQuery{
		Settings:  newSettings,
		ReplyChan: replyChan,
	}

	select {
	case c.settingsChan <- q:
		break
	case <-time.After(1 * time.Second):
		logrus.Error("Settings query timeout")
		return nil
	}

	var r *Settings
	select {
	case r = <-replyChan:
		break
	case <-time.After(1 * time.Second):
		logrus.Error("Settings reply wait timeout")
	}

	return r
}

// EditSettings calls callback with settings instance. Not raises any error on change settings timeout
func (c *Cache) EditSettings(callback func(*Settings)) {
	settings := c.Settings(nil)
	if settings != nil {
		callback(settings)
		c.Settings(settings)
	}
}
