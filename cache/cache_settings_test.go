package cache

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lomik/go-carbon/logging"
	"github.com/stretchr/testify/assert"
)

type SettingsTestCase struct {
	FieldName string
	Valid     []interface{}
	Invalid   []interface{}
	Setter    func(interface{}, interface{}) // param: settings, newValue
	Validate  func(interface{}, interface{}) // param: cache, newValue
}

func TestCacheSettings(t *testing.T) {
	assert := assert.New(t)

	table := []SettingsTestCase{
		SettingsTestCase{
			"MaxSize",
			[]interface{}{0, 1, 100, 10000000},
			[]interface{}{},
			func(setting interface{}, value interface{}) {
				setting.(*Settings).MaxSize = value.(int)
			},
			nil,
		},
		SettingsTestCase{
			"GraphPrefix",
			[]interface{}{"carbon", "graphite"},
			[]interface{}{},
			func(setting interface{}, value interface{}) {
				setting.(*Settings).GraphPrefix = value.(string)
			},
			nil,
		},
		SettingsTestCase{
			"InputCapacity",
			[]interface{}{0, 2, 200, 20000000},
			[]interface{}{},
			func(setting interface{}, value interface{}) {
				setting.(*Settings).InputCapacity = value.(int)
			},
			func(cache interface{}, value interface{}) {
				assert.Equal(value.(int), cache.(*Cache).In().Size())
			},
		},
		SettingsTestCase{
			"OutputCapacity",
			[]interface{}{0, 3, 300, 30000000},
			[]interface{}{},
			func(setting interface{}, value interface{}) {
				setting.(*Settings).OutputCapacity = value.(int)
			},
			func(cache interface{}, value interface{}) {
				assert.Equal(value.(int), cache.(*Cache).Out().Size())
			},
		},
	}

	test := func(isRunning bool) {
		cache := New()
		if isRunning {
			cache.Start()
			defer cache.Stop()
		}

		for _, testCase := range table {
			for _, value := range testCase.Valid {
				logging.Test(func(log *bytes.Buffer) {

					settings := cache.Settings()
					testCase.Setter(settings, value)
					assert.NoError(settings.Apply())

					assert.Contains(log.String(), fmt.Sprintf("-> %#v", value))
					if testCase.Validate != nil {
						testCase.Validate(cache, value)
					}
				})
			}
		}
	}

	// stopped and running
	for _, isRunning := range []bool{false, true} {
		test(isRunning)
	}
}
