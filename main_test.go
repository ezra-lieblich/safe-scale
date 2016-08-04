package main

import (
	"github.com/cloudfoundry/cli/plugin/models"
	"github.com/cloudfoundry/cli/plugin/pluginfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"errors"
	"github.com/nicholasf/fakepoint"
)

var _ = Describe("safescale", func() {
	var (
		connection      *pluginfakes.FakeCliConnection
		ExamplePlugin   *SafeScaler
		result          *AppProp
	)
	BeforeEach(func() {
		connection = &pluginfakes.FakeCliConnection{}
		ExamplePlugin = &SafeScaler{}
	})
	Describe("get arguments", func() {
		It("should fail when insufficient the original app isn't specified", func() {
			err := ExamplePlugin.getArgs([]string{"safe-scale"})
			Expect(err.Error()).To(Equal("ERROR. Insufficient arguments. Did not specify the original app\n"))
		})
		It("should fail when the new app name isn't specified", func() {
			err := ExamplePlugin.getArgs([]string{"safe-scale", "foo-app"})
			Expect(err.Error()).To(Equal("ERROR. Insufficient arguments. Did not specify a name for new app\n"))
		})
		It("should set to default values", func() {
			err := ExamplePlugin.getArgs([]string{"safe-scale", "test-app", "new-app"})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.inst).To(Equal("1"))
			Expect(ExamplePlugin.trans).To(Equal(""))
			Expect(ExamplePlugin.test).To(Equal(""))
			Expect(ExamplePlugin.timeout).To(Equal(120))
		})
		It("should set all flags sucessfully", func() {
			args := []string{"safe-scale", "foo", "new-app", "--inst", "4", "--test", "/test", "-trans", "/trans", "--timeout", "40"}
			err := ExamplePlugin.getArgs(args)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.inst).To(Equal("4"))
			Expect(ExamplePlugin.trans).To(Equal("/trans"))
			Expect(ExamplePlugin.test).To(Equal("/test"))
			Expect(ExamplePlugin.timeout).To(Equal(40))
		})
		It("should set some flags and leave others as default", func() {
			args := []string{"safe-scale", "bar", "new-app", "--trans", "/trans", "-test", "/test"}
			err := ExamplePlugin.getArgs(args)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.inst).To(Equal("1"))
			Expect(ExamplePlugin.trans).To(Equal("/trans"))
			Expect(ExamplePlugin.test).To(Equal("/test"))
			Expect(ExamplePlugin.timeout).To(Equal(120))
		})
	})
	Describe("app properties", func() {
		BeforeEach(func() {
			ExamplePlugin.blue = &AppProp{name:"", routes: []Route{}, alive: false}
		})
		It("app  exists", func() {
			app := plugin_models.GetAppModel{Name: "blue-app"}
			connection.GetAppReturns(app, nil)
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "blue-app", "green-app"})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.blue.name).To(Equal("blue-app"))
			Expect(ExamplePlugin.green.name).To(Equal("green-app"))
		})
		It("app doesn't exist", func() {
			app := plugin_models.GetAppModel{Name: "blue-app"}
			connection.GetAppReturns(app, errors.New("The app doesn't exist"))
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "blue-app"})
			Expect(err.Error()).To(Equal("ERROR. Could not access blue-app in Cloud Foundry\n"))

		})
		It("has no routes", func() {
			app := plugin_models.GetAppModel{}
			connection.GetAppReturns(app, nil)
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "", ""}) //names are not relevant
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.blue.routes).To(Equal([]Route{}))
		})
		It("has one route", func() {
			domain_name := plugin_models.GetApp_DomainFields{Name: "cfapps.io"}
			route := []plugin_models.GetApp_RouteSummary{
				{
					Host:        "trial",
					Domain:        domain_name,
				},
			}
			app := plugin_models.GetAppModel{Routes: route}
			connection.GetAppReturns(app, nil)
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "", ""})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.blue.routes).To(Equal([]Route{{host: "trial", domain: "cfapps.io"}}))
		})
		It("has more than one route", func() {
			domain_name := plugin_models.GetApp_DomainFields{Name: "cfapps.io"}
			route := []plugin_models.GetApp_RouteSummary{
				{
					Host:        "foo",
					Domain:        domain_name,
				},
				{
					Host:        "bar",
					Domain: domain_name,
				},
			}
			app := plugin_models.GetAppModel{Routes: route}
			connection.GetAppReturns(app, nil)
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "", ""})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.blue.routes).To(Equal([]Route{
				{host: "foo", domain: "cfapps.io"}, {host: "bar", domain: "cfapps.io"}}))
		})
		It("has no services", func() {
			app := plugin_models.GetAppModel{}
			connection.GetAppReturns(app, nil)
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "", ""})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.services).To(Equal([]string{}))
			Expect(ExamplePlugin.green.name).To(Equal(""))
		})
		It("has multiple services", func() {
			services := []plugin_models.GetApp_ServiceSummary{
				{Name: "foo"},
				{Name: "bar"},
			}
			app := plugin_models.GetAppModel{Services: services}
			connection.GetAppReturns(app, nil)
			err := ExamplePlugin.getApp(connection, []string{"safe-scale", "", ""})
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.services).To(Equal([]string{"foo", "bar"}))
		})
	})

	Describe("get space", func() {
		It("should fail if it can't get space", func() {
			connection.GetCurrentSpaceReturns(plugin_models.Space{}, errors.New("there is no space"))
			err := ExamplePlugin.getSpace(connection)
			Expect(err.Error()).To(Equal("ERROR. Could not find space in Cloud Foundry\n"))
		})
		It("should get space sucessfully", func() {
			space_field := plugin_models.SpaceFields{Name: "sandbox"}
			connection.GetCurrentSpaceReturns(plugin_models.Space{space_field}, nil)
			err := ExamplePlugin.getSpace(connection)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.space).To(Equal("sandbox"))
		})
	})

	Describe("create new app", func() {
		BeforeEach(func() {
			ExamplePlugin.green = &AppProp{name:"new-app", routes: []Route{}, alive: false}
			ExamplePlugin.blue = &AppProp{routes: []Route{{domain:"cfapps.io"}}}
		})
		It("should fail to create new app if the blue app has no routes", func() {
			ExamplePlugin.blue = &AppProp{routes: []Route{}, name: "foo"}
			err := ExamplePlugin.createNewApp(connection)
			Expect(err.Error()).To(Equal("ERROR. Can't do blue green deployment because foo has no routes\n"))

		})
		It("should push a new app sucesfully", func() {
			connection.CliCommandReturns([]string{"yes"}, nil)
			err := ExamplePlugin.pushApp(connection)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.green.name).To(Equal("new-app"))
			Expect(ExamplePlugin.green.routes).To(Equal([]Route{{domain: "cfapps.io", host: "new-app"}}))
			Expect(ExamplePlugin.green.alive).To(Equal(true))
		})
		It("should fail if it can't push the app", func() {
			connection.CliCommandReturns(nil, errors.New("failed to push the app"))
			err := ExamplePlugin.pushApp(connection)
			Expect(err.Error()).To(Equal("ERROR. Unable to push new-app to Cloud Foundry\n"))
		})
		It("should bind a service sucesfully", func() {
			connection.CliCommandReturns([]string{"sucess"}, nil)
			err := ExamplePlugin.bindService(connection, "foo-db")
			Expect(err).To(BeNil())
		})
		It("should fail if it can't bind a service to an app", func() {
			connection.CliCommandReturns(nil, errors.New("failed to bind service"))
			err := ExamplePlugin.bindService(connection, "bad-service")
			Expect(err.Error()).To(Equal("ERROR. Could not bind bad-service service to new-app\n"))
		})
	})

	Describe("monitoring health", func() {
		BeforeEach(func() {
			ExamplePlugin.green = &AppProp{routes: []Route{{domain: "cfapps.io", host: "foo"}}}
			ExamplePlugin.test = "/test"
		})
		It("should return true if the app is healthy", func() {
			maker := fakepoint.NewFakepointMaker()
			maker.NewGet("https://foo.cfapps.io/test", 200)
			client := maker.Client()
			result := ExamplePlugin.healthTest(client)
			Expect(result).To(BeTrue())
		})
		It("should return false if the app is not healthy", func() {
			maker := fakepoint.NewFakepointMaker()
			maker.NewGet("https://foo.cfapps.io/test", 400)
			client := maker.Client()
			result := ExamplePlugin.healthTest(client)
			Expect(result).To(BeFalse())
		})
	})
	Describe("mapping", func() {
		BeforeEach(func() {
			app1 := &AppProp{
				name:        "green-app",
				routes: []Route{{host: "temp", domain: "cfapps.io"}},
			}
			ExamplePlugin.green = app1
		})

		It("should add a map sucessfully", func() {
			connection.CliCommandReturns([]string{"it worked"}, nil)
			moved := Route{host: "moved", domain:"cfapps.io"}
			err := ExamplePlugin.addMap(connection, ExamplePlugin.green, moved)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.green.routes).To(Equal([]Route{{host: "temp", domain: "cfapps.io"},
				{host: "moved", domain: "cfapps.io"}}))
		})
		It("should fail if it can't add map", func() {
			connection.CliCommandReturns(nil, errors.New("could not map route"))
			bad_route := Route{domain: "fake", host: "cfapps.io"}
			err := ExamplePlugin.addMap(connection, ExamplePlugin.green, bad_route)
			Expect(err.Error()).To(Equal("ERROR. Could not map fake.cfapps.io route to green-app\n"))
			Expect(ExamplePlugin.green.routes).To(Equal([]Route{{host: "temp", domain:"cfapps.io"}}))
		})

	})
	Describe("unmapping", func() {
		BeforeEach(func() {
			app := &AppProp{
				name:        "foo",
				routes: []Route{{host: "foo", domain:"cfapps.io"}, {host: "bar", domain: "cfapps.io"}},
			}
			result = app
		})
		It("should unmap a route from app sucessfully", func() {
			deleted_route := Route{host: "bar", domain: "cfapps.io"}
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err := ExamplePlugin.removeMap(connection, result, deleted_route, false)
			Expect(err).To(BeNil())
			Expect(result.routes).To(Equal([]Route{{host: "foo", domain: "cfapps.io"}}))
		})
		It("should unmap a route from app to an empty route", func() {
			app1 := &AppProp{
				name:        "foo",
				routes: []Route{{host: "bar", domain: "cfapps.io"}},
			}
			result = app1
			deleted_route := Route{host: "bar", domain: "cfapps.io"}
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err := ExamplePlugin.removeMap(connection, result, deleted_route, false)
			Expect(err).To(BeNil())
			Expect(result.routes).To(Equal([]Route{}))
		})
		It("should fail if it can't unmap a route from app sucesfully", func() {
			bad_route := Route{domain: "bad", host: "cfapps.io"} //route doesn't exist
			connection.CliCommandReturns(nil, errors.New("could not unmap route"))
			err := ExamplePlugin.removeMap(connection, result, bad_route, false)
			Expect(err.Error()).To(Equal("ERROR. Could not unmap bad.cfapps.io route from foo\n"))
			Expect(result.routes).To(Equal([]Route{{host: "foo", domain:"cfapps.io"},
				{host: "bar", domain: "cfapps.io"}}))
		})
		It("should delete an orphaned route", func() {
			orphan_route := Route{host: "bar", domain: "cfapps.io"}
			connection.CliCommandReturns([]string{"it worked"}, nil)
			err := ExamplePlugin.deleteRoute(connection, orphan_route)
			Expect(err).To(BeNil())
		})
		It("should fail if it can't delete an orphaned route", func() {
			orphan_route := Route{domain: "bad", host: "route"}
			connection.CliCommandReturns(nil, errors.New("It could not delete route"))
			err := ExamplePlugin.deleteRoute(connection, orphan_route)
			Expect(err.Error()).To(Equal("ERROR. Could not delete bad.route route from space\n"))

		})

	})
	Describe("monitoring transactions", func() {
		BeforeEach(func() {
			ExamplePlugin.blue = &AppProp{routes: []Route{{domain: "cfapps.io", host: "bar"}}, name: "foo"}
			ExamplePlugin.trans = "/trans"
			ExamplePlugin.timeout = 4
		})
		It("should fail when it times out", func() {
			maker := fakepoint.NewFakepointMaker()
			maker.NewGet("https://bar.cfapps.io/trans", 200).Duplicate(100)
			client := maker.Client()
			result := ExamplePlugin.monitorTransactions(client)
			Expect(result.Error()).To(Equal("ERROR. The request timed out. https://bar.cfapps.io/trans endpoint failed to provide HTTP Status Code 204. Can't safely shut down foo\n"))
		})
		It("should pass when transactions are empty initially", func() {
			maker := fakepoint.NewFakepointMaker()
			maker.NewGet("https://bar.cfapps.io/trans", 204).Duplicate(100)
			client := maker.Client()
			result := ExamplePlugin.monitorTransactions(client)
			Expect(result).To(BeNil())
		})
		It("should fail when get request is bad", func() {
			maker := fakepoint.NewFakepointMaker()
			maker.NewGet("https://bar.cfapps.io/trans", 404)
			client := maker.Client()
			result := ExamplePlugin.monitorTransactions(client)
			Expect(result.Error()).To(Equal("ERROR. Status code 404. https://bar.cfapps.io/trans endpoint is not okay. Check to make sure foo is healthy\n"))
		})
	})

	Describe("power down", func() {
		BeforeEach(func() {
			ExamplePlugin.blue = &AppProp{alive: true, routes: []Route{{domain: "cfapps.io", host: "foo"}}}
		})
		It("should power down sucessfully", func() {
			connection.CliCommandReturns([]string{"sucess"}, nil)
			err := ExamplePlugin.powerDown(connection)
			Expect(err).To(BeNil())
			Expect(ExamplePlugin.blue.alive).To(BeFalse())
		})
	})

})