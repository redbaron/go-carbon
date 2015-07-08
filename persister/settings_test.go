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

func TestLoadAndValidate(t *testing.T) {
	assert := assert.New(t)

	writeFile := func(filename string, content string) {
		ioutil.WriteFile(filename, []byte(content), 0644)
	}

	schemas := []string{
		"schemas file empty",
		"schemas file not exists",
		"schemas broken",
		"schemas ok",
	}

	aggr := []string{
		"aggregation file empty",
		"aggregation file not exists",
		"aggregation broken",
		"aggregation ok",
	}

	for _, ss := range schemas {
		for _, aa := range aggr {
			helper.Root(t, func(root string) {
				s := NewSettings()

				hasFail := false

				switch ss {

				case "schemas file empty":
					s.SchemasFile = ""
					hasFail = true

				case "schemas file not exists":
					s.SchemasFile = filepath.Join(root, "schemas.conf")
					hasFail = true

				case "schemas broken":
					s.SchemasFile = filepath.Join(root, "schemas.conf")
					writeFile(s.SchemasFile, SchemasFail)
					hasFail = true

				case "schemas ok":
					s.SchemasFile = filepath.Join(root, "schemas.conf")
					writeFile(s.SchemasFile, SchemasOK)
				}

				switch aa {

				case "aggregation file empty":
					s.AggregationFile = ""

				case "aggregation file not exists":
					s.AggregationFile = filepath.Join(root, "aggregation.conf")
					hasFail = true

				case "aggregation broken":
					s.AggregationFile = filepath.Join(root, "aggregation.conf")
					writeFile(s.AggregationFile, SchemasFail)
					hasFail = true

				case "aggregation ok":
					s.AggregationFile = filepath.Join(root, "aggregation.conf")
					writeFile(s.AggregationFile, SchemasOK)
				}

				if hasFail {
					assert.Error(s.LoadAndValidate(), fmt.Sprintf("%s; %s", ss, aa))
				} else {
					assert.NoError(s.LoadAndValidate(), fmt.Sprintf("%s; %s", ss, aa))
				}
			})
		}
	}
}
