# Making a release

Before making a public release, we should test for a couple of days by pushing
to the [garage plugin repo](https://garage-cf-plugins.eu-gb.mybluemix.net/list).
This will be done automatically by Jenkins after step 2.

1. Run tests and acceptance tests

1. Edit the `PLUGIN_VERSION` variable in `.env`

1. Tag a new revision using [semver](http://semver.org): `git tag vX.X.X`

1. `git push --tags` (to github)

1. Create [a new github release](https://github.com/bluemixgaragelondon/cf-blue-green-deploy/releases/new)

1. Add brief description and attach all the binaries from the [garage plugin repo](https://garage-cf-plugins.eu-gb.mybluemix.net/list)

1. Follow the [instructions for submitting a plugin](https://github.com/cloudfoundry-incubator/cli-plugin-repo#submitting-plugins)
You need to update the following in `repo-index.yml` under `cf-blue-green-deploy`:

   * version
   * updated
   * url - this should be `https://github.com/bluemixgaragelondon/new_plugin/releases/download/vX.X.X/blue-green-deploy.PLATFORM`
   * sum - generate this using `shasum *` in the _artefacts_ directory
