package filetug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShowHelpModal(t *testing.T) {
	t.Parallel()
	nav, _, _ := newNavigatorForTest(t)
	showHelpModal(nav)
	assert.NotNil(t, nav)
}
