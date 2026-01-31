package viewers

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestPrettyJSONError(t *testing.T) {
	t.Parallel()
	input := []byte("{invalid}")
	_, err := prettyJSON(input)
	assert.Error(t, err)
}

func TestPrettyJSONUsesTwoSpaceIndent(t *testing.T) {
	t.Parallel()
	input := []byte("{\"a\":{\"b\":1}}")
	output, err := prettyJSON(input)
	assert.NoError(t, err)

	expected := "{\n  \"a\": {\n    \"b\": 1\n  }\n}"
	outputText := string(output)
	assert.Equal(t, expected, outputText)
}
