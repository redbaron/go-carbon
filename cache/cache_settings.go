package cache

import "github.com/Sirupsen/logrus"

// Settings returns copy of cache settings object
func (c *Cache) Settings(newSettings *Settings) *Settings {

	if newSettings == nil { // read-only
		c.settings.RLock()
		defer c.settings.RUnlock()

		s := *c.settings
		return &s
	}

	c.settings.Lock()
	defer c.settings.Unlock()

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

	changed := c.settings.changed
	c.settings.changed = make(chan bool)
	close(changed)

	s := *c.settings

	return &s
}

// EditSettings calls callback with settings instance. Not raises any error on change settings timeout
func (c *Cache) EditSettings(callback func(*Settings)) {
	settings := c.Settings(nil)
	if settings != nil {
		callback(settings)
		c.Settings(settings)
	}
}
