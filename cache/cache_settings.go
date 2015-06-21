package cache

import "github.com/Sirupsen/logrus"

// Settings returns copy of cache settings object
func (c *Cache) Settings(newSettings *Settings) *Settings {

	if newSettings == nil { // read-only
		c.RLock()
		defer c.RUnlock()

		s := *c.settings
		return &s
	}

	c.Lock()
	defer c.Unlock()

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

	if newSettings.InputCapacity != c.settings.InputCapacity {
		logrus.WithFields(logrus.Fields{
			"old": c.settings.InputCapacity,
			"new": newSettings.InputCapacity,
		}).Info("[cache] cache.InputCapacity changed")

		c.settings.InputCapacity = newSettings.InputCapacity
		if c.inputChan != nil {
			c.inputChan.Resize(c.settings.InputCapacity)
		}
	}

	if newSettings.OutputCapacity != c.settings.OutputCapacity {
		logrus.WithFields(logrus.Fields{
			"old": c.settings.OutputCapacity,
			"new": newSettings.OutputCapacity,
		}).Info("[cache] cache.OutputCapacity changed")

		c.settings.OutputCapacity = newSettings.OutputCapacity
		if c.outputChan != nil {
			c.outputChan.Resize(c.settings.OutputCapacity)
		}
	}

	changed := c.settingsChanged
	c.settingsChanged = make(chan bool)
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
