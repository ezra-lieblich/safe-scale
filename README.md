#Safe Scale Plugin

The CF CLI Plugin uses blue-green deployment methodology to achieve continuous deployment. In addition to the 
regular blue-green deployment, the plugin also monitors endpoints. The user can provide a test endpoint that 
checks if the newly pushed app is healthy. They can also provide a transaction monitoring endpoint that checks 
if the old app instances are still processing transactions. Customers were noticing that when they scale down 
their applications, CF would shut down instances that were still processing transactions and they would be lost. 
This plugin will get the endpoints provided and prevent the blue-green deployment from either mapping the routes 
to an unhealthy app or prevent an old app that still has ongoing transactions from shutting down.

#Assumptions

The plugin makes assumptions about the endpoints provided by the user. Since the plugin accesses status codes to 
determine the state of the endpoint, the endpoints must use these status codes in order for the plugin to read it 
properly. For the test endpoint, it must return status code 200 if the app is healthy. For the transaction endpoint, 
it must return status code 204 (no content) if the app has no more transactions and 200 if there are still transactions 
processing. The plugin only consumes “https://“ endpoints currently

# Requirements

The plugin requires you to be in the same directory as the app you are trying to blue-green deploy.

# Usage

cf safe-scale app_name --inst=int --trans=string --test=string --timeout=int

Flags
"-inst":        "number of instances for new app"	
"-trans":        "endpoint to monitor transaction"
"-test":        "endpoint to test if new app is healthy"
"-timeout":        "time in seconds to monitor transactions"

Note if you don’t provide an endpoint for monitoring transactions or checking health the plugin will just continue 
regular blue-green deployment
# Installation

go get https://github.com/ezra-lieblich/safe-scale
cf install-plugin safe_scale
