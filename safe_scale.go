package main
import (
	"github.com/cloudfoundry/cli/plugin"
	"fmt"
	"CLI-Hello/git-files/cf/errors"
)

type SafeScaler struct{
	blue	*AppProp
	green	*AppProp
	green_routes []Route
}
type AppProp struct {
	name	string
	routes 	[]Route
	alive	bool
}
type Route struct {
	host 	string
	domain 	string
}
func (c *SafeScaler) Run(cliConnection plugin.CliConnection, args []string) {
	/*
	make new transactions only go to new app and close route
	check endpoint to see if the pending transactions are done
	Make sure that the new app is running smoothly
	 */

	var err error
	if c.blue, err = c.getApp(cliConnection, args[1]); err != nil{
		fmt.Println(err)

		return
	}
	if c.green, err = c.getApp(cliConnection, args[2]); err !=nil{
		fmt.Println(err)
		return
	}
	c.green_routes = c.green.routes //need to keep original track of original routes so we can delete them after
	for i, _ := range c.blue.routes{
		if bad:=c.addMap(cliConnection, c.blue.routes[i]); bad!=nil{
			fmt.Println(bad)
		}
	}

	for _, value := range c.blue.routes{	//unmap everything from the blue app
		if err :=c.removeMap(cliConnection, c.blue, value); err!=nil{
			fmt.Println(err)
		}
	}
	//need to add monitoring here!!!!
	/*Need method to check api health endpoint
	Need method to reset apps if it times out or fails
	check health of new app after. If they are not running, reset. have another endpoint to check if new app is healthy
	 */
	for _, ele := range c.green_routes{
		if err :=c.removeMap(cliConnection, c.green, ele); err!=nil{ //unmap original routes from green app
			fmt.Println(err)
		}
	}
	if err:= c.deleteApp(cliConnection, c.blue); err!=nil{ //delete blue app
		fmt.Println(err)
	}
	if err:= c.renameApp(cliConnection, c.green, c.blue.name); err!=nil{ //rename green app after blue app
		fmt.Println(err)
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
					Usage: "safe-scale\n	cf safe-scale old_app new_app",
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
	}
	properties.name = app.Name 
	for _, value := range app.Routes{
		new_route:= Route{
			domain:	value.Domain.Name,
			host: 	value.Host,
		}
		properties.routes = append(properties.routes, new_route)
	}
	return properties, nil
}

func(c *SafeScaler) addMap(cliConnection plugin.CliConnection, route Route)error{
	if _, err :=cliConnection.CliCommand("map-route", route.domain, "--hostname", route.host); err!=nil{
		return err
	}
	c.green.routes= append(c.green.routes, route)
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

func(c *SafeScaler) deleteApp(cliConnection plugin.CliConnection, app *AppProp)error{
	if _, err:=cliConnection.CliCommand("delete", app.name, "-f"); err!=nil{
		return err
	}
	app.alive = false
	return nil

}

func(c *SafeScaler) renameApp(cliConnection plugin.CliConnection, app *AppProp, name string)error{
	if _, err:=cliConnection.CliCommand("rename", app.name, name); err!=nil{
		return err
	}
	app.name= name
	return nil
}

func main() {
	plugin.Start(new(SafeScaler))
}
