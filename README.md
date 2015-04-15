# Blue/Green deployer plugin for CF

## Introduction

**cf-blue-green-deploy** is a plugin for the CF command line tool that automates a few steps involved in zero-downtime deploys

## Overview

The plugin takes care of the following steps packaged into one command:

* Pushes the current version of the app with a timestamp in its name
* Optionally runs smoke tests against the newly pushed app to verify the deployment
* Switches routes between the currently live app and the newly deployed app
* Cleans up old versions of the app


## How to use

* Get the plugin from our repository
```
cf add-plugin-repo garage https://garage-cf-plugins.eu-gb.mybluemix.net/
cf install-plugin blue-green-deploy -r garage
```

* Deploy your app
```
cd your_app_root
cf blue-green-deploy app_name
```

* Deploy with optional smoke tests
```
cf blue-green-deploy app_name --smoke-test <path to test script>
```

The only argument passed to the smoke test script is the FQDN of the newly pushed app. If the smoke test returns with a non-zero exit code the deploy process will stop and fail, the current live app will not be affected.

If the test script exits with a zero exit code, the plugin will remap all routes from the current live app to the new app. The plugin supports routes under custom domains.
