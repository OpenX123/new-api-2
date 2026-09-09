package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewChannelSortOptionsAcceptsPinyin(t *testing.T) {
	options := NewChannelSortOptions("pinyin", "asc", false)

	require.True(t, options.IsPinyinSort())
	require.Equal(t, "asc", options.SortOrder)
}

func TestSortChannelsByPinyin(t *testing.T) {
	channels := []*Channel{
		{Id: 1, Name: "南京"},
		{Id: 2, Name: "北京"},
		{Id: 3, Name: "上海"},
		{Id: 4, Name: "广州"},
		{Id: 5, Name: "阿里"},
	}

	SortChannelsByPinyin(channels, false)

	require.Equal(t, []string{"阿里", "北京", "广州", "南京", "上海"}, []string{
		channels[0].Name,
		channels[1].Name,
		channels[2].Name,
		channels[3].Name,
		channels[4].Name,
	})

	SortChannelsByPinyin(channels, true)

	require.Equal(t, []string{"上海", "南京", "广州", "北京", "阿里"}, []string{
		channels[0].Name,
		channels[1].Name,
		channels[2].Name,
		channels[3].Name,
		channels[4].Name,
	})
}
