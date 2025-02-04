package internal

import "encoding/json"

type EncoderJSON struct{}

func (e EncoderJSON) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (e EncoderJSON) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type encoder interface {
	Serialize(v interface{}) ([]byte, error)
	Deserialize(data []byte, v interface{}) error
}

// global encoder used for events
var EventEncoder encoder = EncoderJSON{}

// global encoder used for snapshots
var SnapshotEncoder encoder = EncoderJSON{}
