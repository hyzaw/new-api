package helper

import (
	"io"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

type observedReadCloser struct {
	io.ReadCloser
	info *relaycommon.RelayInfo
}

func NewObservedReadCloser(body io.ReadCloser, info *relaycommon.RelayInfo) io.ReadCloser {
	if body == nil {
		return nil
	}
	return &observedReadCloser{
		ReadCloser: body,
		info:       info,
	}
}

func (r *observedReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 && r.info != nil {
		r.info.MarkUpstreamFirstByte()
	}
	return n, err
}
