// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !emulator
// +build !emulator

package tests

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/cloud-bigtable-clients-test/testproxypb"
	"github.com/stretchr/testify/assert"
	btpb "google.golang.org/genproto/googleapis/bigtable/v2"
	"google.golang.org/protobuf/testing/protocmp"
)

// Tests that a query will run successfully when receiving a response with no rows
func TestExecuteQuery_EmptyResponse(t *testing.T) {
	// 1. Instantiate the mock server
	recorder := make(chan *executeQueryReqRecord, 1)
	server := initMockServer(t)
	server.ExecuteQueryFn = mockExecuteQueryFn(recorder, &executeQueryAction{
		response: md(column("test", strType())),
	})
	// 2. Build the request to test proxy
	req := testproxypb.ExecuteQueryRequest{
		ClientId: t.Name(),
		Request: &btpb.ExecuteQueryRequest{
			InstanceName: instanceName,
			Query:        "SELECT * FROM table",
		},
	}
	// 3. Perform the operation via test proxy
	res := doExecuteQueryOp(t, server, &req, nil)
	// 4. Verify the read succeeds, gets the expected metadata, and the client sends the request properly
	checkResultOkStatus(t, res)
	assert.Equal(t, len(res.Metadata.Columns), 1)
	assert.True(t, cmp.Equal(res.Metadata, testProxyMd(column("test", strType())), protocmp.Transform()))
	assert.Equal(t, len(res.Rows), 0)

	origReq := <-recorder
	if diff := cmp.Diff(req.Request, origReq.req, protocmp.Transform(), protocmp.IgnoreEmptyMessages()); diff != "" {
		t.Errorf("diff found (-want +got):\n%s", diff)
	}
}

// Tests that a query will run successfully when receiving a rsimple response
func TestExecuteQuery_SingleSimpleRow(t *testing.T) {
	// 1. Instantiate the mock server
	server := initMockServer(t)
	server.ExecuteQueryFn = mockExecuteQueryFn(nil,
		&executeQueryAction{
			response:    md(column("test", strType())),
			endOfStream: false,
		},
		&executeQueryAction{
			response:    partialResultSet("token", strValue("foo")),
			endOfStream: true,
		})
	// 2. Build the request to test proxy
	req := testproxypb.ExecuteQueryRequest{
		ClientId: t.Name(),
		Request: &btpb.ExecuteQueryRequest{
			InstanceName: instanceName,
			Query:        "SELECT * FROM table",
		},
	}
	// 3. Perform the operation via test proxy
	res := doExecuteQueryOp(t, server, &req, nil)
	checkResultOkStatus(t, res)
	// 4. Verify the read succeeds, gets the expected metadata & data, and the client sends the request properly
	assert.Equal(t, len(res.Metadata.Columns), 1)
	assert.True(t, cmp.Equal(res.Metadata, testProxyMd(column("test", strType())), protocmp.Transform()))
	assert.Equal(t, len(res.Rows), 1)
	assert.True(t, cmp.Equal(res.Rows[0], testProxyRow(strValue("foo")), protocmp.Transform()))
}
