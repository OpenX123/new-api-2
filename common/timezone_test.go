package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalTimezoneIsShanghai(t *testing.T) {
	name, offset := time.Now().Zone()

	assert.Equal(t, "Asia/Shanghai", name)
	assert.Equal(t, 8*60*60, offset)
}
