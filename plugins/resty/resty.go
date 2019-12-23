// Licensed to SkyAPM org under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. SkyAPM org licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package resty

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	"github.com/SkyAPM/go2sky/reporter/grpc/common"
	"github.com/go-resty/resty/v2"
)

type spanKey struct{}

const (
	httpClientComponentID int32 = 2
)

// Use attachs a skywalking middleware to the resty client.
func Use(c *resty.Client, tracer *go2sky.Tracer) {
	c.OnBeforeRequest(func(rc *resty.Client, req *resty.Request) error {
		if c == nil || tracer == nil {
			return nil
		}
		ctx := req.Context()
		urlStr := req.URL
		if !strings.HasPrefix(req.URL, "http") {
			if strings.Contains(req.URL, ":443") {
				urlStr = "https://" + req.URL
			} else {
				urlStr = "http://" + req.URL //TODO
			}
		}
		u, _ := url.Parse(urlStr)
		if u.Host == "" {
			u.Host = rc.Header.Get("Host")
		}
		if u.Host == "" {
			u.Host = rc.HostURL
		}
		span, err := tracer.CreateExitSpan(ctx, fmt.Sprintf("/%s%s", req.Method, u.Path), u.Host,
			func(header string) error {
				req.SetHeader(propagation.Header, header)
				return nil
			})
		if err != nil {
			return nil
		}
		span.SetComponent(httpClientComponentID)
		span.Tag(go2sky.TagHTTPMethod, req.Method)
		span.Tag(go2sky.TagURL, req.URL)
		span.SetSpanLayer(common.SpanLayer_Http)
		ctx = context.WithValue(ctx, spanKey{}, &span)
		req.SetContext(ctx)
		return nil
	})
	c.OnAfterResponse(func(rc *resty.Client, res *resty.Response) error {
		if c == nil || tracer == nil {
			return nil
		}
		ctx := res.Request.Context()
		span, ok := ctx.Value(spanKey{}).(*go2sky.Span)
		if !ok {
			return nil
		}
		(*span).Tag(go2sky.TagStatusCode, strconv.Itoa(res.StatusCode()))
		(*span).End()
		return nil
	})
}
