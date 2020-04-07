// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wsmsg

import (
	"fmt"
	"reflect"
)

type IncomingMessage interface {
	// GetID returns a composite of `GetType` @ `GetReqID`
	GetID() string
	GetType() string
	GetReqID() string
	GetCommon() *CommonIn
	GetWithProgress() int64
}

type CommonIn struct {
	Type             string `json:"type"`
	ReqID            string `json:"req_id,omitempty"`
	Fetch            bool   `json:"fetch,omitempty"`
	Listen           bool   `json:"listen,omitempty"`
	StartBlock       int64  `json:"start_block,omitempty"`
	IrreversibleOnly bool   `json:"irreversible_only,omitempty"`
	WithProgress     int64  `json:"with_progress,omitempty"` // send progress each X blocks
}

func (c CommonIn) GetID() string {
	reqID := c.GetReqID()
	if reqID == "" {
		reqID = "(empty)"
	}

	return c.GetType() + "@" + reqID
}

func (c CommonIn) GetType() string        { return c.Type }
func (c CommonIn) GetReqID() string       { return c.ReqID }
func (c CommonIn) GetCommon() *CommonIn   { return &c }
func (c CommonIn) GetWithProgress() int64 { return c.WithProgress }

type CommonOut struct {
	Type  string `json:"type"`
	ReqID string `json:"req_id,omitempty"`
}

func (c *CommonOut) SetType(v string)  { c.Type = v }
func (c *CommonOut) SetReqID(v string) { c.ReqID = v }

// GetType retrieves the message `type` on a Common outgoing structure.
func GetType(msg OutgoingMessager) (string, error) {
	objType := reflect.TypeOf(msg).Elem()
	typeName := OutgoingStructMap[objType]
	if typeName == "" {
		return "", fmt.Errorf("unable to determine message type for msg: %s", msg)
	}

	return typeName, nil
}

// SetType sets the `type` on a Common outgoing structure.
func SetType(msg OutgoingMessager) error {
	objType := reflect.TypeOf(msg).Elem()
	typeName := OutgoingStructMap[objType]
	if typeName == "" {
		return fmt.Errorf("unable to determine message type for msg: %s", msg)
	}
	msg.SetType(typeName)
	return nil
}

// Message registration
var IncomingMessageMap = map[string]reflect.Type{}
var IncomingStructMap = map[reflect.Type]string{}
var OutgoingMessageMap = map[string]reflect.Type{}
var OutgoingStructMap = map[reflect.Type]string{}

func RegisterIncomingMessage(typeName string, obj interface{}) {
	refType := reflect.TypeOf(obj)
	IncomingMessageMap[typeName] = refType
	IncomingStructMap[refType] = typeName
}

func RegisterOutgoingMessage(typeName string, obj interface{}) {
	refType := reflect.TypeOf(obj)
	OutgoingMessageMap[typeName] = refType
	OutgoingStructMap[refType] = typeName
}

// Obj
type M map[string]interface{}
