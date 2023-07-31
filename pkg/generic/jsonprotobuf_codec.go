/*
 * Copyright 2021 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package generic

import (
	"fmt"
	"context"
	"sync/atomic"

	"github.com/cloudwego/kitex/pkg/generic/descriptor"
	"github.com/cloudwego/kitex/pkg/generic/thrift"
	//"github.com/cloudwego/kitex/pkg/generic/thrift"
	//added the extran version
	"github.com/cloudwego/kitex/pkg/remote"
	"github.com/cloudwego/kitex/pkg/remote/codec"
	"github.com/cloudwego/kitex/pkg/remote/codec/perrors"
	"github.com/cloudwego/kitex/pkg/serviceinfo"


	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/cloudwego/kitex/pkg/generic/proto"
)

var (
	_ remote.PayloadCodec = &jsonProtoCodec{}
	_ Closer              = &jsonProtoCodec{}
)

// JSONRequest alias of string
//type JSONRequest = string //???

type jsonProtoCodec struct {
	svcDsc           atomic.Value // *idl
	provider         PbDescriptorProvider
	codec            remote.PayloadCodec
	binaryWithBase64 bool
}

func newJsonProtoCodec(p PbDescriptorProvider, codec remote.PayloadCodec) (*jsonProtoCodec, error) {
	svc := <-p.Provide()
	c := &jsonProtoCodec{codec: codec, provider: p, binaryWithBase64: true}
	c.svcDsc.Store(svc)
	go c.update()
	return c, nil
}
// done

func (c *jsonProtoCodec) update() {
	for {
		svc, ok := <-c.provider.Provide()
		if !ok {
			return
		}
		c.svcDsc.Store(svc)
	}
}
//

func (c *jsonProtoCodec) Marshal(ctx context.Context, msg remote.Message, out remote.ByteBuffer) error {
	method := msg.RPCInfo().Invocation().MethodName()
	if method == "" {
		return perrors.NewProtocolErrorWithMsg("empty methodName in proto Marshal")
	}
	if msg.MessageType() == remote.Exception {
		return c.codec.Marshal(ctx, msg, out)
	}
	svcDsc, ok := c.svcDsc.Load().(*desc.ServiceDescriptor)
	if !ok {
		return perrors.NewProtocolErrorWithMsg("get parser ServiceDescriptor failed")
	}

	//https://pkg.go.dev/github.com/jhump/protoreflect@v1.8.2/desc#ServiceDescriptor.FindMethodByName



	


	wm, err := thrift.NewWriteJSON(svcDsc, method, msg.RPCRole() == remote.Client)

	// wm, err :=
	// if err != nil {
	// 	return err
	// }
	// wm.SetBase64Binary(c.binaryWithBase64)
	// msg.Data().(WithCodec).SetCodec(wm)

	return c.codec.Marshal(ctx, msg, out)
}

func protoWriteJSON(pbSvc desc.ServiceDescriptor, method string, isClient bool) error {
	mt := pbSvc.FindMethodByName(method)
	if mt == nil {
		return fmt.Errorf("method not found in pb descriptor: %v", method)
	}

	pbMsg := dynamic.NewMessage(mt.GetInputType())
	err := pbMsg.Unmarshal(req.RawBody)
	if err != nil {
		return fmt.Errorf("unmarshal pb body error: %v", err)
	}
}

func (c *jsonProtoCodec) Unmarshal(ctx context.Context, msg remote.Message, in remote.ByteBuffer) error {
	if err := codec.NewDataIfNeeded(serviceinfo.GenericMethod, msg); err != nil {
		return err
	}
	svcDsc, ok := c.svcDsc.Load().(*descriptor.ServiceDescriptor)
	if !ok {
		return perrors.NewProtocolErrorWithMsg("get parser ServiceDescriptor failed")
	}
	rm := thrift.NewReadJSON(svcDsc, msg.RPCRole() == remote.Client)
	rm.SetBinaryWithBase64(c.binaryWithBase64)
	msg.Data().(WithCodec).SetCodec(rm)
	return c.codec.Unmarshal(ctx, msg, in)
}

func (c *jsonProtoCodec) getMethod(req interface{}, method string) (*Method, error) {
	fnSvc, err := c.svcDsc.Load().(*descriptor.ServiceDescriptor).LookupFunctionByMethod(method)
	if err != nil {
		return nil, err
	}
	return &Method{method, fnSvc.Oneway}, nil
}

func (c *jsonProtoCodec) Name() string {
	return "JSONThrift"
}

func (c *jsonProtoCodec) Close() error {
	return c.provider.Close()
}
