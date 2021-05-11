package plugin

import (
	"github.com/einride/ctl/internal/codegen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ServiceGenerator struct {
	service protoreflect.ServiceDescriptor
}

func (s ServiceGenerator) GenerateCmd(f *codegen.File) {
	f.Pf("var %s = &cobra.Command{", serviceCmdVarName(s.service))
	f.Pfq("Use: %s,", serviceCmdName(s.service))
	f.P("}")
	f.P()
	for j := 0; j < s.service.Methods().Len(); j++ {
		method := s.service.Methods().Get(j)
		MethodGenerator{service: s.service, method: method}.GenerateCmd(f)
	}
}

func (s ServiceGenerator) GenerateInit(f *codegen.File) {
	f.Pf("rootCmd.AddCommand(%s)", serviceCmdVarName(s.service))
	for j := 0; j < s.service.Methods().Len(); j++ {
		method := s.service.Methods().Get(j)
		MethodGenerator{service: s.service, method: method}.GenerateInit(f)
	}
}
