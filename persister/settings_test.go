package persister

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/lomik/go-carbon/helper"
	"github.com/stretchr/testify/assert"
)

var SchemasOK = `
[carbon]
pattern = ^carbon\.
retentions = 60s:90d
`

var SchemasFail = `
[carbon]
pattern = ^carbon\.
retentions = 60v:90d
`

var AggregationOK = `
[carbon]
pattern = ^carbon\.
xFilesFactor = 0.1
aggregationMethod = min
`

var AggregationFail = `
[carbon]
pattern = ^carb(on\.
xFilesFactor = 0.1
aggregationMethod = min
`

func TestLoadAndValidate(t *testing.T) {
	assert := assert.New(t)

	writeFile := func(filename string, content string) {
		ioutil.WriteFile(filename, []byte(content), 0644)
	}

	schemas := []string{
		"file empty",
		"file not exists",
		"fail",
		"ok",
	}

	aggr := []string{
		"file empty",
		"file not exists",
		"fail",
		"ok",
	}

	for _, enabled := range []bool{false, true} {
		for _, ss := range schemas {
			for _, aa := range aggr {
				helper.Root(t, func(root string) {
					s := NewSettings()
					s.Enabled = enabled

					hasFail := false

					switch ss {

					case "file empty":
						s.SchemasFile = ""
						hasFail = true

					case "file not exists":
						s.SchemasFile = filepath.Join(root, "schemas.conf")
						hasFail = true

					case "fail":
						s.SchemasFile = filepath.Join(root, "schemas.conf")
						writeFile(s.SchemasFile, SchemasFail)
						hasFail = true

					case "ok":
						s.SchemasFile = filepath.Join(root, "schemas.conf")
						writeFile(s.SchemasFile, SchemasOK)
					}

					switch aa {

					case "file empty":
						s.AggregationFile = ""

					case "file not exists":
						s.AggregationFile = filepath.Join(root, "aggregation.conf")
						hasFail = true

					case "fail":
						s.AggregationFile = filepath.Join(root, "aggregation.conf")
						writeFile(s.AggregationFile, AggregationFail)
						hasFail = true

					case "ok":
						s.AggregationFile = filepath.Join(root, "aggregation.conf")
						writeFile(s.AggregationFile, AggregationOK)
					}

					msg := fmt.Sprintf("Enabled: %#v, Schemas: %s, Aggregation %s", enabled, ss, aa)

					if enabled && hasFail {
						assert.Error(s.LoadAndValidate(), msg)
					} else {
						assert.NoError(s.LoadAndValidate(), msg)
					}

					if enabled && !hasFail {
						assert.NotNil(s.schemas, msg)
						assert.NotNil(s.aggregation, msg)
					}
				})
			}
		}
	}
}
