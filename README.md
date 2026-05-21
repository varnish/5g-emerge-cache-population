# 5g-emerge-cache-population
The 5G-Emerge Cache Population Component that tracks video title popularity and initiates satellite transmissions when content meets popularity requirements.

Run this component locally by running the `./bin/run.sh` script. Please configure the component using the environment variables mentioned below.

You can define these environment variables in a `.env` file that is picked up by the script.

Run `./bin/tui.sh` to get a TUI that returns the status of the Cache Population Component.

## Environment variables

* `QUERY_RANGE`: time range to evaluate content popularity (default value: `15s`)
* `SLEEP`: sleep interval in between runs (default value: `10s`)
* `TOP_ITEMS`: how many of the most popular titles are considered for satellite transmission (default value `10`)
* `MINIMUM_PREVIEW_WATCH_DURATION` minimum aggregated watch duration before a title is considered popular enough for previewing purposes (default value: `30s`)
* `UNPOPULARITY_THRESHOLD`: if the aggregated watch duration drops below this threshold, the title has become unpopular (default value: `10s`)
* `UNPOPULARITY_MAX_PROGRESS_PERCENTAGE`: if a title has become unpopular mid-transmission, only cancel it if its transmission progress is below this percentage (default value `70`)
* `UNPOPULARITY_MIN_AGE`: only cancel unpopular transmission that have a minimum age (default value `5m`)
* `PREVIEW_TRANSMISSION_SEGMENTS`: the number of video segments to transmit if a title qualifies for preview transmission (default value `10`)
* `MINIMUM_WATCH_DURATION`: minimum aggregated watch duration before a title is considered popular enough for a full transmission (default value: `60s`)
* `CACHE_EXPIRY`: the Time To Live of the internal cache where transmissions are stored (default value `5m`)
* `TITLES_FILE`: path of the JSON file that contains the titles that can be transmitted (default value: `titles.json`)
* `BREAKPOINT`: if set to `true` it breaks the loop that runs the logic and only runs the cache population component for one run (default value `false`)
* `FENIX_URL`: the URL endpoint of the Fenix API (default value `http://localhost:8888`)
* `FENIX_API_KEY`: the API key to authenticate with the Fenix API (default value is empty)
* `FENIX_MOCK_MODE`: whether or not to mock the Fenix API (default value `false`)
* `SIGNOZ_URL`: the URL endpoint of the Signoz API (default value `http://localhost:8080`)
* `SIGNOZ_API_KEY`: the API key to authenticate with `v4` of the Signoz API (default value is empty)
* `SIGNOZ_API_VERSION`: the version of the Signoz API to use, which is either `v4` or `v5 `(default value `v4`)
* `SIGNOZ_USERNAME`:  the username to authenticate with `v5` of the Signoz API (default value is empty)
* `SIGNOZ_PASSWORD`: the password to authenticate with `v5` of the Signoz API(default value is empty)
* `SIGNOZ_QUERY_SERVICE`: the OpenTelemetry service to query in Signoz (default value `varnish`)
* `LOG_LEVEL`: the log level of the cache population component (default value `info`)
* `API_SERVER_PORT` the port on wich the cache population component exposes its API (default value `8888`)