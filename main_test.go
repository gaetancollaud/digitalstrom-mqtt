package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRequestReadsPasswordFile(t *testing.T) {
	passwordFile := filepath.Join(t.TempDir(), "password")
	require.NoError(t, os.WriteFile(passwordFile, []byte("secret\n"), 0o600))

	password, err := (apiKeyRequest{passwordFile: passwordFile}).readPassword()

	assert.NoError(t, err)
	assert.Equal(t, "secret", password)
}

func TestAPIKeyRequestRejectsPasswordAndPasswordFile(t *testing.T) {
	_, err := (apiKeyRequest{password: "secret", passwordFile: "password"}).readPassword()

	assert.EqualError(t, err, "password and password-file cannot be used together")
}

func TestWritePrivateFileReplacesExistingValueWithRestrictedPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api-key")
	require.NoError(t, os.WriteFile(path, []byte("old-key"), 0o644))

	require.NoError(t, writePrivateFile(path, "new-key"))

	contents, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new-key", string(contents))

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestWritePrivateFileDoesNotReplaceTargetWhenRenameFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api-key")
	require.NoError(t, os.Mkdir(path, 0o700))

	err := writePrivateFile(path, "new-key")

	require.Error(t, err)
	info, statErr := os.Stat(path)
	require.NoError(t, statErr)
	assert.True(t, info.IsDir())
}
