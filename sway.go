// Sway IPC

package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
)

// Magic is Sway message magic string.
var Magic = [6]byte{'i', '3', '-', 'i', 'p', 'c'}

// MessageType is Sway message type code as appears in the message header.
type MessageType uint32

// Needed Sway message type codes.
const (
	SUBSCRIBE  MessageType = 2
	GET_INPUTS MessageType = 100
	InputEvent             = 0x80000015
)

// A MessageHeader contains a Sway message excluding the payload.
type MessageHeader struct {
	Magic  [6]byte
	Length uint32
	Type   MessageType
}

// A MessageHandler is a callback for handling incoming Sway message.
type MessageHandler func(t MessageType, payload io.Reader) error

// ReadMessage receives the next message from the connection r and calls f. If f returns an error, ReadMessage returns that error.
func ReadMessage(r io.Reader, f MessageHandler) error {
	var h MessageHeader
	err := binary.Read(r, ByteOrder, &h)
	if err != nil {
		return err
	}
	if h.Magic != Magic {
		return errors.New("invalid magic string")
	}
	p := &io.LimitedReader{R: r, N: int64(h.Length)}
	ferr := f(h.Type, p)
	_, err = io.Copy(ioutil.Discard, p)
	if ferr != nil {
		return ferr
	}
	return err
}

// WriteEmptyMessage sends a Sway message with no payload.
func WriteEmptyMessage(w io.Writer, t MessageType) error {
	return binary.Write(w, ByteOrder, MessageHeader{Magic: Magic, Type: t})
}

// WriteJSONMessage sends a Sway message with JSON-encoded payload.
func WriteJSONMessage(w io.Writer, t MessageType, p interface{}) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	err = binary.Write(w, ByteOrder, MessageHeader{Magic: Magic, Length: uint32(len(b)), Type: t})
	if err != nil {
		return err
	}
	n, err := w.Write(b)
	if err != nil {
		return err
	}
	if n < len(b) {
		return io.ErrShortWrite
	}
	return nil
}

// ErrNoIPC is returned when Sway IPC socket path is not specified either explicitly or via the environment variable.
var ErrNoIPC = errors.New("Sway IPC socket path is unknown")

// Connect makes a connection to Sway IPC socket. If path is empty, SWAYSOCK environment variable is used.
func Connect(path string) (net.Conn, error) {
	if len(path) == 0 {
		path = os.Getenv("SWAYSOCK")
	}
	if len(path) == 0 {
		return nil, ErrNoIPC
	}
	return net.Dial("unix", path)
}

// An InputDevice contains Sway input device data.
// Only needed fields are included.
type InputDevice struct {
	Identifier          string `json:"identifier"`
	XkbActiveLayoutName string `json:"xkb_active_layout_name"`
}

// An InputEventPayload contains Sway input event data.
type InputEventPayload struct {
	Change string      `json:"change"`
	Input  InputDevice `json:"input"`
}
