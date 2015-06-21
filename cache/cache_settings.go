package cache

// settingsQuery ...
type settingsQuery struct {
	Settings  *Settings      // set new settings if != nil
	ReplyChan chan *Settings // result of query
}

func (c *Cache) handleSettings(newSettings *Settings) *Settings {
	if newSettings != nil {
	}
	copy := *c.settings
	return &copy
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
	c.settingsChan <- q
	r := <-replyChan
	return r
}
