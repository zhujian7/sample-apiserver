package main

import (
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/klog/v2"

	"example.com/mytest-apiserver/pkg/apis/gadgets"
	"example.com/mytest-apiserver/pkg/apis/widgets"
	"example.com/mytest-apiserver/pkg/common"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	gv := schema.GroupVersion{Group: common.GroupName, Version: common.APIVersion}
	Scheme.AddKnownTypes(gv, &widgets.Widget{}, &widgets.WidgetList{}, &gadgets.Gadget{}, &gadgets.GadgetList{})
	metav1.AddToGroupVersion(Scheme, gv)

	// Register meta types
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

func installAPI(s *genericapiserver.GenericAPIServer) error {
	widgetREST := widgets.NewWidgetREST()
	gadgetREST := gadgets.NewGadgetREST()

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(common.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[common.APIVersion] = map[string]rest.Storage{
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

func (s *MyAPIServer) Run(stopCh <-chan struct{}) error {
	return s.GenericAPIServer.PrepareRun().Run(stopCh)
}

func NewConfig() *Config {
	return &Config{
		GenericConfig: genericapiserver.NewRecommendedConfig(Codecs),
	}
}

func (c *Config) Complete() *Config {
	c.GenericConfig.Complete()
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

	stopCh := genericapiserver.SetupSignalHandler()

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

	klog.Infof("Starting my-apiserver...")
	if err := server.Run(stopCh); err != nil {
		klog.Fatalf("Error running server: %v", err)
	}
}
