package main

import (
	"github.com/cloudfoundry/cli/plugin/models"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"
	."github.com/onsi/ginkgo"
	."github.com/onsi/gomega"
	"errors"
)
var _ = Describe("safescale", func() {
	var(
		connection    	*pluginfakes.FakeCliConnection
		ExamplePlugin   *SafeScaler
		result 		*AppProp

	)
	BeforeEach(func() {
		connection = &pluginfakes.FakeCliConnection{}
		ExamplePlugin = &SafeScaler{}
	})
	Describe("get arguments", func() {
		It("should fail when insufficient amount of arguments", func() {
			err:= ExamplePlugin.getArgs([]string{"safe-scale"})
			Expect(err.Error()).To(Equal("Did not specify an app"))
		})
		It("should set to default values", func() {
			err:= ExamplePlugin.getArgs([]string{"safe-scale", "test-app"})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.original_name).To(Equal("test-app"))
			Expect(ExamplePlugin.inst).To(Equal("1"))
			Expect(ExamplePlugin.trans).To(Equal(""))
			Expect(ExamplePlugin.test).To(Equal(""))
		})
		It("should set all flags sucessfully", func(){
			args := []string{"safe-scale", "foo", "--inst", "4", "--test", "/test", "-trans", "/trans"}
			err:= ExamplePlugin.getArgs(args)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.original_name).To(Equal("foo"))
			Expect(ExamplePlugin.inst).To(Equal("4"))
			Expect(ExamplePlugin.trans).To(Equal("/trans"))
			Expect(ExamplePlugin.test).To(Equal("/test"))
		})
		It("should set some flags and leave others as default", func() {
			args := []string{"safe-scale", "bar", "--trans", "/trans", "-test", "/test"}
			err:= ExamplePlugin.getArgs(args)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.original_name).To(Equal("bar"))
			Expect(ExamplePlugin.inst).To(Equal("1"))
			Expect(ExamplePlugin.trans).To(Equal("/trans"))
			Expect(ExamplePlugin.test).To(Equal("/test"))
		})
	})
	Describe("app properties", func() {
		It("app  exists", func(){
			app := plugin_models.GetAppModel{Name: "green-app"}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "green-app")
			Expect(result.name).To(Equal("green-app"))
		})
		It("app doesn't exist", func(){
			app := plugin_models.GetAppModel{Name: "blue-app"}
			connection.GetAppReturns(app, errors.New("The app doesn't exist"))
			_, err := ExamplePlugin.getApp(connection, "blue-app")
			Expect(err.Error()).To(Equal("The app doesn't exist"))

		})
		It("has no routes", func() {
			app := plugin_models.GetAppModel{}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "") //name is not relevant
			Expect(result.routes).To(Equal([]Route{}))
		})
		It("has one route", func(){
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
			Expect(result.routes).To(Equal([]Route{{host: "trial", domain: "cfapps.io"}}))
		})
		It("has more than one route", func(){
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
			Expect(result.routes).To(Equal([]Route{
				{host: "foo", domain: "cfapps.io"}, {host: "bar", domain: "cfapps.io"}}))
		})
		It("has no services", func() {
			app := plugin_models.GetAppModel{}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "")
			Expect(ExamplePlugin.services).To(Equal([]string{}))
		})
		It("has multiple services", func() {
			services:= []plugin_models.GetApp_ServiceSummary{
				{Name: "foo"},
				{Name: "bar"},
			}
			app:= plugin_models.GetAppModel{Services: services}
			connection.GetAppReturns(app, nil)
			result, _ = ExamplePlugin.getApp(connection, "")
			Expect(ExamplePlugin.services).To(Equal([]string{"foo", "bar"}))
		})
	})
	Describe("create new app", func() {
		BeforeEach(func() {
			ExamplePlugin.original_name= "test-app"
			ExamplePlugin.green = &AppProp{name:"", routes: []Route{}, alive: false}
			ExamplePlugin.blue = &AppProp{routes: []Route{{domain:"cfapps.io"}}}
		})
		It("should push a new app sucesfully", func() {
			connection.CliCommandReturns([]string{"yes"}, nil)
			err:=ExamplePlugin.pushApp(connection)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.green.name).To(Equal("new-test-app"))
			Expect(ExamplePlugin.green.routes).To(Equal([]Route{{domain: "cfapps.io", host: "new-test-app"}}))
			Expect(ExamplePlugin.green.alive).To(Equal(true))
		})
		It("should fail if it can't push the app", func() {
			connection.CliCommandReturns(nil, errors.New("failed to push the app"))
			err:= ExamplePlugin.pushApp(connection)
			Expect(err.Error()).To(Equal("failed to push the app"))
		})
		It("should bind a service sucesfully", func() {
			connection.CliCommandReturns([]string{"sucess"}, nil)
			err:= ExamplePlugin.bindService(connection, "foo-db")
			Expect(err).To(BeNil())
		})
		It("should fail if it can't bind a service to an app", func() {
			connection.CliCommandReturns(nil, errors.New("failed to bind service"))
			err:= ExamplePlugin.bindService(connection, "bad-service")
			Expect(err.Error()).To(Equal("failed to bind service"))
		})
	})
	Describe("get space", func() {
		It("should get space sucesfully", func() {
			connection.GetCurrentSpaceReturns(plugin_models.Space{}, errors.New("there is no space"))
			err := ExamplePlugin.getSpace(connection)
			Expect(err.Error()).To(Equal("there is no space"))
		})
		It("should fail if it can't get space", func() {
			space_field:= plugin_models.SpaceFields{Name: "sandbox"}
			connection.GetCurrentSpaceReturns(plugin_models.Space{space_field}, nil)
			err := ExamplePlugin.getSpace(connection)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.space).To(Equal("sandbox"))
		})
	})
	Describe("mapping", func() {
		BeforeEach(func(){
			app1 := &AppProp{
				name: 	"green-app",
				routes: []Route{{host: "temp", domain: "cfapps.io"}},
			}
			ExamplePlugin.green = app1
		})

		It("should add a map sucessfully", func(){
			connection.CliCommandReturns([]string{"it worked"},nil)
			moved := Route{host: "moved", domain:"cfapps.io"}
			err := ExamplePlugin.addMap(connection,ExamplePlugin.green, moved)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.green.routes).To(Equal([]Route{{host: "temp", domain: "cfapps.io"},
								{host: "moved", domain: "cfapps.io"}}))
		})
		It("should fail if it can't add map", func(){
			connection.CliCommandReturns(nil, errors.New("could not map route"))
			bad_route := Route{host: "fake", domain: "cfapps.io"}
			err:= ExamplePlugin.addMap(connection, ExamplePlugin.green, bad_route)
			Expect(err.Error()).To(Equal("could not map route"))
			Expect(ExamplePlugin.green.routes).To(Equal([]Route{{host: "temp", domain:"cfapps.io"}}))
		})

	})
	Describe("unmapping", func() {
		BeforeEach(func(){
			app := &AppProp{
				name:	"foo",
				routes: []Route{{host: "foo", domain:"cfapps.io"}, {host: "bar", domain: "cfapps.io"}},
			}
			result = app
		})
		It("should unmap a route from app sucessfully", func() {
			deleted_route:= Route{host: "bar", domain: "cfapps.io"}
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err:= ExamplePlugin.removeMap(connection, result, deleted_route)
			Expect(err).To(BeNil())
			Expect(result.routes).To(Equal([]Route{{host: "foo", domain: "cfapps.io"}}))
		})
		It("should unmap a route from app to an empty route", func() {
			app1 := &AppProp{
				name:	"foo",
				routes: []Route{{host: "bar", domain: "cfapps.io"}},
			}
			result = app1
			deleted_route:= Route{host: "bar", domain: "cfapps.io"}
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err:= ExamplePlugin.removeMap(connection, result, deleted_route)
			Expect(err).To(BeNil())
			Expect(result.routes).To(Equal([]Route{}))
		})
		It("should fail if it can't unmap a route from app sucesfully", func() {
			bad_route:= Route{host: "bad", domain: "cfapps.io"} //route doesn't exist
			connection.CliCommandReturns(nil, errors.New("could not unmap route"))
			err:= ExamplePlugin.removeMap(connection, result, bad_route)
			Expect(err.Error()).To(Equal("could not unmap route"))
			Expect(result.routes).To(Equal([]Route{{host: "foo", domain:"cfapps.io"},
										{host: "bar", domain: "cfapps.io"}}))
		})

	})
	Describe("power down", func() {
		BeforeEach(func() {
			ExamplePlugin.blue= &AppProp{alive: true, routes: []Route{{domain: "cfapps.io", host: "foo"}}}
		})
		It("should power down sucessfully", func() {
			connection.CliCommandReturns([]string{"sucess"}, nil)
			err:= ExamplePlugin.powerDown(connection)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.blue.alive).To(BeFalse())
		})
	})

})