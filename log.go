package tinyraft

// Log
type Log struct {
	Term  uint64 `json:"term"`
	Index uint64 `json:"index"`
	Cmd   []byte `json:"cmd"`
}

func newLog(term, index uint64, cmd []byte) Log {
	return Log{
		Term:  term,
		Index: index,
		Cmd:   cmd,
	}
}
