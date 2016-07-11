package main

import (
	"github.com/cloudfoundry/cli/plugin/models"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"errors"
)
var _ = ginkgo.Describe("safescale", func() {
	var(
		connection    	*pluginfakes.FakeCliConnection
		ExamplePlugin   *SafeScaler
		result 		*AppProp

	)
	ginkgo.BeforeEach(func() {
		connection = &pluginfakes.FakeCliConnection{}
		ExamplePlugin = &SafeScaler{}
	})
	ginkgo.Context("app properties", func() {
		ginkgo.It("app  exists", func(){
			app := plugin_models.GetAppModel{Name: "green-app"}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "green-app")
			gomega.Expect(result.name).To(gomega.Equal("green-app"))
		})
		ginkgo.It("app doesn't exist", func(){
			app := plugin_models.GetAppModel{Name: "blue-app"}
			connection.GetAppReturns(app, errors.New("The app doesn't exist"))
			_, err := ExamplePlugin.getApp(connection, "blue-app")
			gomega.Expect(err.Error()).To(gomega.Equal("The app doesn't exist"))

		})
		ginkgo.It("has no routes", func() {
			app := plugin_models.GetAppModel{}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "") //name is not relevant
			gomega.Expect(result.routes).To(gomega.Equal([]Route{}))
		})
		ginkgo.It("has one route", func(){
			domain_name := plugin_models.GetApp_DomainFields{Name: "cfapps.io"}
			route := []plugin_models.GetApp_RouteSummary{
				{
					Host:        "trial",
					Domain:        domain_name,
				},
			}
			app := plugin_models.GetAppModel{Routes: route}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "")
			gomega.Expect(result.routes).To(gomega.Equal([]Route{{host: "trial", domain: "cfapps.io"}}))
		})
		ginkgo.It("has more than one route", func(){
			domain_name := plugin_models.GetApp_DomainFields{Name: "cfapps.io"}
			route := []plugin_models.GetApp_RouteSummary{
				{
					Host: 	"foo",
					Domain:	domain_name,
				},
				{
					Host:	"bar",
					Domain: domain_name,
				},
			}
			app := plugin_models.GetAppModel{Routes: route}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "")
			gomega.Expect(result.routes).To(gomega.Equal([]Route{
				{host: "foo", domain: "cfapps.io"}, {host: "bar", domain: "cfapps.io"}}))
		})
		ginkgo.It("has no services", func() {
			app := plugin_models.GetAppModel{}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "")
			gomega.Expect(result.services).To(gomega.Equal([]string{}))
		})
		ginkgo.It("has multiple services", func() {
			services:= []plugin_models.GetApp_ServiceSummary{
				{Name: "foo"},
				{Name: "bar"},
			}
			app:= plugin_models.GetAppModel{Services: services}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "")
			gomega.Expect(result.services).To(gomega.Equal([]string{"foo", "bar"}))
		})
	})
	ginkgo.Context("get space", func() {
		ginkgo.It("unsucessfully", func() {
			connection.GetCurrentSpaceReturns(plugin_models.Space{}, errors.New("there is no space"))
			err := ExamplePlugin.getSpace(connection)
			gomega.Expect(err.Error()).To(gomega.Equal("there is no space"))
		})
		ginkgo.It("sucessfully", func() {
			space_field:= plugin_models.SpaceFields{Name: "sandbox"}
			connection.GetCurrentSpaceReturns(plugin_models.Space{space_field}, nil)
			err := ExamplePlugin.getSpace(connection)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(ExamplePlugin.space).To(gomega.Equal("sandbox"))
		})
	})
	ginkgo.Context("add mappings", func() {
		ginkgo.BeforeEach(func(){
			app1 := &AppProp{
				name: 	"green-app",
				routes: []Route{{host: "temp", domain: "cfapps.io"}},
			}
			ExamplePlugin.green = app1
		})

		ginkgo.It("sucessfully", func(){
			connection.CliCommand("map-route", ExamplePlugin.green.name,
								"cfapps.io","--hostname", "moved")
			connection.CliCommandReturns([]string{"it worked"},nil)
			moved := Route{host: "moved", domain:"cfapps.io"}
			err := ExamplePlugin.addMap(connection,ExamplePlugin.green, moved)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(ExamplePlugin.green.routes).To(gomega.Equal([]Route{{host: "temp", domain: "cfapps.io"},
								{host: "moved", domain: "cfapps.io"}}))
		})
		ginkgo.It("unsucessfully", func(){
			connection.CliCommand("map-route", ExamplePlugin.green.name,
				"cfapps.io","--hostname", "fake") //CL args have wrong host name
			connection.CliCommandReturns(nil,errors.New("could not map route"))
			bad_route := Route{host: "fake", domain: "cfapps.io"}
			err:= ExamplePlugin.addMap(connection, ExamplePlugin.green, bad_route)
			gomega.Expect(err.Error()).To(gomega.Equal("could not map route"))
			gomega.Expect(ExamplePlugin.green.routes).To(gomega.Equal([]Route{{host: "temp", domain:"cfapps.io"}}))
		})

	})
	ginkgo.Context("removes mapping", func() {
		ginkgo.BeforeEach(func(){
			app := &AppProp{
				name:	"foo",
				routes: []Route{{host: "foo", domain:"cfapps.io"}, {host: "bar", domain: "cfapps.io"}},
			}
			result = app
		})
		ginkgo.It("sucessfully", func() {
			deleted_route:= Route{host: "bar", domain: "cfapps.io"}
			connection.CliCommand("unmap-route", result.name,
				deleted_route.domain,"--hostname", deleted_route.domain)
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err:= ExamplePlugin.removeMap(connection, result, deleted_route)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(result.routes).To(gomega.Equal([]Route{{host: "foo", domain: "cfapps.io"}}))
		})
		ginkgo.It("unsucessfuly", func() {
			bad_route:= Route{host: "bad", domain: "cfapps.io"} //route doesn't exist
			connection.CliCommand("unmap-route", result.name,
				bad_route.domain,"--hostname", bad_route.domain)
			connection.CliCommandReturns(nil, errors.New("could not unmap route"))
			err:= ExamplePlugin.removeMap(connection, result, bad_route)
			gomega.Expect(err.Error()).To(gomega.Equal("could not unmap route"))
			gomega.Expect(result.routes).To(gomega.Equal([]Route{{host: "foo", domain:"cfapps.io"},
										{host: "bar", domain: "cfapps.io"}}))
		})

	})
	ginkgo.Context("renames an app", func(){
		ginkgo.BeforeEach(func(){
			app := &AppProp{name: 	"foo"}
			result = app
		})
		ginkgo.It("sucessfully", func() {
			connection.CliCommand("rename", result.name, "bar")
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err:= ExamplePlugin.renameApp(connection, result, "bar")
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(result.name).To(gomega.Equal("bar"))
		})
		ginkgo.It("unsucessfully", func() {
			connection.CliCommand("rename", "foo", "duplicate") //rename to app that already exists
			connection.CliCommandReturns(nil, errors.New("could not rename the app"))
			err:= ExamplePlugin.renameApp(connection, result, "bar")
			gomega.Expect(err.Error()).To(gomega.Equal("could not rename the app"))
			gomega.Expect(result.name).To(gomega.Equal("foo"))
		})
	})
})



