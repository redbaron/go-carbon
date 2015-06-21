package cache

import "github.com/lomik/go-carbon/points"

/*
get, remove, pop, add functions. Called from worker goroutine
*/

// get any key/values pair from Cache
func (c *Cache) get() *points.Points {
	for {
		size := len(c.queue)
		if size == 0 {
			break
		}
		cacheRecord := c.queue[size-1]
		c.queue = c.queue[:size-1]

		if values, ok := c.data[cacheRecord.metric]; ok {
			return values
		}
	}
	for _, values := range c.data {
		return values
	}
	return nil
}

// remove key from cache
func (c *Cache) remove(key string) {
	if value, exists := c.data[key]; exists {
		c.size -= len(value.Data)
		delete(c.data, key)
	}
}

// pop return and remove next for save point from cache
func (c *Cache) pop() *points.Points {
	v := c.get()
	if v != nil {
		c.remove(v.Metric)
	}
	return v
}

// add points to cache
func (c *Cache) add(p *points.Points) {
	if values, exists := c.data[p.Metric]; exists {
		values.Data = append(values.Data, p.Data...)
	} else {
		c.data[p.Metric] = p
	}
	c.size += len(p.Data)
}

// Size returns size
// func (c *Cache) Size() int {
// 	return c.size
// }
