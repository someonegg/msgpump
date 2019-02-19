// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpump provides a message-pump facility.
//
//   Most code should use the inside package msgpeer.
//
// The message-pump will continuously receive, process and send messages after startup.
//
// The message is defined as variable-length byte array, they are distinguished
// by the message-type (a string).
//
// The transport layer is defined by the MessageReadWriter interface, there are
// two default implementations:
//   NetconnMRW over net.Conn
//   WebsocketMRW over websocket.Conn
package msgpump
