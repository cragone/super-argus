package chunk

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// testStoreContract exercises the behavior every Store implementation must
// provide, regardless of backend: Put/Get round-trip, idempotent Delete, a
// not-found Get satisfying errors.Is(err, os.ErrNotExist), and rejection of
// malformed keys.
func testStoreContract(t *testing.T, newStore func() Store) {
	t.Helper()
	s := newStore()
	key := "2023/11/14/7/abc.chk"

	require.NoError(t, s.Put(key, []byte("hello")))
	b, err := s.Get(key)
	require.NoError(t, err)
	require.Equal(t, "hello", string(b))

	require.NoError(t, s.Delete(key))
	require.NoError(t, s.Delete(key)) // idempotent
	_, err = s.Get(key)
	require.ErrorIs(t, err, os.ErrNotExist)

	for _, bad := range []string{"", "../x", "a/../../x", "/abs", "a//b", `a\b`} {
		require.Error(t, s.Put(bad, nil), bad)
	}
}
