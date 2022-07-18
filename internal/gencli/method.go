package gencli

import (
	"reflect"
	"strconv"

	"go.einride.tech/aip/fieldbehavior"
	"go.einride.tech/aip/reflect/aipreflect"
	"go.einride.tech/protoc-gen-go-cli/protoflag"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type newMethodCommandCodeGenerator struct {
	gen     *protogen.Plugin
	files   *protoregistry.Files
	file    *protogen.File
	service *protogen.Service
	method  *protogen.Method
}

func (c newMethodCommandCodeGenerator) goName() string {
	return "new" + c.service.GoName + c.method.GoName + "Command"
}

func (c newMethodCommandCodeGenerator) generateCode(g *protogen.GeneratedFile) error {
	isStreamingClient := c.method.Desc.IsStreamingClient()
	isStreamingServer := c.method.Desc.IsStreamingServer()

	cobraCommand := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: "github.com/spf13/cobra",
		GoName:       "Command",
	})
	cliDial := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
		GoName:       "Dial",
	})
	newClient := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: c.file.GoImportPath,
		GoName:       "New" + c.service.GoName + "Client",
	})
	osReadFile := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: "os",
		GoName:       "ReadFile",
	})
	protojsonUnmarshal := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: "google.golang.org/protobuf/encoding/protojson",
		GoName:       "Unmarshal",
	})
	g.P()
	g.P("func ", c.goName(), "() *", cobraCommand, " {")
	g.P("cmd := &", cobraCommand, "{")
	g.P("Use: ", strconv.Quote(getMethodCommandUse(c.method)), ",")
	g.P("Short: ", strconv.Quote(getShortMethod(c.method)), ",")
	g.P("}")
	g.P("var fromFile string")
	g.P(`cmd.Flags().StringVarP(&fromFile, "from-file", "f", "", "path to a JSON file containing request payload")`)
	g.P(`_ = cmd.MarkFlagFilename("from-file", "json")`)
	for _, field := range c.method.Input.Fields {
		if err := c.generateFieldFlag(g, field, nil); err != nil {
			return err
		}
	}
	g.P("cmd.RunE = func(cmd *", cobraCommand, ", args []string) error {")
	g.P("var request ", c.method.Input.GoIdent)
	g.P(`if cmd.Flags().Changed("from-file") {`)
	g.P("data, err := ", osReadFile, "(fromFile)")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("if err := ", protojsonUnmarshal, "(data, &request); err != nil {")
	g.P("return err")
	g.P("}")
	g.P("}")
	for _, field := range c.method.Input.Fields {
		if err := c.generateUnmarshalFieldFlag(g, field, nil); err != nil {
			return err
		}
	}
	g.P("conn, err := ", cliDial, "(cmd.Context())")
	g.P("if err != nil {")
	g.P("return err")
	g.P("}")
	g.P("client := ", newClient, "(conn)")
	logRequest := protogen.GoIdent{
		GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
		GoName:       "LogRequest",
	}
	g.P(logRequest, "(cmd.Context(), &request)")
	if !isStreamingClient {
		if !isStreamingServer {
			g.P("response, err := client.", c.method.GoName, "(cmd.Context(), &request)")
			g.P("if err != nil {")
			logError := protogen.GoIdent{
				GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
				GoName:       "LogError",
			}
			osExit := protogen.GoIdent{
				GoImportPath: "os",
				GoName:       "Exit",
			}
			g.P(logError, "(cmd.Context(), err)")
			g.P(osExit, "(1)")
			g.P("}")
			logResponse := protogen.GoIdent{
				GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
				GoName:       "LogResponse",
			}
			g.P(logResponse, "(cmd.Context(), response)")
			g.P("return nil")

		} else {
			g.P("streamingClient, err := client.", c.method.GoName, "(cmd.Context(), &request)")
			g.P("if err != nil {")
			logError := protogen.GoIdent{
				GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
				GoName:       "LogError",
			}
			osExit := protogen.GoIdent{
				GoImportPath: "os",
				GoName:       "Exit",
			}
			g.P(logError, "(cmd.Context(), err)")
			g.P(osExit, "(1)")
			g.P("}")
			g.P("for response, err := streamingClient.Recv();; {")
			g.P("if err != nil {")
			eof := protogen.GoIdent{
				GoImportPath: "io",
				GoName:       "EOF",
			}
			g.P("if err == ", eof, " {")
			g.P("return nil")
			g.P("}")
			g.P(logError, "(cmd.Context(), err)")
			g.P(osExit, "(1)")
			g.P("}")
			logResponse := protogen.GoIdent{
				GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
				GoName:       "LogResponse",
			}
			g.P(logResponse, "(cmd.Context(), response)")
			g.P("}")
		}
	} else {
		// TODO not supported yet
		g.P("// Client streaming is not supported yet")
		g.P("return nil")
	}
	g.P("}")
	g.P("return cmd")
	g.P("}")
	return nil
}

func (c newMethodCommandCodeGenerator) generateFieldFlag(
	g *protogen.GeneratedFile,
	field *protogen.Field,
	parents []*protogen.Field,
) error {
	if fieldbehavior.Has(field.Desc, annotations.FieldBehavior_OUTPUT_ONLY) {
		return nil
	}
	switch field.Desc.Kind() {
	case protoreflect.BytesKind:
		return c.generateFlag(g, field, parents, &protoflag.Bytes{})
	case protoreflect.StringKind:
		if field.Desc.IsList() {
			return c.generateFlag(g, field, parents, &protoflag.StringList{})
		}
		return c.generateFlag(g, field, parents, &protoflag.String{})
	case protoreflect.FloatKind:
		if field.Desc.IsList() {
			return c.generateFlag(g, field, parents, &protoflag.FloatList{})
		}
		return c.generateFlag(g, field, parents, &protoflag.Float{})
	case protoreflect.DoubleKind:
		if field.Desc.IsList() {
			return c.generateFlag(g, field, parents, &protoflag.DoubleList{})
		}
		return c.generateFlag(g, field, parents, &protoflag.Double{})
	case protoreflect.Int64Kind:
		if field.Desc.IsList() {
			return c.generateFlag(g, field, parents, &protoflag.Int64List{})
		}
		return c.generateFlag(g, field, parents, &protoflag.Int64{})
	case protoreflect.Int32Kind:
		if field.Desc.IsList() {
			return c.generateFlag(g, field, parents, &protoflag.Int32List{})
		}
		return c.generateFlag(g, field, parents, &protoflag.Int32{})
	case protoreflect.BoolKind:
		if field.Desc.IsList() {
			return c.generateFlag(g, field, parents, &protoflag.BoolList{})
		}
		return c.generateFlag(g, field, parents, &protoflag.Bool{})
	case protoreflect.EnumKind:
		if field.Desc.IsList() {
			return nil // TODO: Implement support for enum lists.
		}
		return c.generateEnumFlag(g, field, parents)
	case protoreflect.MessageKind:
		if field.Desc.IsMap() {
			switch {
			case field.Desc.MapKey().Kind() == protoreflect.StringKind &&
				field.Desc.MapValue().Kind() == protoreflect.StringKind:
				return c.generateFlag(g, field, parents, &protoflag.StringStringMap{})
			default:
				// TODO: Support more maps types.
				return nil
			}
		}
		switch field.Desc.Message().FullName() {
		case "google.protobuf.Timestamp":
			if field.Desc.IsList() {
				return nil // TODO: Support repeated timestamps.
			}
			return c.generateFlag(g, field, parents, &protoflag.Timestamp{})
		case "google.protobuf.Duration":
			if field.Desc.IsList() {
				return nil // TODO: Support repeated durations.
			}
			return c.generateFlag(g, field, parents, &protoflag.Duration{})
		default:
			if field.Desc.IsList() {
				return nil // use --from-json for message lists
			}
			for _, messageField := range field.Message.Fields {
				if err := c.generateFieldFlag(g, messageField, append(parents, field)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c newMethodCommandCodeGenerator) generateUnmarshalFieldFlag(
	g *protogen.GeneratedFile,
	field *protogen.Field,
	parents []*protogen.Field,
) error {
	if fieldbehavior.Has(field.Desc, annotations.FieldBehavior_OUTPUT_ONLY) {
		return nil
	}
	switch field.Desc.Kind() {
	case protoreflect.BytesKind,
		protoreflect.StringKind,
		protoreflect.DoubleKind,
		protoreflect.Int64Kind,
		protoreflect.Int32Kind,
		protoreflect.BoolKind:
		c.setRequestFieldIfChanged(g, field, parents)
	case protoreflect.EnumKind:
		if field.Desc.IsList() {
			return nil // TODO: Support enum lists.
		}
		c.setRequestFieldIfChanged(g, field, parents)
	case protoreflect.MessageKind:
		if field.Desc.IsMap() {
			switch {
			case field.Desc.MapKey().Kind() == protoreflect.StringKind &&
				field.Desc.MapValue().Kind() == protoreflect.StringKind:
				c.setRequestFieldIfChanged(g, field, parents)
				return nil
			default:
				// TODO: Support more maps types.
				return nil
			}
		}
		switch field.Desc.Message().FullName() {
		case "google.protobuf.Timestamp", "google.protobuf.Duration":
			c.setRequestFieldIfChanged(g, field, parents)
			return nil
		default:
			if field.Desc.IsList() {
				return nil // use --from-json for message lists
			}
			for _, messageField := range field.Message.Fields {
				if err := c.generateUnmarshalFieldFlag(g, messageField, append(parents, field)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c newMethodCommandCodeGenerator) generateFlag(
	g *protogen.GeneratedFile,
	field *protogen.Field,
	parents []*protogen.Field,
	value protoflag.Value,
) error {
	reflectValue := reflect.ValueOf(value)
	if reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	t := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: protogen.GoImportPath(reflectValue.Type().PkgPath()),
		GoName:       reflectValue.Type().Name(),
	})
	g.P("var ", getFlagVariableName(field, parents), " ", t)
	g.P("cmd.Flags().Var(")
	g.P("&", getFlagVariableName(field, parents), ",")
	g.P(strconv.Quote(getFlagName(field, parents)), ",")
	g.P(strconv.Quote(getFlagDescription(field)), ",")
	g.P(")")
	if resourceReference := proto.GetExtension(
		field.Desc.Options(), annotations.E_ResourceReference,
	).(*annotations.ResourceReference); resourceReference != nil {
		aipreflect.RangeResourceDescriptorsInPackage(
			c.files,
			c.file.Desc.Package(),
			func(resource *annotations.ResourceDescriptor) bool {
				if resource.GetType() == resourceReference.GetType() && len(resource.GetPattern()) > 0 {
					resourceNameCompletionFunc := g.QualifiedGoIdent(protogen.GoIdent{
						GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
						GoName:       "ResourceNameCompletionFunc",
					})
					g.P(`_ = cmd.RegisterFlagCompletionFunc(`)
					g.P(strconv.Quote(getFlagName(field, parents)), ",")
					g.P(resourceNameCompletionFunc, "(")
					for _, pattern := range resource.GetPattern() {
						g.P(strconv.Quote(pattern), ",")
					}
					g.P("),")
					g.P(")")
					return false
				}
				return true
			},
		)
	} else if field.Desc.Kind() == protoreflect.StringKind &&
		field.Desc.Name() == "name" &&
		field.Desc.Cardinality() != protoreflect.Repeated {
		if resource := proto.GetExtension(
			field.Parent.Desc.Options(), annotations.E_Resource,
		).(*annotations.ResourceDescriptor); resource != nil && len(resource.GetPattern()) > 0 {
			resourceNameCompletionFunc := g.QualifiedGoIdent(protogen.GoIdent{
				GoImportPath: "go.einride.tech/protoc-gen-go-cli/cli",
				GoName:       "ResourceNameCompletionFunc",
			})
			g.P(`_ = cmd.RegisterFlagCompletionFunc(`)
			g.P(strconv.Quote(getFlagName(field, parents)), ",")
			g.P(resourceNameCompletionFunc, "(")
			for _, pattern := range resource.GetPattern() {
				g.P(strconv.Quote(pattern), ",")
			}
			g.P("),")
			g.P(")")
		}
	}
	return nil
}

func (c newMethodCommandCodeGenerator) generateEnumFlag(
	g *protogen.GeneratedFile,
	field *protogen.Field,
	parents []*protogen.Field,
) error {
	enum := g.QualifiedGoIdent(protogen.GoIdent{
		GoImportPath: "go.einride.tech/protoc-gen-go-cli/protoflag",
		GoName:       "Enum",
	})
	g.P(getFlagVariableName(field, parents), " := ", enum, "{")
	g.P("Value: ", field.Enum.GoIdent, "(0),")
	g.P("}")
	g.P("cmd.Flags().Var(")
	g.P("&", getFlagVariableName(field, parents), ",")
	g.P(strconv.Quote(getFlagName(field, parents)), ",")
	g.P(strconv.Quote(getFlagDescription(field)), ",")
	g.P(")")
	return nil
}

func (c newMethodCommandCodeGenerator) setRequestFieldIfChanged(
	g *protogen.GeneratedFile,
	field *protogen.Field,
	parents []*protogen.Field,
) {
	flag := strconv.Quote(getFlagName(field, parents))
	g.P("if cmd.Flags().Changed(", flag, ") {")
	g.P("r := request.ProtoReflect()")
	if len(parents) > 0 {
		for _, parent := range parents {
			g.P("r = r.Mutable(r.Descriptor().Fields().ByName(", strconv.Quote(string(parent.Desc.Name())), ")).Message()")
		}
	}
	g.P(
		"r.Set(r.Descriptor().Fields().ByName(",
		strconv.Quote(string(field.Desc.Name())),
		"), ",
		getFlagVariableName(field, parents),
		".ProtoReflectValue())",
	)
	g.P("}")
}
