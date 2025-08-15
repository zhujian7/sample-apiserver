package main

import (
	"context"

	"example.com/mytest-apiserver/pkg/apis/gadgets"
	"example.com/mytest-apiserver/pkg/apis/widgets"
	mycommon "example.com/mytest-apiserver/pkg/common"
	generatedopenapi "example.com/mytest-apiserver/pkg/generated/openapi"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	basecompatibility "k8s.io/component-base/compatibility"
	"k8s.io/klog/v2"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	gv := schema.GroupVersion{Group: mycommon.GroupName, Version: mycommon.APIVersion}
	Scheme.AddKnownTypes(gv, &widgets.Widget{}, &widgets.WidgetList{}, &gadgets.Gadget{}, &gadgets.GadgetList{})
	metav1.AddToGroupVersion(Scheme, gv)

	// Register meta types
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

func installAPI(s *genericapiserver.GenericAPIServer) error {
	widgetREST := widgets.NewWidgetREST()
	gadgetREST := gadgets.NewGadgetREST()

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(mycommon.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[mycommon.APIVersion] = map[string]rest.Storage{
		"widgets": widgetREST,
		"gadgets": gadgetREST,
	}

	return s.InstallAPIGroup(&apiGroupInfo)
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
}

type MyAPIServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

func (s *MyAPIServer) Run(ctx context.Context) error {
	return s.GenericAPIServer.PrepareRun().RunWithContext(ctx)
}

func NewConfig() *Config {
	return &Config{
		GenericConfig: genericapiserver.NewRecommendedConfig(Codecs),
	}
}

func (c *Config) Complete() *Config {
	c.GenericConfig.EffectiveVersion = basecompatibility.NewEffectiveVersionFromString("1.30.0", "", "")

	// Configure OpenAPI with generated definitions (includes standard types)
	defNamer := openapi.NewDefinitionNamer(Scheme)
	c.GenericConfig.OpenAPIConfig = genericapiserver.
		DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, defNamer)
	c.GenericConfig.OpenAPIV3Config = genericapiserver.
		DefaultOpenAPIV3Config(generatedopenapi.GetOpenAPIDefinitions, defNamer)

	return c
}

func (c *Config) New() (*MyAPIServer, error) {
	genericServer, err := c.GenericConfig.Complete().New("my-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &MyAPIServer{
		GenericAPIServer: genericServer,
	}

	if err := installAPI(s.GenericAPIServer); err != nil {
		return nil, err
	}

	return s, nil
}

func main() {
	klog.InitFlags(nil)

	options := genericoptions.NewRecommendedOptions("", Codecs.LegacyCodec())

	// Now disable etcd for in-memory storage after validation passes
	options.Etcd = nil

	// Disable optional features not available in all clusters
	options.Admission = nil
	options.Features = nil

	options.AddFlags(pflag.CommandLine)

	pflag.Parse()

	if errs := options.Validate(); len(errs) != 0 {
		klog.Errorf("Error validating options: %v", errs)
	}

	config := NewConfig()
	if err := options.ApplyTo(config.GenericConfig); err != nil {
		klog.Fatalf("Error applying options: %v", err)
	}

	config = config.Complete()

	server, err := config.New()
	if err != nil {
		klog.Fatalf("Error creating server: %v", err)
	}

	ctx := context.Background()
	klog.Infof("Starting my-apiserver...")
	if err := server.Run(ctx); err != nil {
		klog.Fatalf("Error running server: %v", err)
	}
}
