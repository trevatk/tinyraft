package tinyraft

import "errors"

// ErrNoCommit
var ErrNoCommit = errors.New("do not commit log entry")
