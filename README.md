# intervals-functions

functions to sync or backfill data from 3rd party sources to [intervals.icu](https://intervals.icu/) that aren't natively supported

# overview

## cronometer

there are two binaries to get cronometer data into intervals:
- `cmd/cronometer`
- `cmd/cronometer_backfill`

`cronometer` can be used as a one-off command or lambda function, that takes yesterday's macros from cronometer (kCal, protein, fat, carbs) and stores those values in intervals.

`cronometer_backfill` will extract daily macros from cronometer for a given date range and save those to intervals.

### usage

```bash
cd intervals-functions/
cp .env.example .env
# update .env with your credentials

# to get yesterday's macros from cronometer
cd cmd/cronometer
go run cronometer.go

# to get a date range of macros from cronometer
cd cmd/cronometer_backfill
go run backfill.go 2026-01-01 2026-03-01
```

## garmin

intervals provides a garmin sync natively, which will sync a lot of data from garmin from that point forward. garmin also provides access to your historical activity data, but not wellness data. if you want to backfill intervals with wellness data (sleep, weight, etc.), `garmin.go` is the way to go.

### usage

before backfilling any data, there is some required setup in intervals. the following wellness fields need to exist in order to have a destination for garmin data:
- `BodyBatteryMin`
- `BodyBatteryMax`
- `StressRestSeconds`
- `StressLowSeconds`
- `StressMediumSeconds`
- `StressHighSeconds`
- `SleepNeedMinutes`
- `SleepRemSeconds`
- `SleepDeepSeconds`
- `SleepLightSeconds`
- `SleepAwakeSeconds`

i have already created these fields and made them publicly available (search for these names or my username `pitcherella`). alternatively, you can create these fields however you want, but the values listed above *must* be the `Code` for the field. all of those values are integers, so i would also recommend a format of `0.f`. for the fields that are seconds or minutes, intervals lets you choose a conversion so that they're represented in hours/minutes/seconds if you wish.

`garmin.go` works by mimicing garmin connect web requests. there's no login implemented, so we'll need some data to make authenticated requests to garmin:
- log in to Garmin Connect
- open developer tools (in Chrome: View -> Developer -> Developer Tools)
- click on the Network tab, and select the `Fetch/XHR` tab
- in Garmin Connect, navigate to a Health Stats page, like Sleep. you should see a request pop up in Developer tools that looks like today's date (`2026-01-01`)
- right click on that date, `Copy` -> `Copy as cURL`
- relative to the `garmin.go` binary you will execute, `mkdir request`
- create `curl.txt` in that folder and paste your cURL response

Garmin sessions don't last forever, so you may need to repeat these steps periodically. Now you can run the backfill for your desired metrics and date ranges.

`garmin.go` supports the following metrics:
- **body battery:** low, high
- **respiration:** average sleep respiration
- **stress:** overall, high, medium, low, rest
- **sleep:** score, total sleep time, spO2, stages, resting HR, sleep need, sleep quality
- **weight:** latest weight

to run the backfill:
```
./bin/garmin -from=YYYY-MM-DD -to=YYYY-MM-DD -athleteId=XYZ -apiKey=ABC -dry-run bodybattery,stress,respiration,sleep,weight
```

- `from` the starting date of your backfill
- `to` the end date of your backfill
- `athleteId` your intervals.icu athlete ID
- `apiKey` your intervals.icu API key
- `dry-run` (optional) will print the results to the terminal and not send any data to intervals

# building

`make build`

for the release binary: `make release`

# testing

`make test`

# notes

- [gocronometer](https://github.com/jrmycanady/gocronometer) doesn't work with 2FA
- running `backfill` with large date ranges can lead to throttling from cronometer. i would recommend doing small date ranges and spreading them out over hours.
- `NETLIFY` as an environment variable is not available at runtime, only [these](https://docs.netlify.com/build/functions/environment-variables/#functions) are. `IS_NETLIFY` is the safe way to denote if `cronometer.go` is running in the netlify environment.
- netlify functions run in UTC time.
- garmin backfill works by leveraging the same requets the garmin connect web app uses. as such, this project is **for personal use only** and could break at any time.
- spO2 has no 4 week window in Garmin Connect, and is not supported for backfill.
- for weight: the garmin API and intervals are both in metric as far as I can tell. weight from garmin is reported in grams, and intervals is looking for the weight value in kilograms.