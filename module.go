package tinyraft

// Module required client funcs to implement raft consensus module
// using custom implementation
type Module interface {
	Apply([]byte) error
}
