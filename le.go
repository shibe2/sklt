// TODO: Add little endian architectures to the constraint below.

// +build amd64

package main

import "encoding/binary"

// ByteOrder defines Sway IPC byte order.
var ByteOrder = binary.LittleEndian
