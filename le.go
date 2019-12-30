// TODO: Add little endian architectures to the constraint below.

// +build 386 amd64 arm arm64

package main

import "encoding/binary"

// ByteOrder defines Sway IPC byte order.
var ByteOrder = binary.LittleEndian
