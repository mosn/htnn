// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cel

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	"mosn.io/htnn/pkg/filtermanager/api"
	"mosn.io/htnn/pkg/log"
	pkgRequest "mosn.io/htnn/pkg/request"
)

var (
	logger = log.DefaultLogger.WithName("cel")

	celEnv     *cel.Env
	initCelEnv = sync.OnceFunc(func() {
		options := []cel.EnvOption{
			cel.CustomTypeAdapter(&customTypeAdapter{}),
			defineRequest(),
		}

		var err error
		celEnv, err = cel.NewEnv(
			options...,
		)
		if err != nil {
			panic(err)
		}
	})
)

type Script struct {
	program cel.Program
}

func compile(env *cel.Env, expr string, celType *cel.Type) (*cel.Ast, error) {
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	// the celType must be built type
	if ast.OutputType() != celType {
		return nil, fmt.Errorf("got %v, wanted %v", ast.OutputType(), celType)
	}
	return ast, nil
}

func Compile(expr string, returnType *cel.Type) (*Script, error) {
	initCelEnv()
	ast, err := compile(celEnv, expr, returnType)
	if err != nil {
		return nil, err
	}
	program, _ := celEnv.Program(ast)

	s := &Script{
		program: program,
	}
	return s, nil
}

var varsPool = sync.Pool{
	New: func() any {
		return map[string]any{
			"request": &request{},
		}
	},
}

func EvalRequest(s *Script, cb api.FilterCallbackHandler, headers api.RequestHeaderMap) (any, error) {
	vars := varsPool.Get().(map[string]any)
	r := vars["request"].(*request)
	r.headers = headers
	r.callback = cb

	res, _, err := s.program.Eval(vars)
	r.headers = nil
	r.callback = nil
	varsPool.Put(vars)

	if err != nil {
		return nil, err
	}

	return res.Value(), nil
}

type request struct {
	customType
	headers  api.RequestHeaderMap
	callback api.FilterCallbackHandler
}

var requestType = cel.ObjectType("htnn.request", traits.ReceiverType)
var requestExprType = decls.NewObjectType("htnn.request")

func defineRequest() cel.EnvOption {
	cls := "request"
	declarations := []*exprpb.Decl{
		decls.NewConst(cls, requestExprType, nil),
	}

	// The methods come from https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes
	for _, dec := range []struct {
		method         string
		parameterTypes []*exprpb.Type
		returnType     *exprpb.Type
	}{
		{
			method:         "path",
			parameterTypes: []*exprpb.Type{},
			returnType:     decls.String,
		},
		{
			method:         "url_path",
			parameterTypes: []*exprpb.Type{},
			returnType:     decls.String,
		},
		{
			method:         "host",
			parameterTypes: []*exprpb.Type{},
			returnType:     decls.String,
		},
		{
			method:         "scheme",
			parameterTypes: []*exprpb.Type{},
			returnType:     decls.String,
		},
		{
			method:         "method",
			parameterTypes: []*exprpb.Type{},
			returnType:     decls.String,
		},
		{
			method:         "id",
			parameterTypes: []*exprpb.Type{},
			returnType:     decls.String,
		},
	} {
		declarations = append(declarations,
			decls.NewFunction(dec.method,
				decls.NewInstanceOverload(fmt.Sprintf("%s_%s", cls, dec.method),
					append([]*exprpb.Type{requestExprType}, dec.parameterTypes...), dec.returnType)),
		)
	}
	return cel.Declarations(declarations...)
}

func fromProperty(cb api.FilterCallbackHandler, property string) ref.Val {
	s, err := cb.GetProperty(property)
	if err != nil {
		logger.Error(err, "failed to get property", "property", property)
		return types.String("")
	}
	return types.String(s)
}

func (r *request) Receive(function string, overload string, args []ref.Val) ref.Val {
	switch function {
	case "path":
		return types.String(r.headers.Path())
	case "url_path":
		return types.String(pkgRequest.GetUrl(r.headers).Path)
	case "host":
		return types.String(r.headers.Host())
	case "scheme":
		return types.String(r.headers.Scheme())
	case "method":
		return types.String(r.headers.Method())
	case "id":
		return fromProperty(r.callback, "request.id")
	}

	return types.NewErr("no such function - %s", function)
}

func (r *request) TypeName() string {
	return requestType.TypeName()
}

type customType struct {
}

// implement ref.Val

func (t *customType) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	panic("not required")
}

func (t *customType) ConvertToType(typeVal ref.Type) ref.Val {
	panic("not required")
}

func (t *customType) Equal(other ref.Val) ref.Val {
	o, ok := other.Value().(*customType)
	if !ok {
		return types.False
	}
	return types.Bool(o == t)
}

func (t *customType) Type() ref.Type {
	return requestType
}

func (t *customType) Value() interface{} {
	return t
}

func (t *customType) HasTrait(trait int) bool {
	return trait == traits.ReceiverType
}

type customTypeAdapter struct {
}

func (customTypeAdapter) NativeToValue(value interface{}) ref.Val {
	val, ok := value.(*customType)
	if ok {
		return val
	}
	return types.DefaultTypeAdapter.NativeToValue(value)
}
