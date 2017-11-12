// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package msgpump provides a message-pump facility.
//
// The message is defined as variable-length byte array, they are
// distinguished by message-type (a string), they can be read and
// write by message-pump parallelly and continuously.
package msgpump
