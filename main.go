package main

import (
	"github.com/cloudfoundry/cli/plugin"
	"fmt"
	"net/http"
	"time"
	"flag"
	"errors"
	"strconv"
)

type SafeScaler struct {
	blue         *AppProp
	green        *AppProp
	green_routes []Route
	blue_routes  []Route
	services     []string
	trans        string
	test         string
	inst         string
	timeout      int
	space        string
	client       *http.Client
}
type AppProp struct {
	name   string
	routes []Route
	alive  bool
}
type Route struct {
	host   string
	domain string
}

func (c *SafeScaler) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] != "safe-scale" {
		return
	}
	if err := c.getArgs(args); err != nil {
		fmt.Println(err)
		return
	}
	if err := c.getApp(cliConnection, args); err != nil {
		fmt.Println(err)
		return
	}
	if err := c.createNewApp(cliConnection); err != nil {
		fmt.Println(err)
		return
	}
	c.client = http.DefaultClient //client for endpoint monitoring
	if healthy := c.healthTest(c.client); !healthy {
		fmt.Println("ERROR. new app is not healthy. Can not continue blue-green deployment. Routes from old app will not be transferred to new app\n")
		return
	}
	if err := c.mapping(cliConnection); err != nil {
		fmt.Println(err)
		return
	}
	if err := c.unmapping(cliConnection); err != nil {
		fmt.Println(err)
		return
	}
	if err := c.monitorTransactions(c.client); err != nil {
		fmt.Println(err)
		return
	}
	if err := c.powerDown(cliConnection); err != nil {
		fmt.Println(err)
		return
	}

}

func (c *SafeScaler) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "safe_scale",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name: "safe-scale",
				HelpText: "Safely scales down your application using blue green deployment",
				UsageDetails: plugin.Usage{
					Usage: "safe-scale\n	cf safe-scale app_name new_app_name [--inst] [--trans] [--test] [--timeout]",
					Options: map[string]string{
						"-inst":        "number of instances for new app",
						"-trans":        "endpoint to monitor transaction",
						"-test":        "endpoint to test if new app is healthy",
						"-timeout":        "time in seconds to monitor transactions",
					},
				},
			},
		},
	}
}

func (c *SafeScaler) getArgs(args []string) error {
	if len(args) == 1 {
		return errors.New("ERROR. Insufficient arguments. Did not specify the original app\n")
	}
	if len(args) == 2 {
		return errors.New("ERROR. Insufficient arguments. Did not specify a name for new app\n")
	}
	//creating flags and setting their default values
	f := flag.NewFlagSet("f", flag.ContinueOnError)
	inst_ptr := f.String("inst", "1", "the number of instances for new app")
	trans_ptr := f.String("trans", "", "endpoint path to monitor transactions")
	test_ptr := f.String("test", "", "endpoint path to test new app deployed")
	timeout_ptr := f.Int("timeout", 120, "time in seconds before transaction monitoring times out")
	//Do not want to parse through the command name and app name. Just focused on flags
	f.Parse(args[3:])
	c.inst = *inst_ptr
	c.test = *test_ptr
	c.trans = *trans_ptr
	c.timeout = *timeout_ptr
	return nil
}

func (c *SafeScaler) getApp(cliConnection plugin.CliConnection, args []string) error {
	//getting app properties
	app, err := cliConnection.GetApp(args[1])
	c.services = []string{}
	if err != nil {
		return errors.New("ERROR. Could not access " + args[1] + " in Cloud Foundry\n")
	}
	properties := &AppProp{
		name:        "",
		routes:        []Route{},
		alive:        true,
	}
	properties.name = app.Name
	//getting routes from app
	for _, value := range app.Routes {
		new_route := Route{
			domain:        value.Domain.Name,
			host:        value.Host,
		}
		properties.routes = append(properties.routes, new_route)
	}
	//getting services from app
	for _, value := range app.Services {
		c.services = append(c.services, value.Name)
	}
	if err = c.getSpace(cliConnection); err != nil {
		return err
	}
	c.blue = properties
	c.blue_routes = c.blue.routes
	c.green = &AppProp{name:args[2], routes: []Route{}, alive: false}
	return nil
}

func (c *SafeScaler) getSpace(cliConnection plugin.CliConnection) error {
	space, err := cliConnection.GetCurrentSpace()
	if err != nil {
		return errors.New("ERROR. Could not find space in Cloud Foundry\n")
	}
	c.space = space.Name
	return nil
}

func (c *SafeScaler) createNewApp(cliConnection plugin.CliConnection) error {
	if len(c.blue.routes) == 0 {
		return errors.New("ERROR. Can't do blue green deployment because " + c.blue.name + " has no routes\n")
	}
	if err := c.pushApp(cliConnection); err != nil {
		return err
	}
	//need to bind all the services of blue app to the green app
	for _, val := range c.services {
		if err := c.bindService(cliConnection, val); err != nil {
			return err
		}
	}
	return nil
}

func (c *SafeScaler) pushApp(cliConnection plugin.CliConnection) error {
	domain := c.blue.routes[0].domain
	if _, err := cliConnection.CliCommand("push", c.green.name, "-i", c.inst, "--hostname", c.green.name, "-d", domain); err != nil {
		return errors.New("ERROR. Unable to push " + c.green.name + " to Cloud Foundry\n")
	}
	c.green.routes = append(c.green.routes, Route{host: c.green.name, domain: domain})
	c.green.alive = true
	return nil

}

func (c *SafeScaler) bindService(cliConnection plugin.CliConnection, val string) error {
	if _, err := cliConnection.CliCommand("bind-service", c.green.name, val); err != nil {
		return errors.New("ERROR. Could not bind " + val + " service to " + c.green.name + "\n")
	}
	return nil
}

func (c *SafeScaler) healthTest(client *http.Client) bool {
	//no endpoint so just continue with deployment
	if c.test == "" {
		return true
	}
	fmt.Println("Testing the health of the new app")
	endpoint := "https://" + c.green.routes[0].host + "." + c.green.routes[0].domain + c.test
	result, err := client.Get(endpoint) //test endpoint
	//not ok or error so test failed 300 multiple things going on
	if result.StatusCode != 200 || err != nil {
		return false
	}
	return true
}

func (c *SafeScaler) mapping(cliConnection plugin.CliConnection) error {
	//creates a temp route for old app
	temp_route, err := c.createRoute(cliConnection)
	if err != nil {
		return err
	}
	//add temp route to old app
	if err = c.addMap(cliConnection, c.blue, temp_route); err != nil {
		return err
	}
	//add all routes from old app to new app
	for _, val := range c.blue_routes {
		if err = c.addMap(cliConnection, c.green, val); err != nil {
			return err
		}
	}
	return nil
}

func (c *SafeScaler) createRoute(cliConnection plugin.CliConnection) (Route, error) {
	temp_route := Route{
		domain: c.blue.routes[0].domain,
		host: "temp-" + c.blue.routes[0].host,
	}
	if _, err := cliConnection.CliCommand("create-route", c.space, temp_route.domain, "--hostname", temp_route.host);
	err != nil {
		return temp_route, errors.New("ERROR. Could not create a temporary route " + temp_route.domain + "." + temp_route.host + "\n")
	}
	return temp_route, nil
}

func (c *SafeScaler) addMap(cliConnection plugin.CliConnection, app *AppProp, route Route) error {
	if _, err := cliConnection.CliCommand("map-route", app.name, route.domain, "--hostname", route.host); err != nil {
		return errors.New("ERROR. Could not map " + route.domain + "." + route.host + " route to " + app.name + "\n")
	}
	app.routes = append(app.routes, route)
	return nil
}

func (c *SafeScaler) unmapping(cliConnection plugin.CliConnection) error {
	//unmap all routes from blue with exception of temp route for monitoring purposes
	for _, val := range c.blue_routes {
		if err := c.removeMap(cliConnection, c.blue, val, false); err != nil {
			return err
		}
	}
	//remove the first route that was pushed with new app
	if err := c.removeMap(cliConnection, c.green, c.green.routes[0], true); err != nil {
		return err
	}

	return nil
}

func (c *SafeScaler) removeMap(cliConnection plugin.CliConnection, app *AppProp, route Route, orphan bool) error {
	if _, err := cliConnection.CliCommand("unmap-route", app.name, route.domain, "--hostname", route.host);
	err != nil {
		return errors.New("ERROR. Could not unmap " + route.domain + "." + route.host + " route from " + app.name + "\n")
	}
	//updating app routes array
	for i, value := range app.routes {
		if value.host == route.host && value.domain == route.domain {
			new_routes := append(app.routes[:i], app.routes[i + 1:]...)
			app.routes = new_routes
			break
		}
	}
	if orphan == true {
		if err := c.deleteRoute(cliConnection, route); err != nil {
			return err
		}
	}
	return nil
}

func (c *SafeScaler) deleteRoute(cliConnection plugin.CliConnection, route Route) error {
	if _, err := cliConnection.CliCommand("delete-route", route.domain, "--hostname", route.host, "-f"); err != nil {
		return errors.New("ERROR. Could not delete " + route.domain + "." + route.host + " route from space\n")
	}
	return nil
}

func (c *SafeScaler) monitorTransactions(client *http.Client) error {
	//no endpoint so just regular blue green deployment
	if c.trans == "" {
		return nil
	}
	fmt.Println("Checking trans endpoint...")
	trans_endpoint := "https://" + c.blue.routes[0].host + "." + c.blue.routes[0].domain + c.trans
	base := time.Now() //baseline time to measure against
	current := time.Since(base).Seconds()
	//loop to continuously monitor transactions until it times out
	for current < float64(c.timeout) {
		result, err := client.Get(trans_endpoint)
		if err != nil {
			return err
		}
		//no content so there are no more transactions have parameter that has utc time
		if result.StatusCode == 204 {
			fmt.Println("No more pending transactions")
			return nil
		}
		if result.StatusCode != 200 {
			return errors.New("ERROR. Status code " + strconv.Itoa(result.StatusCode) + ". " + trans_endpoint + " endpoint is not okay. Check to make sure " + c.blue.name + " is healthy\n")
		}
		time.Sleep(3 * time.Second)
		current = time.Since(base).Seconds()
	}
	return errors.New("ERROR. The request timed out. " + trans_endpoint + " endpoint failed to provide HTTP Status Code 204. Can't safely shut down " + c.blue.name + "\n")
}

func (c *SafeScaler) powerDown(cliConnection plugin.CliConnection) error {
	if err := c.removeMap(cliConnection, c.blue, c.blue.routes[0], true); err != nil {
		return err
	}
	if _, err := cliConnection.CliCommand("stop", c.blue.name); err != nil {
		return errors.New("ERROR. Failed to stop " + c.blue.name + " from running\n")
	}
	c.blue.alive = false
	return nil
}

func main() {
	plugin.Start(new(SafeScaler))
}
