/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"context"
	"reflect"
	"sync"
)

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc/connectivity"
)

import (
	hessian2 "github.com/apache/dubbo-go-hessian2"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

var ErrNoReply = errors.New("request need @response")

type GrpcInvoker struct {
	protocol.BaseInvoker
	quitOnce sync.Once
	client   *Client
}

func NewGrpcInvoker(url common.URL, client *Client) *GrpcInvoker {
	return &GrpcInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
	}
}

func (gi *GrpcInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	var (
		result protocol.RPCResult
	)

	if invocation.Reply() == nil {
		result.Err = ErrNoReply
	}

	in := []reflect.Value{}
	in = append(in, reflect.ValueOf(context.Background()))
	in = append(in, invocation.ParameterValues()...)

	methodName := invocation.MethodName()
	method := gi.client.invoker.MethodByName(methodName)
	res := method.Call(in)

	result.Rest = res[0]
	// check err
	if !res[1].IsNil() {
		result.Err = res[1].Interface().(error)
	} else {
		_ = hessian2.ReflectResponse(res[0], invocation.Reply())
	}

	return &result
}

func (gi *GrpcInvoker) IsAvailable() bool {
	return gi.BaseInvoker.IsAvailable() && gi.client.GetState() != connectivity.Shutdown
}

func (gi *GrpcInvoker) IsDestroyed() bool {
	return gi.BaseInvoker.IsDestroyed() && gi.client.GetState() == connectivity.Shutdown
}

func (gi *GrpcInvoker) Destroy() {
	gi.quitOnce.Do(func() {
		gi.BaseInvoker.Destroy()

		if gi.client != nil {
			_ = gi.client.Close()
		}
	})
}
