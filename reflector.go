package reflector

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// ServiceInfo holds information about a gRPC service
type ServiceInfo struct {
	Name    string
	Methods []MethodInfo
}

// MethodInfo holds information about a gRPC method
type MethodInfo struct {
	Name       string
	InputType  string
	OutputType string
	InputSchema  *MessageSchema
	OutputSchema *MessageSchema
}

// MessageSchema holds the schema of a protobuf message
type MessageSchema struct {
	Name   string
	Fields []FieldInfo
}

// FieldInfo holds information about a protobuf message field
type FieldInfo struct {
	Name     string
	Number   int32
	Type     string
	Repeated bool
}

// GetServices extracts all registered services and their methods from a gRPC server
func GetServices(server *grpc.Server) ([]ServiceInfo, error) {
	var services []ServiceInfo

	// Get all registered services via reflection
	serviceInfo := server.GetServiceInfo()

	for serviceName, info := range serviceInfo {
		service := ServiceInfo{
			Name:    serviceName,
			Methods: []MethodInfo{},
		}

		// Get method information
		for _, method := range info.Methods {
			methodInfo := MethodInfo{
				Name: method.Name,
			}

			// Try to get full method descriptor to extract input/output types
			fullMethodName := fmt.Sprintf("/%s/%s", serviceName, method.Name)
			if desc, err := getMethodDescriptor(fullMethodName); err == nil {
				methodInfo.InputType = string(desc.Input().FullName())
				methodInfo.OutputType = string(desc.Output().FullName())

				// Get input schema
				if inputSchema, err := getMessageSchema(desc.Input()); err == nil {
					methodInfo.InputSchema = inputSchema
				}

				// Get output schema
				if outputSchema, err := getMessageSchema(desc.Output()); err == nil {
					methodInfo.OutputSchema = outputSchema
				}
			}

			service.Methods = append(service.Methods, methodInfo)
		}

		services = append(services, service)
	}

	return services, nil
}

// getMethodDescriptor attempts to find a method descriptor by full method name
func getMethodDescriptor(fullMethodName string) (protoreflect.MethodDescriptor, error) {
	// Parse full method name: /package.service/method
	parts := strings.Split(strings.TrimPrefix(fullMethodName, "/"), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid method name format: %s", fullMethodName)
	}

	serviceName := parts[0]

	// Try to find the service descriptor in the global registry
	var methodDesc protoreflect.MethodDescriptor
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			sd := services.Get(i)
			if string(sd.FullName()) == serviceName {
				methods := sd.Methods()
				for j := 0; j < methods.Len(); j++ {
					md := methods.Get(j)
					if string(md.Name()) == parts[1] {
						methodDesc = md
						return false // Stop iteration
					}
				}
			}
		}
		return true // Continue iteration
	})

	if methodDesc == nil {
		return nil, fmt.Errorf("method descriptor not found for %s", fullMethodName)
	}

	return methodDesc, nil
}

// getMessageSchema extracts the schema of a protobuf message
func getMessageSchema(msgDesc protoreflect.MessageDescriptor) (*MessageSchema, error) {
	if msgDesc == nil {
		return nil, fmt.Errorf("message descriptor is nil")
	}

	schema := &MessageSchema{
		Name:   string(msgDesc.FullName()),
		Fields: []FieldInfo{},
	}

	fields := msgDesc.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldInfo := FieldInfo{
			Name:     string(field.Name()),
			Number:   int32(field.Number()),
			Type:     field.Kind().String(),
			Repeated: field.Cardinality() == protoreflect.Repeated,
		}

		// For message types, use the full type name
		if field.Kind() == protoreflect.MessageKind {
			fieldInfo.Type = string(field.Message().FullName())
		}

		schema.Fields = append(schema.Fields, fieldInfo)
	}

	return schema, nil
}

// FormatServices formats service information for logging or display
func FormatServices(services []ServiceInfo) string {
	var sb strings.Builder

	for _, service := range services {
		sb.WriteString(fmt.Sprintf("  - %s\n", service.Name))
		for _, method := range service.Methods {
			sb.WriteString(fmt.Sprintf("    * %s", method.Name))
			if method.InputType != "" && method.OutputType != "" {
				sb.WriteString(fmt.Sprintf("(%s) returns (%s)", method.InputType, method.OutputType))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// FormatServicesDetailed formats service information with detailed schemas
func FormatServicesDetailed(services []ServiceInfo) string {
	var sb strings.Builder

	for _, service := range services {
		sb.WriteString(fmt.Sprintf("Service: %s\n", service.Name))
		for _, method := range service.Methods {
			sb.WriteString(fmt.Sprintf("  Method: %s\n", method.Name))
			sb.WriteString(fmt.Sprintf("    Input:  %s\n", method.InputType))
			if method.InputSchema != nil {
				for _, field := range method.InputSchema.Fields {
					repeated := ""
					if field.Repeated {
						repeated = "repeated "
					}
					sb.WriteString(fmt.Sprintf("      - %s%s %s = %d\n", repeated, field.Type, field.Name, field.Number))
				}
			}
			sb.WriteString(fmt.Sprintf("    Output: %s\n", method.OutputType))
			if method.OutputSchema != nil {
				for _, field := range method.OutputSchema.Fields {
					repeated := ""
					if field.Repeated {
						repeated = "repeated "
					}
					sb.WriteString(fmt.Sprintf("      - %s%s %s = %d\n", repeated, field.Type, field.Name, field.Number))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
