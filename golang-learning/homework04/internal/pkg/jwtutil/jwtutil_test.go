package jwtutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_GenerateAndParse_RoundTrip(t *testing.T) {
	mgr := NewManager("test-secret", time.Hour)

	token, err := mgr.Generate(42, "alice", "admin")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := mgr.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "alice", claims.Username)
	assert.Equal(t, "admin", claims.Role)
}

func TestManager_Parse_Expired(t *testing.T) {
	mgr := NewManager("test-secret", -time.Hour) // 签发时就已经过期

	token, err := mgr.Generate(1, "bob", "user")
	require.NoError(t, err)

	_, err = mgr.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestManager_Parse_WrongSecret(t *testing.T) {
	signer := NewManager("secret-a", time.Hour)
	verifier := NewManager("secret-b", time.Hour)

	token, err := signer.Generate(1, "bob", "user")
	require.NoError(t, err)

	_, err = verifier.Parse(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestManager_Parse_Garbage(t *testing.T) {
	mgr := NewManager("test-secret", time.Hour)

	_, err := mgr.Parse("not-a-real-token")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestManager_Parse_Empty(t *testing.T) {
	mgr := NewManager("test-secret", time.Hour)

	_, err := mgr.Parse("")
	assert.ErrorIs(t, err, ErrInvalidToken)
}
