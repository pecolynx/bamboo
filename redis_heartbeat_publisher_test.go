package bamboo_test

import (
	"context"
	"testing"

	"github.com/pecolynx/bamboo"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func Test_redisBambooHeartbeatPublisher_Ping(t *testing.T) {
	ctx := context.Background()
	type inputs struct {
		addr string
	}
	type outputs struct {
		pingError error
	}
	tests := []struct {
		name    string
		inputs  inputs
		outputs outputs
	}{
		{
			name: "valid port",
			inputs: inputs{
				addr: "localhost:6380",
			},
			outputs: outputs{
				pingError: nil,
			},
		},
		{
			name: "invalid port",
			inputs: inputs{
				addr: "localhost:6381",
			},
			outputs: outputs{
				pingError: unix.ECONNREFUSED,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heartbeatPublisher := bamboo.NewRedisBambooHeartbeatPublisher(&redis.UniversalOptions{
				Addrs: []string{tt.inputs.addr},
			})
			err := heartbeatPublisher.Ping(ctx)
			if tt.outputs.pingError != nil {
				assert.ErrorIs(t, err, tt.outputs.pingError)
				return
			}
			require.Nil(t, err)
		})
	}
}
