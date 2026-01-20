package sticky

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTable(t *testing.T) {
	table := NewTable([]Column{})
	assert.NotNil(t, table)
}
