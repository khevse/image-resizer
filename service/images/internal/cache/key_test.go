package cache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKey(t *testing.T) {
	require.Equal(t, "098f6bcd4621d373cade4e832627b4f6", NewKey("test"))
}
