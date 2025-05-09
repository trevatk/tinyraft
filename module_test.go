package tinyraft

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testModule struct {
	mtx   sync.Mutex
	words []string
}

// interface compliance
var _ Module = (*testModule)(nil)

func newTestModule() *testModule {
	return &testModule{
		mtx:   sync.Mutex{},
		words: []string{},
	}
}

// Apply
func (tm *testModule) Apply(cmd []byte) error {
	tm.mtx.Lock()
	defer tm.mtx.Unlock()
	w := string(cmd)
	tm.words = append(tm.words, w)
	return nil
}

func TestModule(t *testing.T) {
	assert := assert.New(t)

	tm := newTestModule()
	t.Run("helloworld", func(t *testing.T) {
		helloWorld := []byte("hello world")
		assert.NoError(tm.Apply(helloWorld))
	})
}
