package main
import (
	"github.com/cloudfoundry/cli/plugin"
	"fmt"
	"CLI-Hello/git-files/cf/errors"
	"net/http"
	"time"
	"flag"
)

type SafeScaler struct{
	blue	*AppProp
	green	*AppProp
	green_routes []Route
	blue_routes []Route
	trans string
	test string
	inst string
	original_name string
	space string
}
type AppProp struct {
	name	string
	routes 	[]Route
	alive	bool
	services []string
}
type Route struct {
	host 	string
	domain 	string
}
func (c *SafeScaler) Run(cliConnection plugin.CliConnection, args []string) {
	var err error
	if err = c.getArgs(args); err!=nil{
		fmt.Println(err)
		return
	}
	if c.blue, err = c.getApp(cliConnection, args[1]); err != nil{
		fmt.Println(err)
		return
	}
	if err = c.createNewApp(cliConnection); err!=nil{
		fmt.Println(err)
		return
	}
	if err = c.mapping(cliConnection); err != nil{
		fmt.Println(err)
		return
	}
	if healthy := c.healthTest(); !healthy {
		fmt.Println("new app is not healthy. Need to restart app")
		//c.restart
	}
	if err = c.unmapping(cliConnection); err!= nil{
		fmt.Println(err)
		return
	}
	if err = c.monitorTransactions(cliConnection); err!= nil{
		fmt.Println(err)
		c.restart(cliConnection)
		return
	}
	if err = c.powerDown(cliConnection); err!=nil{
		fmt.Println(err)
		return
	}



}

func (c *SafeScaler) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "SafeScaler",
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
				HelpText: "safely scales down your application using blue green deployment",
				UsageDetails: plugin.Usage{
					Usage: "safe-scale\n	cf safe-scale app_name [--inst] [--trans] [--test]",
					Options: map[string]string{
						"-inst":	"number of instances for new app",
						"-trans":	"endpoint to monitor transaction",
						"-test":	"endpoint to test if new app is healthy",
					},
				},
			},
		},
	}
}

func (c *SafeScaler) getApp(cliConnection plugin.CliConnection, name string)(*AppProp, error){ //get routes from app
	app, err := cliConnection.GetApp(name)
	if err!=nil{
		return nil, err
	}
	properties := &AppProp{
		name:	"",
		routes:	[]Route{},
		alive: 	true,
		services: []string{},
	}
	properties.name = app.Name
	for _, value := range app.Routes{ //get routes of app
		new_route:= Route{
			domain:	value.Domain.Name,
			host: 	value.Host,
		}
		properties.routes = append(properties.routes, new_route)
	}
	for _, value:= range app.Services{
		properties.services = append(properties.services, value.Name)
	}
	if err = c.getSpace(cliConnection); err!=nil{
		return nil, err
	}
	return properties, nil
}

func(c *SafeScaler) addMap(cliConnection plugin.CliConnection, app *AppProp, route Route)error{
	if _, err :=cliConnection.CliCommand("map-route",app.name, route.domain, "--hostname", route.host); err!=nil{
		return err
	}
	app.routes= append(app.routes, route)
	return nil
}
func(c *SafeScaler) removeMap(cliConnection plugin.CliConnection, app *AppProp, route Route)error{
	if _, err :=cliConnection.CliCommand("unmap-route", app.name, route.domain, "--hostname", route.host);
	err!=nil{return err}
	for i, value := range app.routes{
		if value.host == route.host && value.domain == route.domain{
			new_routes := append(app.routes[:i], app.routes[i+1:]...)
			app.routes=new_routes
			return nil
		}
	}
	return errors.New("could not find map to unmap")

}

func(c *SafeScaler) healthTest() bool{ //tests new apps health
	if c.test == ""{ //no test so just continue with deployment
		return true
	}
	result,err := http.Get("http://"+c.green.routes[0].host+"."+c.green.routes[0].domain+c.test) //test endpoint
	if result.StatusCode != 200 || err!=nil { //not ok or error so test failed
		return false
	}
	return true

}
func(c *SafeScaler) monitorTransactions(cliConnection plugin.CliConnection)error{
	trans_endpoint := "http://"+c.blue.routes[0].host+"."+c.blue.routes[0].domain+c.trans
	base := time.Now()
	current:= time.Since(base).Seconds()
	for current<120{
		time.Sleep(3*time.Second)
		result,err := http.Get(trans_endpoint)
		if err !=nil{
			return err
		}
		if result.StatusCode == 204{ //no content so there are no more transactions
			return nil
		}
		current= time.Since(base).Seconds()
	}
	return errors.New("The request timed out")
}


func(c *SafeScaler) renameApp(cliConnection plugin.CliConnection, app *AppProp, name string)error{
	if _, err:=cliConnection.CliCommand("rename", app.name, name); err!=nil{
		return err
	}
	app.name= name
	return nil
}

func(c *SafeScaler) restart(cliConnection plugin.CliConnection){ //restart has to map everything back from blue and remove blue routes from green
	fmt.Println("restarting both apps to original state")
	for _,val :=range c.blue_routes{
		c.addMap(cliConnection, c.blue, val)
		c.removeMap(cliConnection, c.green,val)
	}

}

func (c *SafeScaler) getArgs(args []string)error{
	if len(args) == 1{
		return errors.New("Did not specify an app")
	}

	if len(args) == 2 {
		c.original_name = args[1]
		return nil
	}
	f := flag.NewFlagSet("flag", flag.ExitOnError)
	f.String("inst", "1", "the number of instances for new app")
	f.String("trans", "", "endpoint path to monitor transactions")
	f.String("test", "", "optional path to test new app deployed")
	flags := []string{}
	visit := func(a *flag.Flag) {
		flags = append(flags, a.Value.String())
	}
	f.VisitAll(visit) //note visits all flags in alphabetic order
	f.Parse(args[2:]) //first two args are the command and the app name
	c.inst = flags[0]
	c.test = flags[1]
	c.trans = flags[2]
	return nil
}

func(c *SafeScaler) getSpace(cliConnection plugin.CliConnection) error{
	space, err :=cliConnection.GetCurrentSpace()
	if err !=nil{
		return err
	}
	c.space = space.Name
	return nil
}
func(c *SafeScaler) createNewApp(cliConnection plugin.CliConnection)error{
	if err := c.pushApp(cliConnection); err!= nil{
		return err
	}
	for _, val := range c.blue.services{
		if err := c.bindService(cliConnection, val); err!=nil{
			return err
		}
	}
	return nil
}
func(c *SafeScaler) pushApp(cliConnection plugin.CliConnection) error{
	new_name := "new"+c.original_name
	if _ , err := cliConnection.CliCommand("push", new_name, "-i",
		c.inst, "--hostname", new_name, "-d", c.blue.routes[0].domain); err!= nil{
		return err
	}
	c.green.name = new_name
	c.green.routes= append(c.green.routes, Route{host: new_name, domain: c.blue.routes[0].domain})
	return nil

}
func(c *SafeScaler) bindService(cliConnection plugin.CliConnection, val string) error{
	if _ , err := cliConnection.CliCommand("bind-service", c.green.name, val); err!=nil{
		return err
	}
	c.green.services = append(c.green.services, val)
	return nil
}

func(c *SafeScaler) mapping(cliConnection plugin.CliConnection) error{ //creates a temp route for old app and maps old routes to new app
	temp_route, err := c.createRoute(cliConnection)
	if err != nil{
		return err
	}
	if err = c.addMap(cliConnection, c.blue, temp_route); err != nil{//add temp route to old app
		return err
	}
	for _, val := range c.blue_routes{ //add all old routes to new app
		if err = c.addMap(cliConnection, c.green, val); err != nil{//add temp route to old app
			return err
		}
	}
	return nil
}

func(c *SafeScaler) unmapping(cliConnection plugin.CliConnection) error{
	for _, val := range c.blue.routes[:len(c.blue.routes)]{ //unmap all routes from blue with exception of temp route
		if err:= c.removeMap(cliConnection, c.blue, val); err!=nil{
			return err
		}
	}
	if err:= c.removeMap(cliConnection, c.green, c.green.routes[0]); err!=nil{ //remove the first route that was pushed with new app
		return err
	}

	return nil
}

func(c *SafeScaler) createRoute(cliConnection plugin.CliConnection) (Route, error){
	temp_route := Route{
		domain: c.blue.routes[0].domain,
		host: "temp"+c.blue.routes[0].host,
	}
	if _, err :=cliConnection.CliCommand("create-route", c.space, temp_route.domain, "--hostname", temp_route.host);
	err !=nil{
		return temp_route, err
	}
	return temp_route, nil
}
func(c *SafeScaler) powerDown(cliConnection plugin.CliConnection) error{
	if err:= c.removeMap(cliConnection, c.blue, c.blue.routes[0]); err!=nil{
		return err
	}
	if _,err :=cliConnection.CliCommand("stop", c.blue.name); err!=nil{
		return err
	}
	c.blue.alive = false
	return nil
}
func main() {
	plugin.Start(new(SafeScaler))
}

