package main

import (
	"encoding/json"
	"testing"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/stretchr/testify/assert"
)

func TestSnowflakeLookup_MarshalJSON(t *testing.T) {
	lookup := snowflakeLookup{
		discord.Snowflake(1337): struct{}{},
		discord.Snowflake(42):   struct{}{},
		discord.Snowflake(777):  struct{}{},
	}

	d, err := json.Marshal(lookup)
	assert.NoError(t, err)
	assert.EqualValues(t, `["42","777","1337"]`, string(d))
}

func TestSnowflakeLookup_UnmarshalJSON(t *testing.T) {
	lookup := make(snowflakeLookup)
	input := []byte(`["1337","42","777"]`)

	err := json.Unmarshal(input, &lookup)
	assert.NoError(t, err)

	expected := snowflakeLookup{
		discord.Snowflake(1337): struct{}{},
		discord.Snowflake(42):   struct{}{},
		discord.Snowflake(777):  struct{}{},
	}

	assert.Equal(t, expected, lookup)
}

func TestConfigFromBytes(t *testing.T) {
	input := []byte(`
{
	"prefix": "dr.",
	"permissions": {
		"docs": [
			"1337"
		],
		"config": {
			"42": [
				"777"
			]
		}
	}
}
`)

	config, err := configFromBytes(input)
	assert.NoError(t, err)

	expected := configuration{
		Prefix: "dr.",
		Permissions: commandPermissions{
			Docs: map[discord.Snowflake]struct{}{
				1337: {},
			},
			Config: map[discord.GuildID]snowflakeLookup{
				42: map[discord.Snowflake]struct{}{
					777: {},
				},
			},
		},
		Aliases:   map[string]string{},
		Blacklist: map[discord.Snowflake]struct{}{},
	}

	assert.Equal(t, expected, config)
}
