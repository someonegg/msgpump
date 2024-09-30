// Copyright 2023 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpump provides a binary message-pump facility.
//
// The message-pump will continuously receive, process and send messages after startup.
//
// The message is defined as variable-length byte array.
//
// The transport layer is defined by the MessageReadWriter interface, there is an implementation over net.Conn.
package msgpump
