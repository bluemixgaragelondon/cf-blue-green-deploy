# Making a release

Before making a public release, we should test for a couple of days by pushing
to the [garage plugin repo](https://garage-cf-plugins.eu-gb.mybluemix.net/list).
This will be done automatically by the [Bluemix DevOps Pipeline](https://console.ng.bluemix.net/devops/pipelines/4e5bb6ac-762d-42aa-abe1-71beabeafbb1) after step 2.

1. Check the output of the [latest build](https://console.ng.bluemix.net/devops/pipelines/4e5bb6ac-762d-42aa-abe1-71beabeafbb1) is green.

1. Edit the `PLUGIN_VERSION` variable in `.env`

1. Tag a new revision using [semver](http://semver.org): `git tag vX.X.X`

1. `git push --tags` (to github)

1. Create [a new github release](https://github.com/bluemixgaragelondon/cf-blue-green-deploy/releases/new)

1. Add brief description and attach all the binaries from the [garage plugin repo](https://garage-cf-plugins.eu-gb.mybluemix.net/list)

1. Follow the [instructions for submitting a plugin](https://github.com/cloudfoundry-incubator/cli-plugin-repo#submitting-plugins)
You need to update the following in `repo-index.yml` under `cf-blue-green-deploy`:

   * version
   * updated timestamp
   * url - this should be `https://github.com/bluemixgaragelondon/new_plugin/releases/download/vX.X.X/blue-green-deploy.PLATFORM`
   * sum - copy from [the garage staging repo](https://garage-cf-plugins.eu-gb.mybluemix.net/list) as this version will have passed all of the testing.

# Running the acceptance tests

You can run the acceptance tests on any cloud foundry installation by following these steps:

1. Edit `.env`:

   * Update the `CF_URL="api.eu-gb.bluemix.net"` to match your cloud foundry api url.

   * replace the values of `CF_USERNAME` and `CF_ORG` with your username and organization name (for a personal bluemix account this is typically your email address).

   * set the value of `CF_SPACE` to the name of a space in your org where the test should run. If it does not exist it will be created.

   *  set the value of `TEST_ACCEPTANCE_APP_NAME` and `TEST_ACCEPTANCE_APP_HOSTNAME` to any unique values that are valid for the test app domain (eg. eu-gb.mybluemix.net).

1. Source `.env` to your shell.

1. Edit `acceptance/app/manifest.yml`. It governs the example app that is pushed during the acceptance test.

   * Either remove the `hosts:` section, or provide at least one unique hostname.

   * Provide at least one domain. In the `domains:` section, use any domain that is available to your cloud foundry org/space, eg. `eu-gb.mybluemix.net`.

   * The remaining fields can be left unchanged.

1. Make sure that the version of `blue-green-deploy` that should be tested is currently installed.

1. Set the `CF_PASSWORD` variable in your shell. On an interactive shell, run `read -s CF_PASSWORD` and type in your password followed by return. Avoid using `export` with this field, as any sub-shell could then read your password.

1. To run the acceptance tests: `CF_PASSWORD=$CF_PASSWORD script/test_acceptance`.

1. If the tests passed, there should be a message similar to `ACCEPTANCE TESTS PASSED!` printed when the test has finished. The exit value is 0 for a successful test.
