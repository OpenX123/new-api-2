package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserGetRatioNormalization(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	tests := []struct {
		name  string
		ratio *float64
		want  float64
	}{
		{"nil means default 1.0", nil, 1.0},
		{"zero normalized to 1.0", f(0), 1.0},
		{"negative normalized to 1.0", f(-2), 1.0},
		{"normal value kept", f(0.8), 0.8},
		{"greater than one kept", f(2.5), 2.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Ratio: tt.ratio}
			assert.Equal(t, tt.want, u.GetRatio())
			b := &UserBase{Ratio: tt.ratio}
			assert.Equal(t, tt.want, b.GetRatio())
		})
	}
}

func TestToBaseUserCarriesRatio(t *testing.T) {
	f := 0.8
	u := &User{Id: 1, Ratio: &f}
	assert.Equal(t, &f, u.ToBaseUser().Ratio)
}
