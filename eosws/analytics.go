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

package eosws

import (
	"context"
	"fmt"

	"github.com/dfuse-io/dauth"
	"github.com/dfuse-io/logging"
	"go.uber.org/zap"
)

// TrackUserEvent tracks a event `name` with a set of key/value pairs for a
// particular user.
//
// **Important** If your read this from a call site **DO NOT** modify
// call site arguments unless you correctly ensures BigQuery analytics view
// are ready to handle the changes. If you are not sure of what you are doing,
// ask someone from the devops team.
func TrackUserEvent(ctx context.Context, name string, keyvals ...interface{}) {
	keyValueCount := len(keyvals)
	zlogger := logging.Logger(ctx, zlog)

	var fields []zap.Field
	if keyValueCount > 0 {
		if keyValueCount%2 != 0 {
			zlogger.Error("keyvals parameters should be a multiple of 2")
		}

		fields = make([]zap.Field, keyValueCount/2)
		for i := 0; i < keyValueCount; i += 2 {
			key := toString(keyvals[i])
			value := keyvals[i+1]

			fields[(i+1)/2] = zap.Any(key, value)
		}
	}

	creds := dauth.GetCredentials(ctx)
	fields = append(fields, creds.GetLogFields()...)
	zlogger.Info(name, fields...)
}

func toString(input interface{}) string {
	switch v := input.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
