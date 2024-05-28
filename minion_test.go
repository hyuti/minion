package minion_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"minion"
	"testing"
	"time"
)

func TestMinion_Start(t *testing.T) {
	testCases := []struct {
		name   string
		input  func() (context.Context, context.CancelFunc)
		expect func(t *testing.T, err error)
	}{
		{
			name: "tc1: timeout error returned",
			expect: func(t *testing.T, err error) {
				require.ErrorIs(t, err, minion.ErrTimeout)
			},
			input: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Second)
			},
		},
		{
			name: "tc2: no error returned",
			expect: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			input: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Second*3)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.input()
			defer cancel()
			gru := minion.New[any]()
			gru.WithEvent(func(_ any) {
			})
			gru.AddMinionWithCtx(ctx, func(ctx context.Context) any {
				time.Sleep(2 * time.Second)
				return nil
			})
			gru.Start()
			tc.expect(t, gru.Error())
		})
	}
}
